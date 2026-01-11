package awips

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/twpayne/go-geom"
)

// The rules of latitude and longitude information in text products is defined in NWS directive 10-1701 section 5.6.
// The directive states that WFO and National products have different LAT..LON formats. The below functions allow for either of the formats to be parsed.

const LatLonRegexp = `(?m)(^LAT\.\.\.LON\s+(\d+\s*)+)`
const PointRegexp = `(?m)(\d{4,8})`

type LatLon struct {
	Original string       `json:"original"`
	Coords   []geom.Coord `json:"points"`
}

// Find the LAT...LON information in the text.
func FindLatLon(text string) string {
	latlonRegexp := regexp.MustCompile(LatLonRegexp)
	return latlonRegexp.FindString(text)
}

// Parse the text and retrieve coordinates from the LAT...LON information.
func ParseLatLon(text string) (*LatLon, error) {

	original := FindLatLon(text)

	if original == "" {
		return nil, nil
	}

	segments := FindLatLonSegments(original)

	coords, err := ParseLatLonSegments(segments)
	if err != nil {
		return nil, errors.New("failed to parse lat/lon segments: " + err.Error())
	}

	return &LatLon{
		Original: original,
		Coords:   coords,
	}, nil

}

// A segment is a 4 to 8 digit string separated by a space according to the directive.
// We just find parts of the string that are 4 to 8 digits.
func FindLatLonSegments(text string) []string {
	segmentRegexp := regexp.MustCompile(PointRegexp)
	return segmentRegexp.FindAllString(text, -1)
}

// Parse an array of strings as segments into coordinates.
// Each segment can either be a 4 or 8 digit string as per NWS directive 10-1701 section 5.6. Any other length is invalid.
func ParseLatLonSegments(segments []string) ([]geom.Coord, error) {
	points := []geom.Coord{}

	for i := 0; i < len(segments); i += 1 {
		segment := segments[i]

		switch len(segment) {
		// 4-5 digit segments usually in WFO products
		// NWS LAT...LON is latitude then longitude, so Y then X
		case 4:
			fallthrough
		case 5:
			// Latitude
			y, err := Parse45Digit(segment)
			if err != nil {
				return nil, err
			}

			// Next segment
			i++
			if i >= len(segments) {
				return nil, errors.New("missing longitude segment for latitude " + segment)
			}
			segment = segments[i]

			// Longitude
			x, err := Parse45Digit(segment)
			if err != nil {
				return nil, err
			}

			points = append(points, geom.Coord{-x, y})
		case 8:
			coords, err := ParseCoord8(segment)
			if err != nil {
				return nil, err
			}

			points = append(points, coords)
		}
	}

	last := points[len(points)-1]
	if !last.Equal(geom.XY, points[0]) {
		points = append(points, points[0])
	}

	return points, nil
}

// Parse a 4-5 digit string to a valid floating point.
func Parse45Digit(text string) (float64, error) {

	x := 0.0

	if len(text) < 4 || len(text) > 5 {
		return x, fmt.Errorf("string must be 4 or 5 digits, got %s (%d)", text, len(text))
	}

	n, err := strconv.Atoi(text)
	if err != nil {
		return x, err
	}

	x = float64(n) / 100.0

	// NWS directive 10-1701 section 5.6 infers a west-bias for longitudes, meaning that longitudes greater than 180 degrees should continue to be greater than 180.
	// For example, if the coordinate is 179.00 E (-179.00), it will be sent as 18100 (181.00 W).
	// Some WFO products have their warning area in the east, so they are exempt from this (e.g. Guam).

	return x, nil
}

// Parse an 8 digit string to a valid coordinate.
func ParseCoord8(text string) (geom.Coord, error) {
	if len(text) != 8 {
		return nil, errors.New("string must be 8 digits")
	}

	latString := text[0:4]
	lonString := text[4:8]

	lat, err := strconv.Atoi(latString)
	if err != nil {
		return nil, err
	}
	lon, err := strconv.Atoi(lonString)
	if err != nil {
		return nil, err
	}

	latFloat := float64(lat) / 100.0
	lonFloat := float64(lon) / 100.0

	if lonFloat < 50.0 {
		lonFloat += 100.0
	}

	lonFloat = -lonFloat

	return geom.Coord{lonFloat, latFloat}, nil

}

// NWS directive 10-1701 section 5.6 infers a west-bias for longitudes, meaning that longitudes greater than 180 degrees should continue to be greater than 180.
// For example, if the coordinate is 179.00 E (-179.00), it will be sent as 18100 (181.00 W).
// Some WFO products have their warning area in the east, so they are exempt from this (e.g. Guam).
//
// This function applies the west-bias to the coordinates.
func LonWestBias(coords []geom.Coord) []geom.Coord {

	for _, coord := range coords {
		if coord.X() > 180.0 {
			coord[0] = coord.X() - 360.0
		}
	}

	return coords
}

// Convert the LatLon points to a Polygon.
func (latlon *LatLon) ToPolygon() (*geom.Polygon, error) {
	polygon := geom.NewPolygon(geom.XY)

	return polygon.SetCoords([][]geom.Coord{latlon.Coords})
}

// Convert the LatLon points to a MultiPolygon.
func (latlon *LatLon) ToMultiPolygon() (*geom.MultiPolygon, error) {
	polygon := geom.NewMultiPolygon(geom.XY)

	return polygon.SetCoords([][][]geom.Coord{{latlon.Coords}})
}

func (latlon *LatLon) ToFloat() [][]float64 {
	coords := [][]float64{}
	for _, coord := range latlon.Coords {
		coords = append(coords, []float64{coord.X(), coord.Y()})
	}

	return coords
}

func (latlon *LatLon) ToFloatClosing() [][]float64 {
	coords := latlon.ToFloat()

	if coords[0][0] != coords[len(coords)-1][0] || coords[0][1] != coords[len(coords)-1][1] {
		coords = append(coords, coords[0])
	}

	return coords
}

func (latlon *LatLon) SetWestCoords() {
	for _, coord := range latlon.Coords {
		coord[0] = -coord.X()
	}
}
