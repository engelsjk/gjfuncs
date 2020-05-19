package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/engelsjk/gjfuncs/gjfuncs"
	"github.com/paulmach/orb/geojson"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	input   = kingpin.Arg("input", "input path").Default(".").String()
	dupekey = kingpin.Flag("dupekey", "feature property key to remove duplicate values").Default("").Short('k').String()
	output  = kingpin.Flag("output", "output filepath").Default("").Short('o').String()
)

var (
	ErrorOpenInput                = errors.New("error: unable to open input (filepath)")
	ErrorInvalidInputDir          = errors.New("error: input dir does not exist or is not valid")
	ErrorInvalidOutputDir         = errors.New("error: output dir does not exist or is not valid")
	ErrorInvalidFeatureCollection = errors.New("error: invalid geojson feature collection")
	ErrorJSONConversion           = errors.New("error: unable to convert json")
	ErrorSaveFile                 = errors.New("error: unable to save output file")
	WarningNoFiles                = errors.New("warning: no files found in input dir")
)

func main() {

	kingpin.Parse()

	if !gjfuncs.DirExists(*input) {
		log.Fatal(ErrorInvalidInputDir)
	}

	if !gjfuncs.DirExists(filepath.Dir(*input)) {
		log.Fatal(ErrorInvalidOutputDir)
	}

	files, err := ioutil.ReadDir(*input)
	if err != nil {
		log.Fatal(err)
	}

	numFiles := len(files)
	if numFiles == 0 {
		log.Fatal(WarningNoFiles)
	}

	duplicates := make(map[string]bool)
	newCollection := geojson.NewFeatureCollection()

	numFeatures := 0
	for _, f := range files {

		filename := f.Name()

		if !gjfuncs.IsGeoJSONExt(filename) {
			continue
		}

		inputFilePath := filepath.Join(*input, f.Name())

		b, err := gjfuncs.Open(inputFilePath)
		if err != nil {
			log.Fatal(ErrorOpenInput)
		}

		// if feature...
		f, err := geojson.UnmarshalFeature(b)
		if err == nil {
			if *dupekey != "" {
				v, ok := f.Properties[*dupekey]
				if ok {
					if s, ok := v.(string); ok {
						if duplicates[s] {
							continue
						}
						duplicates[s] = true
					}
				}
			}
			newCollection.Append(f)
			numFeatures++
			continue
		}

		// if feature collection...
		fc, err := geojson.UnmarshalFeatureCollection(b)
		if err != nil {
			log.Fatal(ErrorInvalidFeatureCollection)
		}

		for _, f := range fc.Features {
			if *dupekey != "" {
				v, ok := f.Properties[*dupekey]
				if ok {
					if s, ok := v.(string); ok {
						if duplicates[s] {
							continue
						}
						duplicates[s] = true
					}
				}
			}
			newCollection.Append(f)
			numFeatures++
		}
	}

	b, err := json.MarshalIndent(newCollection, "", " ")
	if err != nil {
		log.Fatal(ErrorJSONConversion)
	}

	outputFilePath := filepath.Join(".", "feature-collection.geojson")
	if *output != "" {
		outputFilePath = *output
	}

	err = ioutil.WriteFile(outputFilePath, b, 0644)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("saved %d features to %s\n", numFeatures, outputFilePath)
}
