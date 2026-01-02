package models

import (
	"time"

	"github.com/twpayne/go-geos"
)

type VTECEvent struct {
	ID           int        `json:"id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Issued       time.Time  `json:"issued"`
	Starts       *time.Time `json:"starts"`
	Expires      time.Time  `json:"expires"`
	Ends         time.Time  `json:"ends"`
	EndInitial   time.Time  `json:"end_initial"`
	Class        string     `json:"class"`
	Phenomena    string     `json:"phenomena"`
	WFO          string     `json:"wfo"`
	Significance string     `json:"significance"`
	EventNumber  int        `json:"event_number"`
	Year         int        `json:"year"`
	Title        string     `json:"title"`
	IsEmergency  bool       `json:"is_emergency"`
	IsPDS        bool       `json:"is_pds"`
}

type VTECUpdate struct {
	ID            int        `json:"id"`
	CreatedAt     time.Time  `json:"created_at,omitempty"`
	Issued        time.Time  `json:"issued"`
	Starts        *time.Time `json:"starts,omitempty"`
	Expires       time.Time  `json:"expires"`
	Ends          time.Time  `json:"ends,omitempty"`
	Text          string     `json:"text"`
	Product       string     `json:"product"`
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

type VTECUGC struct {
	ID           int        `json:"id"`
	CreatedAt    time.Time  `json:"created_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at,omitempty"`
	WFO          string     `json:"wfo"`
	Phenomena    string     `json:"phenomena"`
	Significance string     `json:"significance"`
	EventNumber  int        `json:"event_number"`
	UGC          int        `json:"ugc"`
	Issued       time.Time  `json:"issued"`
	Starts       *time.Time `json:"starts,omitempty"`
	Expires      time.Time  `json:"expires"`
	Ends         time.Time  `json:"ends,omitempty"`
	EndInitial   time.Time  `json:"end_initial,omitempty"`
	Action       string     `json:"action"`
	Year         int        `json:"year"`
}
