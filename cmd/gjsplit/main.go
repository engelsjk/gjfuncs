package main

import (
	"fmt"

	"github.com/engelsjk/gjfunks/gjfunks"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	banner = `╋╋╋╋╋╋╋╋╋╋╋┏┓╋┏┓
╋╋╋╋┏┓╋╋╋╋╋┃┃┏┛┗┓
┏━━┓┗╋━━┳━━┫┃┣┓┏┛
┃┏┓┃┏┫━━┫┏┓┃┃┣┫┃
┃┗┛┃┃┣━━┃┗┛┃┗┫┃┗┓
┗━┓┃┃┣━━┫┏━┻━┻┻━┛
┏━┛┣┛┃╋╋┃┃
┗━━┻━┛╋╋┗┛
splitting one geojson file into many
try "gjsplit --help" to get started
`
)

var (
	input       = kingpin.Arg("input", "input file").Default("").String()
	output      = kingpin.Flag("output", "output dir").Default("").Short('o').String()
	stdOut      = kingpin.Flag("stdout", "print to stdout only").Default("false").Bool()
	keepOnlyKey = kingpin.Flag("keep-only-key", "keep only this feature property key").Default("").String()
	outKey      = kingpin.Flag("out-key", "feature property key-value for output file prefixes").Default("").String()
	outPrefix   = kingpin.Flag("out-prefix", "output file prefix").Default("").String()
	fixToSpec   = kingpin.Flag("fix-to-spec", "fix polygon/multipolygon features to meet RFC7946 S3.1.6").Default("false").Bool()
	dryRun      = kingpin.Flag("dry-run", "no output files saved").Default("false").Short('d').Bool()
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

	b := loader.ReadInput()

	if err := gjfunks.Split(loader, b, gjfunks.SplitOptions{
		InputFilePath: *input,
		OutputDir:     *output,
		OutKey:        *outKey,
		KeepOnlyKey:   *keepOnlyKey,
		OutPrefix:     *outPrefix,
		FixToSpec:     *fixToSpec,
		StdOut:        *stdOut,
		DryRun:        *dryRun,
	}); err != nil {
		fmt.Println(err.Error())
		return
	}
}
