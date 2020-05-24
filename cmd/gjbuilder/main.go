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

// ToDo: add some flag options for handling features
// 1. fix ring orientation for polygons/multipolygons to match spec
// 2. limit coordinate decimal precision
// 3. keep or remove properties by tag
// ?. ???

const (
	name   = "gjbuilder"
	banner = `
╋╋╋╋╋┏┓╋╋╋╋╋┏┓╋╋┏┓
╋╋╋╋┏┫┃╋╋╋╋╋┃┃╋╋┃┃
┏━━┓┗┫┗━┳┓┏┳┫┃┏━┛┣━━┳━┓
┃┏┓┃┏┫┏┓┃┃┃┣┫┃┃┏┓┃┃━┫┏┛
┃┗┛┃┃┃┗┛┃┗┛┃┃┗┫┗┛┃┃━┫┃
┗━┓┃┃┣━━┻━━┻┻━┻━━┻━━┻┛
┏━┛┣┛┃
┗━━┻━┛
building one file from many .geojson files
try "gjbuilder --help" to get started
`
	defaultFilename = "gjfeatures"
)

var (
	input     = kingpin.Arg("input", "input path").Default("").String()
	output    = kingpin.Flag("output", "output filepath").Default("").Short('o').String()
	dupekey   = kingpin.Flag("dupekey", "feature property key to remove duplicate values").Default("").Short('k').String()
	ndjson    = kingpin.Flag("ndjson", "output as newline-delimited json").Default("false").Short('n').Bool()
	overwrite = kingpin.Flag("overwrite", "overwrite existing output file").Default("false").Short('f').Bool()
)

func main() {

	kingpin.Parse()

	/////////////////////////////

	/////////////////////////////

	if *input == "" {
		fmt.Printf(banner)
		return
	}

	loader := Loader{
		InputPath:      *input,
		OutputFilePath: *output,
		Overwrite:      *overwrite,
	}

	loader.Input()
	loader.Output(*ndjson)

	files := loader.Files()

	if err := run(loader, files, *ndjson); err != nil {
		fmt.Println(err.Error())
		return
	}

}

func run(loader Loader, files []os.FileInfo, isNewlineDelimited bool) error {

	if loader.Err != nil {
		return loader.Err
	}

	outfile, err := os.OpenFile(loader.OutputFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("%s: unable to create a new output file %s", name, filepath.Base(loader.OutputFilePath))
	}
	defer outfile.Close()

	filesToProcess := files
	duplicates := make(map[string]bool)

	filename := make(chan string)
	last := make(chan bool)
	var wg sync.WaitGroup

	////

	logger := log.New(outfile, "", 0)

	if !isNewlineDelimited {
		// save last file for the end to handle trailing "," in feature collection features array
		filesToProcess = files[:len(files)-1]

		logger.Output(2, `{"type":"FeatureCollection","features":[`)
	}

	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func(filename <-chan string, last <-chan bool) {
			defer wg.Done()
			worker(filename, duplicates, *dupekey, logger, *ndjson)
		}(filename, last)
	}

	for _, fi := range filesToProcess {
		if !gjfunks.IsGeoJSONExt(fi.Name()) {
			continue
		}
		inputFilePath := filepath.Join(loader.InputPath, fi.Name())
		filename <- inputFilePath
	}

	close(filename)
	wg.Wait()

	if !isNewlineDelimited {
		// run last file to handle trailing "," in feature collection features array
		lastFilePath := filepath.Join(loader.InputPath, files[len(files)-1].Name())
		process(lastFilePath, duplicates, *dupekey, logger, true)

		logger.Output(2, `]}`)
	}

	return nil
}

func worker(filename <-chan string, duplicates map[string]bool, dupeKey string, logger *log.Logger, isND bool) {
	for fi := range filename {
		process(fi, duplicates, dupeKey, logger, isND)
	}
}

func process(filename string, duplicates map[string]bool, dupeKey string, logger *log.Logger, isNewlineDelimited bool) {

	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("%s: unable to open input file %s...skipping it\n", name, filepath.Base(filename))
		return
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("%s: unable to read input file %s...skipping it\n", name, filepath.Base(filename))
		return
	}

	var fs []*geojson.Feature

	f, err := geojson.UnmarshalFeature(b)
	if err == nil {
		fs = append(fs, f)
	} else {
		fc, err := geojson.UnmarshalFeatureCollection(b)
		if err != nil {
			fmt.Printf("%s: invalid geojson feature or collection in input file %s...skipping it\n", name, filepath.Base(filename))
			return
		}
		fs = fc.Features
	}

	numFeatures := len(fs)
	badFeatures := 0
	duplicateFeatures := 0

	for _, f := range fs {
		if isDuplicate(f, dupeKey, duplicates) {
			duplicateFeatures++
			continue
		}

		b, err := json.Marshal(f)
		if err != nil {
			badFeatures++
			continue
		}

		line := string(b)
		if !isNewlineDelimited {
			line = fmt.Sprintf("%s,", line)
		}
		logger.Output(2, line)
	}

	if badFeatures+duplicateFeatures > 0 {
		fstr := "feature"
		if numFeatures > 1 {
			fstr = "features"
		}
		fmt.Printf("%s: skipped %d of %d %s in input file %s (%d duplicate, %d bad conversion)\n",
			name, badFeatures+duplicateFeatures, numFeatures, fstr, filepath.Base(filename), duplicateFeatures, badFeatures,
		)
	}
}

func isDuplicate(f *geojson.Feature, dupeKey string, duplicates map[string]bool) bool {
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

type Loader struct {
	InputPath      string
	OutputFilePath string
	Overwrite      bool
	Err            error
}

func (l *Loader) Input() {
	if l.Err != nil {
		return
	}

	if !gjfunks.DirExists(l.InputPath) {
		l.Err = fmt.Errorf("%s: input path does not exist or is not valid", name)
		return
	}
}

func (l *Loader) Output(isNewlineDelimited bool) {
	if l.Err != nil {
		return
	}

	var filename string
	defaultFilename := "gjfeatures"

	if l.OutputFilePath != "" {
		filename = filepath.Base(l.OutputFilePath)
	} else {
		filename = fmt.Sprintf("%s.geojson", defaultFilename)
		if isNewlineDelimited {
			filename = fmt.Sprintf("%s.ndjson", defaultFilename)
		}
		l.OutputFilePath = filepath.Join(".", filename)
	}

	if !gjfunks.DirExists(filepath.Dir(l.OutputFilePath)) {
		l.Err = fmt.Errorf("%s: output path does not exist or is not valid", name)
		return
	}
	if gjfunks.FileExists(l.OutputFilePath) && !l.Overwrite {
		l.Err = fmt.Errorf("%s: file '%s' already exists...but you can use --overwrite if you want to replace the old file", name, l.OutputFilePath)
		return
	}
	if gjfunks.FileExists(l.OutputFilePath) {
		err := os.Remove(l.OutputFilePath)
		if err != nil {
			l.Err = fmt.Errorf("%s: unable to remove output file", name)
			return
		}
	}
}

func (l *Loader) Files() []os.FileInfo {
	if l.Err != nil {
		return nil
	}
	files, err := ioutil.ReadDir(l.InputPath)
	if err != nil {
		l.Err = fmt.Errorf("%s: unable to read directory files", name)
		return nil
	}
	numFiles := len(files)
	if numFiles == 0 {
		l.Err = fmt.Errorf("%s: no files found in input path", name)
		return nil
	}
	return files
}

/////////////////////////////
