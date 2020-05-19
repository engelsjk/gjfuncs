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
	input   = kingpin.Flag("input", "input path").Default(".").Short('i').String()
	dupekey = kingpin.Flag("dupekey", "feature property key to remove duplicate values").Default("").Short('k').String()
	output  = kingpin.Flag("output", "output filepath").Default("").Short('o').String()
)

var (
	ErrorOpenInput                = errors.New("error: unable to open input (filepath)")
	ErrorInvalidInputDir          = errors.New("error: input dir does not exist or is not valid")
	ErrorInvalidFeatureCollection = errors.New("error: invalid geojson feature collection")
	ErrorJSONConversion           = errors.New("error: unable to convert json")
	ErrorSaveFile                 = errors.New("error: unable to save output file")
)

func main() {

	kingpin.Parse()

	if !gjfuncs.DirExists(*input) {
		log.Fatal(ErrorInvalidInputDir)
	}

	files, err := ioutil.ReadDir(*input)
	if err != nil {
		log.Fatal(err)
	}

	var outputFilePath string
	duplicates := make(map[string]bool)
	newCollection := geojson.NewFeatureCollection()

	numFeatures := 0
	for _, f := range files {

		filename := f.Name()

		if !gjfuncs.IsGeoJSONExt(filename) {
			continue
		}

		filePath := filepath.Join(*input, f.Name())

		b, err := gjfuncs.Open(filePath)
		if err != nil {
			log.Fatal(ErrorOpenInput)
		}

		fc, err := geojson.UnmarshalFeatureCollection(b)
		if err != nil {
			log.Fatal(ErrorInvalidFeatureCollection)
		}

		for _, f := range fc.Features {
			v, ok := f.Properties[*dupekey]
			if !ok {
				continue
			}
			if s, ok := v.(string); ok {
				if duplicates[s] {
					continue
				}
				duplicates[s] = true
			}

			newCollection.Append(f)
			numFeatures++
		}
	}

	b, err := json.MarshalIndent(newCollection, "", " ")
	if err != nil {
		log.Fatal(ErrorJSONConversion)
	}

	if *output != "" {
		outputFilePath = *output
	} else {
		outputFilePath = filepath.Join(".", "feature-collection.geojson")
	}

	err = ioutil.WriteFile(outputFilePath, b, 0644)
	if err != nil {
		log.Fatal(ErrorSaveFile)
	}
	fmt.Printf("saved %d features to %s\n", numFeatures, outputFilePath)
}
