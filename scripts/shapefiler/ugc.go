package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/twpayne/go-geos"
)

type UGC struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	State     string     `json:"state"`
	Number    string     `json:"number"`
	Type      string     `json:"type"`
	Area      float64    `json:"area"`
	Centre    *geos.Geom `json:"centre"`
	Geometry  *geos.Geom `json:"geometry"`
	CWA       []string   `json:"cwa"`
	IsMarine  bool       `json:"is_marine"`
	IsFire    bool       `json:"is_fire"`
	ValidFrom time.Time  `json:"valid_from"`
	ValidTo   *time.Time `json:"valid_to"`
}

func ToCSV(ugcs map[string]UGC) ([][]string, error) {

	records := [][]string{}

	header := []string{"ugc", "name", "state", "type", "number", "area", "geom", "cwa", "is_marine", "is_fire", "valid_from"}
	records = append(records, header)

	for _, ugc := range ugcs {
		cwa := "{"
		for i, c := range ugc.CWA {
			cwa += fmt.Sprintf("\"%s\"", c)
			if i < len(ugc.CWA)-1 {
				cwa += ","
			}
		}
		cwa += "}"

		record := []string{
			ugc.ID,
			ugc.Name,
			ugc.State,
			ugc.Type,
			ugc.Number,
			"0.0",
			ugc.Geometry.ToWKT(),
			cwa,
			fmt.Sprintf("%v", ugc.IsMarine),
			fmt.Sprintf("%v", ugc.IsFire),
			DateToString(&ugc.ValidFrom),
		}
		records = append(records, record)
	}

	return records, nil
}

func ToSQL(ugcs map[string]UGC) (string, error) {
	result := "INSERT INTO postgis.ugcs(ugc, name, state, type, number, area, geom, cwa, is_marine, is_fire, valid_from) VALUES \n"

	for _, ugc := range ugcs {

		geometry := ugc.Geometry.ToWKT()

		ugc.Name = strings.ReplaceAll(ugc.Name, "'", "''")

		cwa := "'{"
		for i, c := range ugc.CWA {
			cwa += fmt.Sprintf("\"%s\"", c)
			if i < len(ugc.CWA)-1 {
				cwa += ","
			}
		}
		cwa += "}'"

		result += fmt.Sprintf("('%s', '%s', '%s', '%s', %s, 0.0, ST_GeomFromWKT('%s'), %s, %v, %v, %s),\n",
			ugc.ID, ugc.Name, ugc.State, ugc.Type, ugc.Number, geometry, cwa, ugc.IsMarine, ugc.IsFire, DateToString(&ugc.ValidFrom))
	}

	result = result[:len(result)-2]

	return result, nil
}
