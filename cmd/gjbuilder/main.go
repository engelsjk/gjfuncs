package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/engelsjk/gjfunks/gjfunks"
	"github.com/paulmach/orb/geojson"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	input     = kingpin.Arg("input", "input path").Default(".").String()
	output    = kingpin.Flag("output", "output filepath").Default("").Short('o').String()
	dupekey   = kingpin.Flag("dupekey", "feature property key to remove duplicate values").Default("").Short('k').String()
	ndjson    = kingpin.Flag("ndjson", "output as newline-delimited json").Default("false").Short('n').Bool()
	overwrite = kingpin.Flag("overwrite", "overwrite existing output file").Default("false").Short('f').Bool()
)

var (
	WarningInputEmpty             = errors.New("warning: input is empty")
	WarningOutputAlreadyExists    = errors.New("warning: output file already exists")
	ErrorOpenInput                = errors.New("error: unable to open input (filepath)")
	ErrorRemoveOutput             = errors.New("error: unable to remove output file")
	ErrorInvalidInputDir          = errors.New("error: input dir does not exist or is not valid")
	ErrorInvalidOutputDir         = errors.New("error: output dir does not exist or is not valid")
	ErrorInvalidFeatureCollection = errors.New("error: invalid geojson feature collection")
	ErrorJSONConversion           = errors.New("error: unable to convert json")
	ErrorReadDirFiles             = errors.New("error: unable to read directory files")
	ErrorSaveFile                 = errors.New("error: unable to save output file")
	ErrorWriteFile                = errors.New("error: unable to write to output file")
	ErrorCloseFile                = errors.New("error: unable to close input file")
	WarningNoFiles                = errors.New("warning: no files found in input dir")
)

func main() {

	kingpin.Parse()

	if *input == "" {
		fmt.Println(WarningInputEmpty)
		return
	}

	if !gjfunks.DirExists(*input) {
		fmt.Println(ErrorInvalidInputDir)
		return
	}

	if !gjfunks.DirExists(filepath.Dir(*output)) {
		fmt.Println(ErrorInvalidOutputDir)
		return
	}

	outputFilePath := filepath.Join(".", "features.ndjson")
	if *output != "" {
		outputFilePath = *output
	}

	if gjfunks.FileExists(outputFilePath) && !*overwrite {
		fmt.Println(WarningOutputAlreadyExists)
		return
	}

	err := os.Remove(outputFilePath)
	if err != nil {
		fmt.Println(ErrorRemoveOutput)
		return
	}

	////////////////

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

	fmt.Printf("processing %d files:\n", numFiles)

	////////////////////

	file, err := os.OpenFile(outputFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	duplicates := make(map[string]bool)

	filenames := make(chan string)
	var wg sync.WaitGroup

	////

	logger := log.New(file, "", 0)

	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func(filenames <-chan string) {
			defer wg.Done()
			worker(filenames, duplicates, logger)
		}(filenames)
	}

	for _, fi := range files {
		filename := fi.Name()
		if !gjfunks.IsGeoJSONExt(filename) {
			continue
		}
		inputFilePath := filepath.Join(*input, fi.Name())
		filenames <- inputFilePath
	}

	close(filenames)
	wg.Wait()
}

func worker(filenames <-chan string, duplicates map[string]bool, logger *log.Logger) {
	for filename := range filenames {
		processNDJSON(filename, duplicates, logger)
	}
}

func processNDJSON(filename string, duplicates map[string]bool, logger *log.Logger) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("%s:%s\n", ErrorOpenInput, err.Error())
		return
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("%s:%s\n", ErrorOpenInput, err.Error())
		return
	}

	// var isDup bool
	var fs []*geojson.Feature

	// if feature...
	f, err := geojson.UnmarshalFeature(b)
	if err == nil {
		fs = append(fs, f)
	} else {
		// if feature collection...
		fc, err := geojson.UnmarshalFeatureCollection(b)
		if err != nil {
			fmt.Printf("%s:%s\n", ErrorInvalidFeatureCollection, filename)
			return
		}
		fs = fc.Features
	}

	for _, f := range fs {
		// if isDup = mapDuplicate(f, *dupekey, duplicates); isDup {
		// 	continue
		// }
		b, err := json.Marshal(f)
		if err != nil {
			fmt.Printf("%s\n", ErrorJSONConversion)
			return
		}
		logger.Output(2, string(b))
	}
}

func mapDuplicate(f *geojson.Feature, dupekey string, duplicates map[string]bool) bool {
	if dupekey != "" {
		v, ok := f.Properties[dupekey]
		if ok {
			if s, ok := v.(string); ok {
				if duplicates[s] {
					return true
				}
				duplicates[s] = true
			}
		}
	}
	return false
}

////

func processAsFeatureCollection(fileInfos []os.FileInfo, output string) (int, error) {

	numFiles := len(fileInfos)
	newCollection := geojson.NewFeatureCollection()
	numFeatures := 0

	duplicates := make(map[string]bool)
	var isDup bool

	const numJobs = 5
	var wg sync.WaitGroup
	var m sync.Mutex
	wg.Add(numFiles)

	for _, fi := range fileInfos {

		go func(fi os.FileInfo) {
			filename := fi.Name()
			fmt.Printf("%s\n", filename)

			if !gjfunks.IsGeoJSONExt(filename) {
				return
			}

			inputFilePath := filepath.Join(*input, fi.Name())

			file, err := gjfunks.GetFile(inputFilePath)
			if err != nil {
				fmt.Printf("%s:%s\n", ErrorOpenInput, inputFilePath)
				return
			}

			b, err := gjfunks.Open(file)
			if err != nil {
				fmt.Printf("%s:%s\n", ErrorOpenInput, inputFilePath)
				return
			}

			// if feature...
			f, err := geojson.UnmarshalFeature(b)
			if err == nil {
				m.Lock()
				if isDup = mapDuplicate(f, *dupekey, duplicates); isDup {
					fmt.Println("is duplicate")
					return
				}
				m.Unlock()
				newCollection.Append(f)
				numFeatures++
				return
			}

			// if feature collection...
			fc, err := geojson.UnmarshalFeatureCollection(b)
			if err != nil {
				log.Fatal(ErrorInvalidFeatureCollection)
			}

			for _, f := range fc.Features {
				m.Lock()
				if isDup = mapDuplicate(f, *dupekey, duplicates); isDup {
					continue
				}
				m.Unlock()
				newCollection.Append(f)
				numFeatures++
			}

			err = file.Close()
			if err != nil {
				log.Fatal(ErrorCloseFile)
			}

			wg.Done()
		}(fi)
	}

	wg.Wait()

	b, err := json.MarshalIndent(newCollection, "", " ")
	if err != nil {
		return numFeatures, ErrorJSONConversion
	}

	err = ioutil.WriteFile(output, b, 0644)
	if err != nil {
		return numFeatures, ErrorSaveFile
	}

	return numFeatures, nil
}
