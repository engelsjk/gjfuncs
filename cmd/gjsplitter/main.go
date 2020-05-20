package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"strings"

	"github.com/engelsjk/gjfuncs/gjfuncs"
	"github.com/paulmach/orb/geojson"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	input  = kingpin.Arg("input", "input file").Default("").String()
	key    = kingpin.Flag("key", "feature property key-value for filename prefix.").Default("").Short('k').String()
	prefix = kingpin.Flag("prefix", "output file prefix").Default("").Short('p').String()
	output = kingpin.Flag("output", "output dir").Default("").Short('o').String()
	dryRun = kingpin.Flag("dryrun", "no output files saved").Default("false").Short('d').Bool()
)

var (
	ErrorOpenInput                = errors.New("error: unable to open input (filepath or stdin)")
	WarningStdinEmpty             = errors.New("warning: stdin is empty")
	ErrorInvalidInputFile         = errors.New("error: input file does not exist or is not valid")
	ErrorInvalidOutputDir         = errors.New("error: output dir does not exist or is not valid")
	ErrorInvalidFeatureCollection = errors.New("error: invalid geojson feature collection")
	ErrorInvalidFeature           = errors.New("error: invalid geojson feature")
	ErrorJSONConversion           = errors.New("error: unable to convert json")
	ErrorSaveFile                 = errors.New("error: unable to save output file")
)

func main() {

	kingpin.Parse()

	f, err := gjfuncs.GetFile(*input)
	if err != nil {
		fmt.Println(ErrorOpenInput)
		return
	}
	fi, err := f.Stat()
	if err != nil {
		fmt.Println(ErrorOpenInput)
		return
	}
	if fi.Size() == 0 {
		fmt.Println(WarningStdinEmpty)
		return
	}

	b, err := gjfuncs.Open(f)
	if err != nil {
		fmt.Println(ErrorOpenInput)
		return
	}

	fc, err := geojson.UnmarshalFeatureCollection(b)
	if err != nil {
		fmt.Println(ErrorInvalidFeatureCollection)
	}

	if *input != "" {
		if !gjfuncs.FileExists(*input) {
			fmt.Println(ErrorInvalidInputFile)
			return
		}
	}

	if *output != "" {
		if !gjfuncs.DirExists(*output) {
			fmt.Println(ErrorInvalidOutputDir)
			return
		}
	}

	numFeatures := len(fc.Features)
	for i, f := range fc.Features {

		//////////////////
		// write to stdout

		if *output == "" && *key == "" {
			// json w/ no indents if stdout
			b, err := f.MarshalJSON()
			if err != nil {
				fmt.Println(ErrorInvalidFeature)
				return
			}
			fmt.Println(string(b))
			continue
		}

		////////////////
		// write to file

		// pad index by max width zeros
		w := 1 + int(math.Log10(float64(numFeatures)))
		idx := fmt.Sprintf("%0*d", w, i+1)

		// filename "feature-[1,2,3...].geojson (default)"
		filename := Filename("", "feature", idx)

		// filename "input-[1,2,3...].geojson"
		if *input != "" {
			filename = Filename(*input, "", idx)
		}

		// filename "prefix-[1,2,3...].geojson"
		if *prefix != "" {
			filename = Filename("", *prefix, idx)
		}

		// output filename by feature property key-value if available, e.g. "keyvalue.geojson"
		if *key != "" {
			v, ok := f.Properties[*key]
			if ok {
				if s, ok := v.(string); ok {
					filename = Filename("", s, "")
				}
			}
		}

		outputFilePath := filename
		if *output != "" {
			outputFilePath = filepath.Join(*output, filename)
		}

		// print to stdout (no file save) if dry run
		if *dryRun {
			fmt.Println(outputFilePath)
			continue
		}

		// indent json (pretty-print kinda) if writing to file
		b, err := json.MarshalIndent(f, "", " ")
		if err != nil {
			fmt.Println(ErrorJSONConversion)
			return
		}

		err = ioutil.WriteFile(outputFilePath, b, 0644)
		if err != nil {
			fmt.Println(ErrorSaveFile)
			return
		}
	}
}

func Filename(filename, prefix, suffix string) string {
	fn := filepath.Base(filename)
	if prefix == "" {
		return fmt.Sprintf("%s-%s.geojson", strings.TrimSuffix(fn, filepath.Ext(fn)), suffix)
	}
	if suffix == "" {
		return fmt.Sprintf("%s.geojson", prefix)
	}
	return fmt.Sprintf("%s-%s.geojson", prefix, suffix)
}
