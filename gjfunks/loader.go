package gjfunks

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Loader struct {
	Name           string
	InputDir       string
	OutputDir      string
	InputFilePath  string
	OutputFilePath string
	Overwrite      bool
	Err            error
}

// ErrorInvalidInputFile         = "gjsplitter: input file does not exist or is not valid\n"
// ErrorInvalidOutputDir         = "gjsplitter: output dir does not exist or is not valid\n"
// ErrorInvalidFeatureCollection = "gjsplitter: invalid geojson feature collection\n"
// ErrorInvalidFeature           = "gjsplitter: invalid geojson feature\n"
// ErrorJSONConversion           = "gjsplitter: unable to convert json\n"
// ErrorSaveFile                 = "gjsplitter: unable to save output file\n"

func (l *Loader) CheckInputDir() {
	if l.Err != nil {
		return
	}

	if !DirExists(l.InputDir) {
		l.Err = fmt.Errorf("%s: input directory does not exist or is not valid", l.Name)
		return
	}
}

func (l *Loader) CheckOutputDir() {
	if l.Err != nil {
		return
	}

	if !DirExists(l.OutputDir) {
		l.Err = fmt.Errorf("%s: output directory does not exist or is not valid", l.Name)
		return
	}
}

func (l *Loader) SetOutputFilePath(isNewlineDelimited bool) {
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

	if !DirExists(filepath.Dir(l.OutputFilePath)) {
		l.Err = fmt.Errorf("%s: output filepath does not exist or is not valid", l.Name)
		return
	}
	if FileExists(l.OutputFilePath) && !l.Overwrite {
		l.Err = fmt.Errorf("%s: file '%s' already exists...but you can use --overwrite if you want to replace the old filepath", l.Name, l.OutputFilePath)
		return
	}
	if FileExists(l.OutputFilePath) {
		err := os.Remove(l.OutputFilePath)
		if err != nil {
			l.Err = fmt.Errorf("%s: unable to remove output filepath", l.Name)
			return
		}
	}
}

func (l *Loader) ListFiles() []os.FileInfo {
	if l.Err != nil {
		return nil
	}
	files, err := ioutil.ReadDir(l.InputDir)
	if err != nil {
		l.Err = fmt.Errorf("%s: unable to read files in input directory", l.Name)
		return nil
	}
	numFiles := len(files)
	if numFiles == 0 {
		l.Err = fmt.Errorf("%s: no files found in input directory", l.Name)
		return nil
	}
	return files
}

func (l *Loader) ReadInput() []byte {
	if l.Err != nil {
		return nil
	}

	if l.InputFilePath != "" {
		if !FileExists(l.InputFilePath) {
			l.Err = fmt.Errorf("%s: input filepath does not exist or is not valid", l.Name)
			return nil
		}
	}

	file, err := GetFile(l.InputFilePath)
	if err != nil {
		l.Err = fmt.Errorf("%s: unable to open input (filepath or stdin)", l.Name)
		return nil
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		l.Err = fmt.Errorf("%s: unable to open input (filepath or stdin)", l.Name)
		return nil
	}
	if fi.Size() == 0 {
		l.Err = fmt.Errorf("%s: input is empty", l.Name)
		return nil
	}

	b, err := Open(file)
	if err != nil {
		l.Err = fmt.Errorf("%s: unable to open input (filepath or stdin)", l.Name)
		return nil
	}
	return b
}
