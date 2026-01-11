package internal

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/metdatasystem/us/pkg/awips"
	"github.com/twpayne/go-geom"
)

type ugc struct {
	ID        int               `json:"id,omitempty"`
	UGC       string            `json:"ugc"` // UGC code
	Name      string            `json:"name"`
	State     string            `json:"state"`
	Type      string            `json:"type"` // Either "C" (county) or "Z" (zone)
	Number    int               `json:"number"`
	Area      float64           `json:"area"`
	Geom      geom.MultiPolygon `json:"geom"`
	CWA       []string          `json:"cwa"` // County Warning Area
	IsMarine  bool              `json:"is_marine"`
	IsFire    bool              `json:"is_fire"`
	ValidFrom time.Time         `json:"valid_from"`
	ValidTo   *time.Time        `json:"valid_to"`
}

type ugcMinimal struct {
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

func getUGCs(ctx context.Context, tx pgx.Tx, ugcList *awips.UGC, isFire bool) ([]*ugcMinimal, error) {
	ugcs := []*ugcMinimal{}

	// For each state...
	for _, state := range ugcList.States {
		ugcType := state.Type
		// ...and for each area...
		for _, area := range state.Areas {
			if isFire {
				ugcType = "F"
			}

			if area == "000" || area == "ALL" {
				u, err := getUGCForStateMinimal(ctx, tx, state.ID, ugcType)
				if err != nil {
					return nil, err
				}

				ugcs = append(ugcs, u...)
			} else {
				ugcCode := state.ID + ugcType + area
				u, err := findUGCByCodeMinimal(ctx, tx, ugcCode)
				if err != nil {
					return nil, err
				}
				if u != nil {
					ugcs = append(ugcs, u)
				}
			}
		}
	}

	return ugcs, nil
}

// Get all UGCs for a given state but without geometry or area data
func getUGCForStateMinimal(ctx context.Context, tx pgx.Tx, state string, ugcType string) ([]*ugcMinimal, error) {

	rows, err := tx.Query(ctx, `
	SELECT id, ugc, name, state, type, number, cwa, is_marine, is_fire, valid_from, valid_to
	FROM postgis.ugcs WHERE state = $1 AND ugcType = $2 AND valid_to IS NULL
	`, state, ugcType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ugcs []*ugcMinimal
	for rows.Next() {
		ugc := &ugcMinimal{}
		if err := scanUGCMinimal(rows, ugc); err != nil {
			return nil, err
		}
		ugcs = append(ugcs, ugc)
	}

	return ugcs, nil
}

// Find a UGC by its code (e.g. "TXZ123" or "TXF123" for fire zones)
// but does not return geometry or area data
func findUGCByCodeMinimal(ctx context.Context, tx pgx.Tx, ugcCode string) (*ugcMinimal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := tx.QueryRow(ctx, `
	SELECT id, ugc, name, state, type, number, cwa, is_marine, is_fire, valid_from, valid_to
	FROM postgis.ugcs WHERE ugc = $1 AND valid_to IS NULL
	`, ugcCode)

	ugc := &ugcMinimal{}
	if err := scanUGCMinimal(row, ugc); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return ugc, nil
}

func scanUGCMinimal(row pgx.Row, ugc *ugcMinimal) error {
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
