package gjfunks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/paulmach/orb/geojson"
)

type BuildOptions struct {
	FilterKey         string
	KeepOnlyKey       string
	NDJSON            bool
	FixToSpec         bool
	SplitMultiPolygon bool
	Duplicates        *sync.Map
}

func Build(loader Loader, files []os.FileInfo, opts BuildOptions) error {

	if loader.Err != nil {
		return loader.Err
	}

	numWorkers := 15

	outfile, err := os.OpenFile(loader.OutputFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to create new output file %s", filepath.Base(loader.OutputFilePath))
	}
	defer outfile.Close()

	filename := make(chan string)
	var wg sync.WaitGroup

	opts.Duplicates = &sync.Map{}

	mu := &sync.Mutex{}

	////

	logger := log.New(outfile, "", 0)
	newFC := geojson.NewFeatureCollection()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(filename <-chan string) {
			defer wg.Done()
			buildWorker(filename, mu, newFC, logger, opts)
		}(filename)
	}

	for _, fi := range files {
		if !IsGeoJSONExt(fi.Name()) {
			continue
		}
		inputFilePath := filepath.Join(loader.InputDir, fi.Name())
		filename <- inputFilePath
	}

	close(filename)
	wg.Wait()

	if !opts.NDJSON {
		b, err := json.MarshalIndent(newFC, "", " ")
		if err != nil {
			return err
		}
		_, err = outfile.Write(b)
		if err != nil {
			return err
		}
	}

	return nil
}

func buildWorker(filename <-chan string, mu *sync.Mutex, newFC *geojson.FeatureCollection, logger *log.Logger, opts BuildOptions) {
	for fi := range filename {
		fc := buildProcess(fi, newFC, logger, opts)
		for _, f := range fc.Features {
			mu.Lock()
			newFC.Append(f)
			mu.Unlock()
		}
	}
}

func buildProcess(filename string, newFC *geojson.FeatureCollection, logger *log.Logger, opts BuildOptions) *geojson.FeatureCollection {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("unable to open input file %s...skipping it\n", filepath.Base(filename))
		return nil
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("unable to read input file %s...skipping it\n", filepath.Base(filename))
		return nil
	}

	var fs []*geojson.Feature
	fc := geojson.NewFeatureCollection()

	f, err := geojson.UnmarshalFeature(b)
	if err == nil {
		fs = append(fs, f)
	} else {
		fct, err := geojson.UnmarshalFeatureCollection(b)
		if err != nil {
			fmt.Printf("invalid geojson feature or collection in input file %s...skipping it\n", filepath.Base(filename))
			return nil
		}
		fs = fct.Features
	}

	numFeatures := len(fs)
	badFeatures := 0
	duplicateFeatures := 0

	for _, f := range fs {
		if IsDuplicate(f, opts.FilterKey, opts.Duplicates) {
			duplicateFeatures++
			continue
		}

		if opts.FixToSpec {
			FixPolygons(f)
		}

		if opts.KeepOnlyKey != "" {
			RemoveAllPropertiesExcept(f, opts.KeepOnlyKey)
		}

		if opts.SplitMultiPolygon {
			tmpFs := SplitMultiPolygon(f)
			for _, tmpF := range tmpFs {
				b, err := json.Marshal(tmpF)
				if err != nil {
					badFeatures++
					continue
				}
				if opts.NDJSON {
					logger.Output(2, string(b))
					continue
				}
				fc.Append(f)
			}
			continue
		}

		b, err := json.Marshal(f)
		if err != nil {
			badFeatures++
			continue
		}

		if opts.NDJSON {
			logger.Output(2, string(b))
			continue
		}
		fc.Append(f)
	}

	if badFeatures+duplicateFeatures > 0 {
		fstr := "feature"
		if numFeatures > 1 {
			fstr = "features"
		}
		fmt.Printf("skipped %d of %d %s in input file %s (%d duplicate, %d bad conversion)\n",
			badFeatures+duplicateFeatures, numFeatures, fstr, filepath.Base(filename), duplicateFeatures, badFeatures,
		)
	}

	return fc
}

func IsDuplicate(f *geojson.Feature, dupeKey string, duplicates *sync.Map) bool {
	if dupeKey == "" {
		return false
	}
	v, ok := f.Properties[dupeKey]
	if ok {
		if _, ok := duplicates.Load(v); ok {
			return true
		}
		duplicates.Store(v, true)
	}
	return false
}
