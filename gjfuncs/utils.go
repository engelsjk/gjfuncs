package gjfuncs

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

func GetFile(filename string) (*os.File, error) {
	if filename == "" {
		return os.Stdin, nil
	} else if FileExists(filename) {
		return os.Open(filename)
	}
	return nil, errors.New("unable to open file")
}

func Open(filename string) ([]byte, error) {
	f, err := GetFile(filename)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	err = f.Close()
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
