package postgis

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twpayne/go-geos"
)

// Find a UGC by its code (e.g. "TXZ123" or "TXF123" for fire zones)
func FindUGCByCode(db *pgxpool.Pool, ugcCode string) (*UGC, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := db.QueryRow(ctx, `
	SELECT * FROM postgis.ugcs WHERE ugc = $1 AND valid_to IS NULL
	`, ugcCode)

	ugc := &UGC{}
	if err := ugc.scan(row); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return ugc, nil
}

// Find a UGC by its code (e.g. "TXZ123" or "TXF123" for fire zones)
// but does not return geometry or area data
func FindUGCByCodeMinimal(db *pgxpool.Pool, ugcCode string) (*UGCMinimal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := db.QueryRow(ctx, `
	SELECT id, ugc, name, state, type, number, cwa, is_marine, is_fire, valid_from, valid_to
	FROM postgis.ugcs WHERE ugc = $1 AND valid_to IS NULL
	`, ugcCode)

	ugc := &UGCMinimal{}
	if err := ugc.scan(row); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return ugc, nil
}

// Get all UGCs for a given state
func GetUGCForState(db *pgxpool.Pool, state string, ugcType string) ([]*UGC, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
	SELECT * FROM postgis.ugcs WHERE state = $1 AND type = $2 AND valid_to IS NULL
	`, state, ugcType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ugcs []*UGC
	for rows.Next() {
		ugc := &UGC{}
		if err := ugc.scan(rows); err != nil {
			return nil, err
		}
		ugcs = append(ugcs, ugc)
	}

	return ugcs, nil
}

// Get all UGCs for a given state but without geometry or area data
func GetUGCForStateMinimal(db *pgxpool.Pool, state string, ugcType string) ([]*UGCMinimal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
	SELECT id, ugc, name, state, type, number, cwa, is_marine, is_fire, valid_from, valid_to
	FROM postgis.ugcs WHERE state = $1 AND ugcType = $2 AND valid_to IS NULL
	`, state, ugcType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ugcs []*UGCMinimal
	for rows.Next() {
		ugc := &UGCMinimal{}
		if err := ugc.scan(rows); err != nil {
			return nil, err
		}
		ugcs = append(ugcs, ugc)
	}

	return ugcs, nil
}

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

func (ugc *UGC) scan(row pgx.Row) error {
	return row.Scan(
		&ugc.ID,
		&ugc.UGC,
		&ugc.Name,
		&ugc.State,
		&ugc.Type,
		&ugc.Number,
		&ugc.Area,
		&ugc.Geom,
		&ugc.CWA,
		&ugc.IsMarine,
		&ugc.IsFire,
		&ugc.ValidFrom,
		&ugc.ValidTo,
	)
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

func (ugc *UGCMinimal) scan(row pgx.Row) error {
	return row.Scan(
		&ugc.ID,
		&ugc.UGC,
		&ugc.Name,
		&ugc.State,
		&ugc.Type,
		&ugc.Number,
		&ugc.CWA,
		&ugc.IsMarine,
		&ugc.IsFire,
		&ugc.ValidFrom,
		&ugc.ValidTo,
	)
}
