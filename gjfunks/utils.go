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

func FixRingWinding(p orb.Polygon) {
	// fix ring orientation to rfc7946 s3.1.6

	// outer ring must be counter-clockwise
	if p[0].Orientation() == -1 {
		p[0].Reverse()
	}
	// inner rings (aka holes) must be clockwise
	for _, pi := range p[1:] {
		if pi.Orientation() == 1 {
			pi.Reverse()
		}
	}
}

func RemoveAllPropertiesExcept(f *geojson.Feature, keepOnlyKey string) {
	for k := range f.Properties {
		if k != keepOnlyKey {
			delete(f.Properties, k)
		}
	}
}

func FixPolygons(f *geojson.Feature) {
	if mp, ok := f.Geometry.(orb.MultiPolygon); ok {
		for _, p := range mp {
			FixRingWinding(p)
		}
	}
	if p, ok := f.Geometry.(orb.Polygon); ok {
		FixRingWinding(p)
	}
}

func ConvertSingleMultiPolygonToPolygon(f *geojson.Feature) {
	if mp, ok := f.Geometry.(orb.MultiPolygon); ok {
		if len(mp) != 1 {
			return
		}
		f.Geometry = mp[0]
	}
}

func SplitMultiPolygon(f *geojson.Feature) []*geojson.Feature {
	features := []*geojson.Feature{}
	mp, ok := f.Geometry.(orb.MultiPolygon)
	if !ok {
		return features
	}
	for _, p := range mp {
		newf := geojson.NewFeature(p)
		CopyProperties(f, newf)
		features = append(features, newf)
	}
	return features
}

func CopyProperties(f1, f2 *geojson.Feature) {
	for k, v := range f1.Properties {
		f2.Properties[k] = v
	}
}
