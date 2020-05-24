package main

import (
	"encoding/json"
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
	input     = kingpin.Arg("input", "input path").Default("").String()
	output    = kingpin.Flag("output", "output filepath").Default("").Short('o').String()
	dupekey   = kingpin.Flag("dupekey", "feature property key to remove duplicate values").Default("").Short('k').String()
	ndjson    = kingpin.Flag("ndjson", "output as newline-delimited json").Default("false").Short('n').Bool()
	overwrite = kingpin.Flag("overwrite", "overwrite existing output file").Default("false").Short('f').Bool()
)

var (
	ErrorInvalidInputPath  = "gjbuilder: Input path does not exist or is not valid"
	ErrorInvalidOutputPath = "gjbuilder: Output path does not exist or is not valid"

	WarningOutputAlreadyExists = "gjbuilder: File '%s' already exists. You can use --overwrite if you want to delete the old file.\n"
	ErrorOpenInput             = "gjbuilder: Unable to open input file %s\n"
	ErrorRemoveOutput          = "gjbuilder: Unable to remove output file\n"

	ErrorInvalidFeatureCollection = "gjbuilder: Invalid geojson feature collection in file %s\n"
	ErrorJSONConversion           = "gjbuilder: Unable to convert json\n"
	ErrorReadDirFiles             = "gjbuilder: Unable to read directory files\n"
	ErrorSaveFile                 = "gjbuilder: Unable to save output file\n"
	ErrorWriteFile                = "gjbuilder: Unable to write to output file\n"
	ErrorCloseFile                = "gjbuilder: Unable to close input file\n"
	WarningNoFiles                = "gjbuilder: No files found in input dir\n"
)

const (
	defaultFilename = "gjfeatures"
)

const (
	banner = `
╋╋╋╋╋┏┓╋╋╋╋╋┏┓╋╋┏┓
╋╋╋╋┏┫┃╋╋╋╋╋┃┃╋╋┃┃
┏━━┓┗┫┗━┳┓┏┳┫┃┏━┛┣━━┳━┓
┃┏┓┃┏┫┏┓┃┃┃┣┫┃┃┏┓┃┃━┫┏┛
┃┗┛┃┃┃┗┛┃┗┛┃┃┗┫┗┛┃┃━┫┃
┗━┓┃┃┣━━┻━━┻┻━┻━━┻━━┻┛
┏━┛┣┛┃
┗━━┻━┛
harder.faster.better.stronger...geojson

try "gjbuilder --help" to get started
\n`
)

func main() {

	kingpin.Parse()

	inputPath := *input
	outputFilePath := *output
	dupeKey := *dupekey
	isND := *ndjson
	isOverwrite := *overwrite

	/////////////////////////////

	var outputFilename string
	if outputFilePath != "" {
		outputFilename = filepath.Base(outputFilePath)
	} else {
		outputFilename = fmt.Sprintf("%s.geojson", defaultFilename)
		if isND {
			outputFilename = fmt.Sprintf("%s.ndjson", defaultFilename)
		}
		outputFilePath = filepath.Join(".", outputFilename)
	}

	/////////////////////////////

	if inputPath == "" {
		fmt.Printf(banner)
		return
	}

	if !gjfunks.DirExists(inputPath) {
		fmt.Printf(ErrorInvalidInputPath)
		return
	}

	if !gjfunks.DirExists(filepath.Dir(*output)) {
		fmt.Printf(ErrorInvalidOutputPath)
		return
	}

	if gjfunks.FileExists(outputFilePath) && !isOverwrite {
		fmt.Printf(WarningOutputAlreadyExists, outputFilename)
		return
	}

	if gjfunks.FileExists(outputFilePath) && isOverwrite {
		err := os.Remove(outputFilePath)
		if err != nil {
			fmt.Printf(ErrorRemoveOutput)
			return
		}
	}

	/////////////////////////////

	files, err := ioutil.ReadDir(inputPath)
	if err != nil {
		fmt.Printf(ErrorReadDirFiles)
		return
	}

	numFiles := len(files)
	if numFiles == 0 {
		fmt.Printf(WarningNoFiles)
		return
	}

	////////////////////

	file, err := os.OpenFile(outputFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	duplicates := make(map[string]bool)

	filenames := make(chan string)
	last := make(chan bool)
	var wg sync.WaitGroup

	////

	logger := log.New(file, "", 0)

	if !isND {
		logger.Output(2, `{"type":"FeatureCollection","features":[`)
	}

	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func(filenames <-chan string, last <-chan bool) {
			defer wg.Done()
			worker(filenames, duplicates, dupeKey, logger, isND)
		}(filenames, last)
	}

	for _, fi := range files[:len(files)-1] {
		filename := fi.Name()
		if !gjfunks.IsGeoJSONExt(filename) {
			continue
		}
		inputFilePath := filepath.Join(inputPath, fi.Name())
		filenames <- inputFilePath
	}

	close(filenames)
	wg.Wait()

	lastFilePath := filepath.Join(inputPath, files[len(files)-1].Name())
	process(lastFilePath, duplicates, dupeKey, logger, true)

	if !isND {
		logger.Output(2, `]}`)
	}

}

func worker(filenames <-chan string, duplicates map[string]bool, dupeKey string, logger *log.Logger, isND bool) {
	for filename := range filenames {
		process(filename, duplicates, dupeKey, logger, isND)
	}
}

func process(filename string, duplicates map[string]bool, dupeKey string, logger *log.Logger, isND bool) {

	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf(ErrorOpenInput, filepath.Base(filename))
		return
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf(ErrorOpenInput, filepath.Base(filename))
		return
	}

	var fs []*geojson.Feature

	f, err := geojson.UnmarshalFeature(b)
	if err == nil {
		fs = append(fs, f)
	} else {
		// if feature collection...
		fc, err := geojson.UnmarshalFeatureCollection(b)
		if err != nil {
			fmt.Printf(ErrorInvalidFeatureCollection, filepath.Base(filename))
			return
		}
		fs = fc.Features
	}

	var isDup bool
	for _, f := range fs {
		if isDup = mapDuplicate(f, dupeKey, duplicates); isDup {
			continue
		}

		// ToDo: feature flag options
		// 1. reverse ring orientation
		// ...(check if polygon or if multipolygon)
		// 2. limit coordinate decimal precision
		// 3. keep/filter properties
		// ?. ???

		b, err := json.Marshal(f)
		if err != nil {
			fmt.Printf(ErrorJSONConversion)
			continue
		}
		line := string(b)
		if !isND {
			line = fmt.Sprintf("%s,", line)
		}
		logger.Output(2, line)
	}
}

func mapDuplicate(f *geojson.Feature, dupeKey string, duplicates map[string]bool) bool {
	if dupeKey != "" {
		v, ok := f.Properties[dupeKey]
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
