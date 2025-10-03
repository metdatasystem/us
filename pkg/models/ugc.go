package models

import (
	"time"

	"github.com/twpayne/go-geos"
)

type UGC struct {
	ID        int        `json:"id,omitempty"`
	UGC       string     `json:"ugc"` // UGC code
	Name      string     `json:"name"`
	State     string     `json:"state"`
	Type      string     `json:"type"` // Either "C" (county) or "Z" (zone)
	Number    int        `json:"number"`
	Area      float64    `json:"area"`
	Geom      *geos.Geom `json:"geom"`
	CWA       []string   `json:"cwa"` // County Warning Area
	IsMarine  bool       `json:"is_marine"`
	IsFire    bool       `json:"is_fire"`
	ValidFrom time.Time  `json:"valid_from"`
	ValidTo   *time.Time `json:"valid_to"`
}

type UGCMinimal struct {
	ID        int        `json:"id,omitempty"`
	UGC       string     `json:"ugc"` // UGC code
	Name      string     `json:"name"`
	State     string     `json:"state"`
	Type      string     `json:"type"` // Either "C" (county) or "Z" (zone)
	Number    int        `json:"number"`
	CWA       []string   `json:"cwa"` // County Warning Area
	IsMarine  bool       `json:"is_marine"`
	IsFire    bool       `json:"is_fire"`
	ValidFrom time.Time  `json:"valid_from"`
	ValidTo   *time.Time `json:"valid_to"`
}
