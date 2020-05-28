package gjfunks

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
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

func FmtFilename(filename, prefix, suffix string) string {
	fn := filepath.Base(filename)
	if prefix == "" {
		return fmt.Sprintf("%s-%s.geojson", strings.TrimSuffix(fn, filepath.Ext(fn)), suffix)
	}
	if suffix == "" {
		return fmt.Sprintf("%s.geojson", prefix)
	}
	return fmt.Sprintf("%s-%s.geojson", prefix, suffix)
}

func FixPolygons(f *geojson.Feature) {
	if mp, ok := f.Geometry.(orb.MultiPolygon); ok {
		for _, p := range mp {
			fixPolygon(p)
		}
	}
	if p, ok := f.Geometry.(orb.Polygon); ok {
		fixPolygon(p)
	}
}
