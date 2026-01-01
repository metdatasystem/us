package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/everystreet/go-shapefile"
	"github.com/twpayne/go-geos"
)

func ParseMarineZones(scanner *shapefile.ZipScanner, t time.Time) error {

	slog.Info("Parsing marine zones...")

	// Start the scanner
	err := scanner.Scan()
	if err != nil {
		return err
	}

	info, err := scanner.Info()
	if err != nil {
		return err
	}

	ugcRecords := make(map[string]UGC, info.NumRecords)
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

		idAttr, _ := record.Attributes.Field("ID")
		id := fmt.Sprintf("%v", idAttr.Value())
		state := id[0:2]
		number := id[3:]

		zonename, _ := record.Attributes.Field("NAME")
		name := fmt.Sprintf("%v", zonename.Value())

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

		centre := geos.NewPoint([]float64{lon, lat})

		cwaAttr, _ := record.Attributes.Field("WFO")
		cwa := fmt.Sprintf("%v", cwaAttr.Value())
		cwaArr := make([]string, len(cwa)/3)

		for i := 0; i < len(cwa); i += 3 {
			cwaArr[i/3] = cwa[i : i+3]
		}

		ugc := UGC{
			ID:        id,
			Name:      name,
			State:     state,
			Number:    number,
			Type:      "Z",
			Area:      0.0, // Will be calculated in DB
			Centre:    centre,
			Geometry:  mpolygon,
			CWA:       cwaArr,
			IsMarine:  true,
			IsFire:    false,
			ValidFrom: t,
			ValidTo:   nil,
		}

		ugcRecords[ugc.ID] = ugc

		count++

	}

	// Err() returns the first error encountered during calls to Record()
	err = scanner.Err()
	if err != nil {
		return err
	}

	records, err := ToCSV(ugcRecords)
	if err != nil {
		return err
	}

	err = WriteToCSV("marinezones", records)
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Wrote %d records to marinezones.csv\n", len(ugcRecords)))

	return err
}
