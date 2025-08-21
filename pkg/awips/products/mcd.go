package products

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/metdatasystem/us/pkg/awips"
	"github.com/twpayne/go-geom"
)

type MCD struct {
	Original         string       `json:"original"`
	Number           int          `json:"number"`
	Issued           time.Time    `json:"issued"`
	Expires          time.Time    `json:"expires"`
	Concerning       string       `json:"concerning"`
	Polygon          geom.Polygon `json:"polygon"`
	WatchProbability int          `json:"watch_probability"`
	MostProbTornado  string       `json:"most_prob_tornado"`
	MostProbGust     string       `json:"most_prob_gust"`
	MostProbHail     string       `json:"mopst_prob_hail"`
}

// Parses a Mesoscale Discussion (MCD) from the given text.
func ParseMCD(text string) (*MCD, error) {

	numberRegexp := regexp.MustCompile("([0-9]+)")

	// Identify the MCD number
	mcdRegex := regexp.MustCompile("(Mesoscale Discussion )([0-9]{4})")
	mcdString := mcdRegex.FindString(text)
	numberString := numberRegexp.FindString(mcdString)
	if numberString == "" {
		return nil, errors.New("error parsing mcd: No MCD number found")
	}
	// Convert the number
	number, err := strconv.Atoi(numberString)
	if err != nil {
		return nil, fmt.Errorf("error parsing mcd number: %s", err.Error())
	}

	// Find the valid times of the MCD
	validRegex := regexp.MustCompile("(Valid|VALID) ([0-9]{6}Z) - ([0-9]{6}Z)\n")
	validString := strings.TrimSpace(validRegex.FindString(text))
	timeRegex := regexp.MustCompile("([0-9]{6}Z)")
	times := timeRegex.FindAllString(validString, 2)
	if len(times) != 2 {
		return nil, fmt.Errorf("error parsing mcd: Invalid number of valid times. Found %d, expected 2", len(times))
	}
	// Parse the issuance times
	issued, err := time.Parse("021504Z", times[0])
	if err != nil {
		return nil, fmt.Errorf("error parsing mcd issued time: %s", err.Error())
	}
	// Parse the expiration times
	expires, err := time.Parse("021504Z", times[1])
	if err != nil {
		return nil, fmt.Errorf("error parsing mcd expire time: %s", err.Error())
	}

	// Find the concerning text
	concerningRegex := regexp.MustCompile(`(Concerning\.\.\.)(.+)`)
	concerningString := concerningRegex.FindString(text)
	if concerningString == "" {
		return nil, fmt.Errorf("error parsing mcd: No concerning text found")
	}
	concerning := strings.ReplaceAll(concerningString, "Concerning...", "")

	//  Parse the LatLon segment
	latlon, err := awips.ParseLatLon(text)
	if err != nil {
		return nil, fmt.Errorf("error parsing mcd latlon: %s", err.Error())
	}
	// Get the polygon
	polygon, err := latlon.ToPolygon()
	if err != nil {
		return nil, fmt.Errorf("error parsing mcd latlon to polygon: %s", err.Error())
	}

	// Find the probability of watch issuance
	probabilityRegexp := regexp.MustCompile(`(Probability of Watch Issuance\.\.\.)(.+)`)
	probabilityString := probabilityRegexp.FindString(text)
	var probability int
	if probabilityString != "" {
		valueString := numberRegexp.FindString(probabilityString)

		if valueString == "" {
			return nil, fmt.Errorf("error parsing mcd: Found probability string but no numbers")
		}

		probability, err = strconv.Atoi(valueString)
		if err != nil {
			return nil, fmt.Errorf("error parsing mcd probability: %s", err.Error())
		}
	}
	// Find the probable tornado intensity
	probTornadoRegexp := regexp.MustCompile(`(MOST PROBABLE PEAK TORNADO INTENSITY\.\.\.)([\w ]+)`)
	probTornadoString := probTornadoRegexp.FindString(text)
	var probTornado string
	if probTornadoString != "" {
		values := strings.Split(probTornadoString, "...")
		if len(values) < 2 {
			return nil, fmt.Errorf("tornado probability string was found but split returned %d elements", len(values))
		}
		probTornado = values[1]
	}

	// Find the probable gust intensity
	probGustRegexp := regexp.MustCompile(`(MOST PROBABLE PEAK TORNADO INTENSITY\.\.\.)([\w ]+)`)
	probGustString := probGustRegexp.FindString(text)
	var probGust string
	if probGustString != "" {
		values := strings.Split(probGustString, "...")
		if len(values) < 2 {
			return nil, fmt.Errorf("gust probability string was found but split returned %d elements", len(values))
		}
		probGust = values[1]
	}

	// Find the probable hail intensity
	probHailRegexp := regexp.MustCompile(`(MOST PROBABLE PEAK TORNADO INTENSITY\.\.\.)([\w ]+)`)
	probHailString := probHailRegexp.FindString(text)
	var probHail string
	if probHailString != "" {
		values := strings.Split(probHailString, "...")
		if len(values) < 2 {
			return nil, fmt.Errorf("hail probability string was found but split returned %d elements", len(values))
		}
		probHail = values[1]
	}

	mcd := MCD{
		Original:         text,
		Number:           number,
		Issued:           issued,
		Expires:          expires,
		Concerning:       concerning,
		Polygon:          *polygon,
		WatchProbability: probability,
		MostProbTornado:  probTornado,
		MostProbGust:     probGust,
		MostProbHail:     probHail,
	}

	return &mcd, nil
}
