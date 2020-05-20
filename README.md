# gjfuncs

A small library and suite of tools for manipuilating GeoJSON files.

## gjsplitter

Splits a GeoJSON FeatureCollection up into separate Features.

```bash
go get github.com/engelsjk/cmd/gjsplitter
```

* Reads in from a file or stdin, spits out to separate files or stdout.
* Output file names use prefixes or unique values from a defined property key.

```bash
usage: gjsplitter [<flags>] [<input>]

Flags:
      --help       Show context-sensitive help (also try --help-long and --help-man).
  -k, --key=""     feature property key-value for filename prefix.
  -p, --prefix=""  output file prefix
  -o, --output=""  output dir
  -d, --dryrun     no output files saved

Args:
  [<input>]  input file
```  
 


