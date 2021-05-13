package main

import (
	"fmt"

	"github.com/engelsjk/gjfunks"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	name   = "gjsplit"
	banner = `splitting one geojson file into many
try "gjsplit --help" to get started
`
)

var (
	input       = kingpin.Arg("input", "input file").Default("").String()
	nd          = kingpin.Flag("nd", "newline delimited input").Default("false").Bool()
	output      = kingpin.Flag("output", "output dir").Default("").Short('o').String()
	stdOut      = kingpin.Flag("stdout", "print to stdout only").Default("false").Bool()
	keepOnlyKey = kingpin.Flag("keeponlykey", "keep only this feature property key").Default("").String()
	outKey      = kingpin.Flag("outkey", "feature property key-value for output file prefixes").Default("").String()
	outPrefix   = kingpin.Flag("outprefix", "output file prefix").Default("").String()
	flatFile    = kingpin.Flag("flat", "flat output file").Default("true").Bool()
	fixToSpec   = kingpin.Flag("fixtospec", "fix polygon/multipolygon features to meet RFC7946 S3.1.6").Default("false").Bool()
	dryRun      = kingpin.Flag("dryrun", "no output files saved").Default("false").Short('d').Bool()
)

func main() {

	kingpin.Parse()

	if *input == "" {
		fmt.Printf(banner)
		return
	}

	loader := gjfunks.Loader{
		InputFilePath: *input,
		OutputDir:     *output,
	}

	if *output != "" {
		loader.CheckOutputDir()
	}

	if err := gjfunks.Split(loader, gjfunks.SplitOptions{
		InputFilePath:    *input,
		NewlineDelimited: *nd,
		OutputDir:        *output,
		OutKey:           *outKey,
		KeepOnlyKey:      *keepOnlyKey,
		OutPrefix:        *outPrefix,
		FlatFile:         *flatFile,
		FixToSpec:        *fixToSpec,
		StdOut:           *stdOut,
		DryRun:           *dryRun,
	}); err != nil {
		fmt.Println(err.Error())
		return
	}
}
