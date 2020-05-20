package gjbuilder

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
	ErrorReadDirFiles             = errors.New("error: unable to read directory files")
	ErrorSaveFile                 = errors.New("error: unable to save output file")
	WarningNoFiles                = errors.New("warning: no files found in input dir")
)

func main() {

	kingpin.Parse()

	if !gjfuncs.DirExists(*input) {
		fmt.Println(ErrorInvalidInputDir)
		return
	}

	if !gjfuncs.DirExists(filepath.Dir(*input)) {
		log.Fatal(ErrorInvalidOutputDir)
		return
	}

	files, err := ioutil.ReadDir(*input)
	if err != nil {
		fmt.Println(ErrorReadDirFiles)
		return
	}

	numFiles := len(files)
	if numFiles == 0 {
		fmt.Println(WarningNoFiles)
		return
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

		file, err := gjfuncs.GetFile(inputFilePath)
		if err != nil {
			fmt.Println(ErrorOpenInput)
			return
		}
		defer file.Close()

		b, err := gjfuncs.Open(file)
		if err != nil {
			fmt.Println(ErrorOpenInput)
			return
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
			fmt.Println(ErrorInvalidFeatureCollection)
			return
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
		fmt.Println(ErrorJSONConversion)
		return
	}

	outputFilePath := filepath.Join(".", "feature-collection.geojson")
	if *output != "" {
		outputFilePath = *output
	}

	err = ioutil.WriteFile(outputFilePath, b, 0644)
	if err != nil {
		fmt.Println(ErrorSaveFile) //
		return
	}
	fmt.Printf("saved %d features to %s\n", numFeatures, outputFilePath)
}
