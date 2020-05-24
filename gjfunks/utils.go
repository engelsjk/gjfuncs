package gjfunks

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func GetFile(filename string) (*os.File, error) {
	if filename == "" {
		return os.Stdin, nil
	}
	if FileExists(filename) {
		return os.Open(filename)
	}
	return nil, fmt.Errorf("unable to get file %s", filename)
}

func Open(f *os.File) ([]byte, error) {
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func DirExists(dir string) bool {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func IsGeoJSONExt(filename string) bool {
	return filepath.Ext(filename) == ".geojson"
}
