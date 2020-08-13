package gjfunks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"sync"

	"github.com/paulmach/orb/geojson"
)

type SplitOptions struct {
	InputFilePath string
	OutputDir     string
	OutKey        string
	OutPrefix     string
	FlatFile      bool
	KeepOnlyKey   string
	FixToSpec     bool
	StdOut        bool
	DryRun        bool
}

type FeatureAndID struct {
	Feature *geojson.Feature
	ID      string
}

func Split(loader Loader, input []byte, opts SplitOptions) error {
	if loader.Err != nil {
		return loader.Err
	}

	numWorkers := 15

	fc, err := geojson.UnmarshalFeatureCollection(input)
	if err != nil {
		fmt.Printf(err.Error())
		return fmt.Errorf("input is an invalid feature collection")
	}

	numFeatures := len(fc.Features)
	width := 1 + int(math.Log10(float64(numFeatures)))

	fid := make(chan *FeatureAndID)

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(fid <-chan *FeatureAndID) {
			defer wg.Done()
			splitWorker(fid, opts)
		}(fid)
	}

	var counter int64 = 0
	for _, f := range fc.Features {
		counter++
		id := fmt.Sprintf("%0*d", width, counter)
		fid <- &FeatureAndID{
			Feature: f,
			ID:      id,
		}
	}

	close(fid)
	wg.Wait()

	return nil
}

func splitWorker(fid <-chan *FeatureAndID, opts SplitOptions) {
	for f := range fid {
		splitProcess(f, opts)
	}
}

func splitProcess(fid *FeatureAndID, opts SplitOptions) {

	filename := makeFilename(fid, opts)
	outputFilePath := makeFilepath(fid, opts)

	if opts.KeepOnlyKey != "" {
		RemoveAllPropertiesExcept(fid.Feature, opts.KeepOnlyKey)
	}

	if opts.FixToSpec {
		FixPolygons(fid.Feature)
	}

	if opts.StdOut {
		b, err := fid.Feature.MarshalJSON()
		if err != nil {
			fmt.Printf("unable to convert feature to geojson")
			return
		}
		fmt.Println(string(b))
		return
	}

	if opts.DryRun {
		b, err := fid.Feature.MarshalJSON()
		if err != nil {
			fmt.Printf("unable to convert feature to geojson")
			return
		}
		fmt.Println(string(b))
		return
	}

	var b []byte
	var err error
	if opts.FlatFile {
		b, err = json.Marshal(fid.Feature)
	} else {
		b, err = json.MarshalIndent(fid.Feature, "", " ")
	}
	if err != nil {
		fmt.Printf("unable to convert feature to geojson")
		return
	}

	err = ioutil.WriteFile(outputFilePath, b, 0644)
	if err != nil {
		fmt.Printf("unable to save file %s", filename)
		return
	}
}

func makeFilename(fid *FeatureAndID, opts SplitOptions) string {
	if opts.OutKey != "" {
		v, ok := fid.Feature.Properties[opts.OutKey]
		if ok {
			if s, ok := v.(string); ok {
				return FmtFilename("", s, "")
			}
		}
	}
	if opts.OutPrefix != "" {
		return FmtFilename("", opts.OutPrefix, fid.ID)
	}
	if opts.InputFilePath != "" {
		return FmtFilename(opts.InputFilePath, "", fid.ID)
	}
	return FmtFilename("", "feature", fid.ID)
}

func makeFilepath(fid *FeatureAndID, opts SplitOptions) string {
	filename := makeFilename(fid, opts)
	if opts.OutputDir != "" {
		return filepath.Join(opts.OutputDir, filename)
	}
	return filename
}
