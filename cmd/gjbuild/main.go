package main

import (
	"fmt"

	"github.com/engelsjk/gjfunks/gjfunks"
	"gopkg.in/alecthomas/kingpin.v2"
)

// ToDo: add some flag options for handling features?
// 1. limit coordinate decimal precision
// 2. keep or remove properties by tag
// ?. ???

const (
	name   = "gjbuild"
	banner = `╋╋╋╋╋┏┓╋╋╋╋╋┏┓╋╋┏┓
╋╋╋╋┏┫┃╋╋╋╋╋┃┃╋╋┃┃
┏━━┓┗┫┗━┳┓┏┳┫┃┏━┛┃
┃┏┓┃┏┫┏┓┃┃┃┣┫┃┃┏┓┃
┃┗┛┃┃┃┗┛┃┗┛┃┃┗┫┗┛┃
┗━┓┃┃┣━━┻━━┻┻━┻━━┛
┏━┛┣┛┃
┗━━┻━┛
building one geojson file from many
try "gjbuild --help" to get started
`
	defaultFilename = "gjfeatures"
)

var (
	input       = kingpin.Arg("input", "input path").Default("").String()
	output      = kingpin.Flag("output", "output filepath").Default("").Short('o').String()
	filterKey   = kingpin.Flag("filter-key", "feature property key to filter duplicate values").Default("").String()
	keepOnlyKey = kingpin.Flag("keep-only-key", "keep only this feature property key").Default("").String()
	ndjson      = kingpin.Flag("ndjson", "output as newline-delimited json").Default("false").Bool()
	fixToSpec   = kingpin.Flag("fix-to-spec", "fix polygon/multipolygon features to meet RFC7946 S3.1.6").Default("false").Bool()
	overwrite   = kingpin.Flag("overwrite", "overwrite existing output file").Default("false").Bool()
)

func main() {

	kingpin.Parse()

	if *input == "" {
		fmt.Printf(banner)
		return
	}

	loader := gjfunks.Loader{
		InputDir:       *input,
		OutputFilePath: *output,
		Overwrite:      *overwrite,
	}

	loader.CheckInputDir()
	loader.SetOutputFilePath(*ndjson)

	files := loader.ListFiles()

	if err := gjfunks.Build(loader, files, gjfunks.BuildOptions{
		FilterKey:   *filterKey,
		KeepOnlyKey: *keepOnlyKey,
		NDJSON:      *ndjson,
		FixToSpec:   *fixToSpec,
	}); err != nil {
		fmt.Println(err.Error())
		return
	}
}
