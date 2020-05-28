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
	Verbose     bool
}

func Build(loader Loader, files []os.FileInfo, opts BuildOptions) error {

	if loader.Err != nil {
		return loader.Err
	}

	numWorkers := 15
	countFilesGeoJSON := 0

	outfile, err := os.OpenFile(loader.OutputFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("unable to create new output file %s", filepath.Base(loader.OutputFilePath))
	}
	defer outfile.Close()

	filesToProcess := files

	duplicates := make(map[string]bool)

	filename := make(chan string)
	last := make(chan bool)
	var wg sync.WaitGroup

	////

	logger := log.New(outfile, "", 0)

	if !opts.NDJSON {
		// save last file for the end to handle trailing "," in feature collection features array
		filesToProcess = files[:len(files)-1]
		logger.Output(2, `{"type":"FeatureCollection","features":[`)
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(filename <-chan string, last <-chan bool) {
			defer wg.Done()
			buildWorker(filename, duplicates, logger, opts)
		}(filename, last)
	}

	for _, fi := range filesToProcess {
		if !IsGeoJSONExt(fi.Name()) {
			continue
		}
		countFilesGeoJSON++
		inputFilePath := filepath.Join(loader.InputDir, fi.Name())
		filename <- inputFilePath
	}

	close(filename)
	wg.Wait()

	if !opts.NDJSON {
		// note: the last file is ran to handle trailing "," in feature collection features array
		lastFilePath := filepath.Join(loader.InputDir, files[len(files)-1].Name())
		opts.NDJSON = true
		buildProcess(lastFilePath, duplicates, logger, opts)
		logger.Output(2, `]}`)
	}

	return nil
}

func buildWorker(filename <-chan string, duplicates map[string]bool, logger *log.Logger, opts BuildOptions) {
	for fi := range filename {
		buildProcess(fi, duplicates, logger, opts)
	}
}

func buildProcess(filename string, duplicates map[string]bool, logger *log.Logger, opts BuildOptions) {

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

		line := string(b)
		if !opts.NDJSON {
			line = fmt.Sprintf("%s,", line)
		}
		logger.Output(2, line)
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

func isDuplicate(f *geojson.Feature, dupeKey string, duplicates map[string]bool) bool {
	if dupeKey == "" {
		return false
	}
	v, ok := f.Properties[dupeKey]
	if ok {
		if s, ok := v.(string); ok {
			if duplicates[s] {
				return true
			}
			duplicates[s] = true
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
