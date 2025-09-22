package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/twpayne/go-geos"
)

type Warning struct {
	ID            int        `json:"id"`
	CreatedAt     time.Time  `json:"created_at,omitzero"`
	UpdatedAt     time.Time  `json:"updated_at,omitzero"`
	Issued        time.Time  `json:"issued"`
	Starts        *time.Time `json:"starts,omitzero"`
	Expires       time.Time  `json:"expires"`
	Ends          time.Time  `json:"ends,omitzero"`
	EndInitial    time.Time  `json:"end_initial,omitzero"`
	Text          string     `json:"text"`
	WFO           string     `json:"wfo"`
	Action        string     `json:"action"`
	Class         string     `json:"class"`
	Phenomena     string     `json:"phenomena"`
	Significance  string     `json:"significance"`
	EventNumber   int        `json:"event_number"`
	Year          int        `json:"year"`
	Title         string     `json:"title"`
	IsEmergency   bool       `json:"is_emergency"`
	IsPDS         bool       `json:"is_pds"`
	Geom          *geos.Geom `json:"geom,omitempty"`
	Direction     *int       `json:"direction"`
	Location      *geos.Geom `json:"location"`
	Speed         *int       `json:"speed"`
	SpeedText     *string    `json:"speed_text"`
	TMLTime       *time.Time `json:"tml_time"`
	UGC           []string   `json:"ugc"`
	Tornado       string     `json:"tornado,omitempty"`
	Damage        string     `json:"damage,omitempty"`
	HailThreat    string     `json:"hail_threat,omitempty"`
	HailTag       string     `json:"hail_tag,omitempty"`
	WindThreat    string     `json:"wind_threat,omitempty"`
	WindTag       string     `json:"wind_tag,omitempty"`
	FlashFlood    string     `json:"flash_flood,omitempty"`
	RainfallTag   string     `json:"rainfall_tag,omitempty"`
	FloodTagDam   string     `json:"flood_tag_dam,omitempty"`
	SpoutTag      string     `json:"spout_tag,omitempty"`
	SnowSquall    string     `json:"snow_squall,omitempty"`
	SnowSquallTag string     `json:"snow_squall_tag,omitempty"`
}

func (warning *Warning) GenerateID() string {
	return fmt.Sprintf("%v-%v-%v-%04v-%v", warning.WFO, warning.Phenomena, warning.Significance, warning.EventNumber, warning.Year)
}

func (w *Warning) MarshalJSON() ([]byte, error) {
	type Alias Warning // Use type alias to avoid recursion

	aux := struct {
		*Alias
		Geom     []byte `json:"geom,omitempty"`
		Location []byte `json:"location,omitempty"`
	}{
		Alias: (*Alias)(w),
	}

	if w.Geom != nil {
		aux.Geom = w.Geom.ToWKB()
	}

	if w.Location != nil {
		aux.Location = w.Geom.ToWKB()
	}

	return json.Marshal(aux)
}

func (w *Warning) UnmarshalJSON(data []byte) error {
	type Alias Warning // Use type alias to avoid recursion

	aux := struct {
		*Alias
		Geom     []byte `json:"geom,omitempty"`
		Location []byte `json:"location,omitempty"`
	}{
		Alias: (*Alias)(w),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("failed to unmarshal warning: %w", err)
	}

	if len(aux.Geom) > 0 {
		geom, err := geos.NewGeomFromWKB(aux.Geom)
		if err != nil {
			return fmt.Errorf("failed to parse geometry WKB: %w", err)
		}
		w.Geom = geom
	}

	if len(aux.Location) > 0 {
		location, err := geos.NewGeomFromWKB(aux.Location)
		if err != nil {
			return fmt.Errorf("failed to parse location WKB: %w", err)
		}
		w.Location = location
	}

	return nil
}
