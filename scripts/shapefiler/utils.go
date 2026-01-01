package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"time"

	"github.com/everystreet/go-geojson/v2"
	"github.com/everystreet/go-shapefile"
	"github.com/twpayne/go-geos"
)

func DateToString(date *time.Time) string {
	if date == nil {
		return "null"
	} else {
		return fmt.Sprintf("'%s'", date.Format(time.DateOnly))
	}
}

func getFloat(unk interface{}) (float64, error) {
	v := reflect.ValueOf(unk)
	v = reflect.Indirect(v)
	if !v.Type().ConvertibleTo(floatType) {
		return math.NaN(), fmt.Errorf("cannot convert %v to float64", v.Type())
	}
	fv := v.Convert(floatType)
	return fv.Float(), nil
}

func getorCreateFile(filename string) (*os.File, error) {
	_, err := os.Stat(filename)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	} else {
		err := os.Remove(filename)

		if err != nil {
			return nil, err
		}

	}

	return os.Create(filename)
}

func WriteToCSV(name string, records [][]string) error {
	file, err := getorCreateFile(fmt.Sprintf("%s.csv", name))
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, record := range records {
		err := writer.Write(record)
		if err != nil {
			return err
		}
	}

	return nil
}

func WriteToFile(filename string, contents []byte) error {
	var err error
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		_, err = os.Create(filename)
		if err != nil {
			return err
		}
	} else {
		err = os.Remove(filename)
		if err != nil {
			return err
		}
	}

	err = os.WriteFile(filename, contents, os.ModePerm)

	return err
}

func CreateZipScanner(filename string) (*shapefile.ZipScanner, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// Create new ZipScanner
	// The filename can be replaced with an empty string if you don't want to check filenames inside the zip file
	scanner, err := shapefile.NewZipScanner(file, stat.Size(), filename, shapefile.PointPrecision(6))
	if err != nil {
		return nil, err
	}

	return scanner, nil
}

func GetShape(shape *geojson.Feature) (*geos.Geom, error) {
	var mpolygon *geojson.Feature
	switch f := shape.Geometry.(type) {
	case *geojson.Polygon:
		mpolygon = geojson.NewMultiPolygon(*f)
	case *geojson.MultiPolygon:
		mpolygon = shape
	default:
		return nil, errors.New("shape was not a valid polygon")
	}

	geometry, err := mpolygon.MarshalJSON()
	if err != nil {
		panic(err)
	}

	geom, err := geos.NewGeomFromGeoJSON(string(geometry))
	if err != nil {
		panic(err)
	}

	// reduced := simplify.DouglasPeucker(0.0001).Simplify(geom.Geometry)
	// if !ok {
	// 	return nil, fmt.Errorf("could not assert type of orb.Geometry to orb.MultiPolygon")
	// }

	return geom, nil
}
