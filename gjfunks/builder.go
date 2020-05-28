package gjfunks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

type BuildOptions struct {
	FilterKey   string
	KeepOnlyKey string
	NDJSON      bool
	FixToSpec   bool
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

	var duplicates sync.Map

	filename := make(chan string)
	var wg sync.WaitGroup

	////

	logger := log.New(outfile, "", 0)
	newFC := geojson.NewFeatureCollection()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(filename <-chan string) {
			defer wg.Done()
			buildWorker(filename, newFC, logger, opts, duplicates)
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

func buildWorker(filename <-chan string, newFC *geojson.FeatureCollection, logger *log.Logger, opts BuildOptions, duplicates sync.Map) {
	for fi := range filename {
		buildProcess(fi, newFC, logger, opts, duplicates)
	}
}

func buildProcess(filename string, newFC *geojson.FeatureCollection, logger *log.Logger, opts BuildOptions, duplicates sync.Map) {

	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("unable to open input file %s...skipping it\n", filepath.Base(filename))
		return
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("unable to read input file %s...skipping it\n", filepath.Base(filename))
		return
	}

	var fs []*geojson.Feature

	f, err := geojson.UnmarshalFeature(b)
	if err == nil {
		fs = append(fs, f)
	} else {
		fc, err := geojson.UnmarshalFeatureCollection(b)
		if err != nil {
			fmt.Printf("invalid geojson feature or collection in input file %s...skipping it\n", filepath.Base(filename))
			return
		}
		fs = fc.Features
	}

	numFeatures := len(fs)
	badFeatures := 0
	duplicateFeatures := 0

	for _, f := range fs {
		if isDuplicate(f, opts.FilterKey, duplicates) {
			duplicateFeatures++
			continue
		}

		if opts.KeepOnlyKey != "" {
			removeAllPropertiesExcept(f, opts.KeepOnlyKey)
		}

		if opts.FixToSpec {
			FixPolygons(f)
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

		newFC.Append(f)
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
}

func isDuplicate(f *geojson.Feature, dupeKey string, duplicates sync.Map) bool {
	if dupeKey == "" {
		return false
	}
	v, ok := f.Properties[dupeKey]
	if ok {
		if s, ok := v.(string); ok {
			if _, ok := duplicates.Load(s); ok {
				return true
			}
			duplicates.Store(s, true)
		}
	}
	return false
}

func removeAllPropertiesExcept(f *geojson.Feature, keepOnlyKey string) {
	for k := range f.Properties {
		if k != keepOnlyKey {
			delete(f.Properties, k)
		}
	}
}

func fixPolygon(p orb.Polygon) {
	// fix ring orientation to rfc7946 s3.1.6

	// outer ring must be counter-clockwise
	if p[0].Orientation() == -1 {
		p[0].Reverse()
	}

	// inner rings (aka holes) must be clockwise
	for _, pi := range p[1:] {
		if pi.Orientation() == 1 {
			pi.Reverse()
		}
	}
}
