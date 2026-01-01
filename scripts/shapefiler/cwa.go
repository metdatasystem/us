package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/everystreet/go-shapefile"
	"github.com/twpayne/go-geos"
)

type CWA struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Centre    *geos.Geom `json:"centre"`
	Geometry  *geos.Geom `json:"geometry"`
	Area      float64    `json:"area"`
	WFO       string     `json:"wfo"`
	Region    string     `json:"region"`
	ValidFrom time.Time  `json:"valid_from"`
}

func ParseCWA(scanner *shapefile.ZipScanner, t time.Time) error {

	// Start the scanner
	err := scanner.Scan()
	if err != nil {
		return err
	}

	info, err := scanner.Info()
	if err != nil {
		return err
	}

	cwaRecords := make([]CWA, info.NumRecords)
	count := 0

	for {
		record := scanner.Record()
		if record == nil {
			break
		}

		shape := record.Shape.GeoJSONFeature()
		mpolygon, err := GetShape(shape)
		if err != nil {
			return err
		}

		cwaAttr, _ := record.Attributes.Field("CWA")
		id := fmt.Sprintf("%v", cwaAttr.Value())

		cwaName, _ := record.Attributes.Field("CITY")
		name := fmt.Sprintf("%v", cwaName.Value())

		lonAttr, _ := record.Attributes.Field("LON")
		lon, err := getFloat(lonAttr.Value())
		if err != nil {
			return err
		}

		latAttr, _ := record.Attributes.Field("LAT")
		lat, err := getFloat(latAttr.Value())
		if err != nil {
			return err
		}

		location := geos.NewPoint([]float64{lon, lat})

		regionAttr, _ := record.Attributes.Field("REGION")
		region := fmt.Sprintf("%v", regionAttr.Value())

		cwa := CWA{
			ID:        id,
			Name:      name,
			Centre:    location,
			Geometry:  mpolygon,
			Area:      0.0,
			WFO:       id,
			Region:    region,
			ValidFrom: t,
		}

		cwaRecords[count] = cwa

		count++

	}

	// Err() returns the first error encountered during calls to Record()
	err = scanner.Err()
	if err != nil {
		return err
	}

	records := [][]string{}

	header := []string{"id", "name", "area", "geom", "wfo", "region", "valid_from"}
	records = append(records, header)

	for _, cwa := range cwaRecords {
		geometry := cwa.Geometry.ToWKT()

		record := []string{
			cwa.ID,
			cwa.Name,
			fmt.Sprintf("%f", cwa.Area),
			geometry,
			cwa.WFO,
			cwa.Region,
			DateToString(&cwa.ValidFrom),
		}
		records = append(records, record)
	}

	err = WriteToCSV("cwa", records)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Wrote %d records to cwa.csv\n", len(cwaRecords)))

	return nil
}
