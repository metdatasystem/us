package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metdatasystem/us/pkg/models"
)

func ScanWarning(row pgx.Row, warning *models.Warning) error {
	return row.Scan(
		&warning.ID,
		&warning.CreatedAt,
		&warning.UpdatedAt,
		&warning.Issued,
		&warning.Starts,
		&warning.Expires,
		&warning.Ends,
		&warning.EndInitial,
		&warning.Text,
		&warning.WFO,
		&warning.Action,
		&warning.Class,
		&warning.Phenomena,
		&warning.Significance,
		&warning.EventNumber,
		&warning.Year,
		&warning.Title,
		&warning.IsEmergency,
		&warning.IsPDS,
		&warning.Geom,
		&warning.Direction,
		&warning.Location,
		&warning.Speed,
		&warning.SpeedText,
		&warning.TMLTime,
		&warning.UGC,
		&warning.Tornado,
		&warning.Damage,
		&warning.HailThreat,
		&warning.HailTag,
		&warning.WindThreat,
		&warning.WindTag,
		&warning.FlashFlood,
		&warning.RainfallTag,
		&warning.FloodTagDam,
		&warning.SpoutTag,
		&warning.SnowSquall,
		&warning.SnowSquallTag,
	)
}

func FindWarning(db *pgxpool.Pool, wfo string, phenomena string, significance string, eventNumber int, year int) (*models.Warning, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
	SELECT * FROM warnings.warnings WHERE wfo = $1 AND phenomena = $2 AND significance = $3 AND event_number = $4 AND year = $5
	`, wfo, phenomena, significance, eventNumber, year)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var warning *models.Warning
	if rows.Next() {
		warning = &models.Warning{}
		if err := ScanWarning(rows, warning); err != nil {
			return nil, err
		}
	}

	return warning, nil
}

func InsertWarning(db *pgxpool.Pool, warning *models.Warning) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
       INSERT INTO warnings.warnings(
	       issued, starts, expires, ends, end_initial, text, 
		   wfo, action, class, phenomena, significance, event_number, year, 
		   title, is_emergency, is_pds, geom, direction, location, speed, speed_text, tml_time, 
		   ugc, tornado, damage, hail_threat, hail_tag, wind_threat, wind_tag, flash_flood, 
		   rainfall_tag, flood_tag_dam, spout_tag, snow_squall, snow_squall_tag
       ) VALUES (
	       $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, 
		   $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35
       )
       `,
		warning.Issued,
		warning.Starts,
		warning.Expires,
		warning.Ends,
		warning.EndInitial,
		warning.Text,
		warning.WFO,
		warning.Action,
		warning.Class,
		warning.Phenomena,
		warning.Significance,
		warning.EventNumber,
		warning.Year,
		warning.Title,
		warning.IsEmergency,
		warning.IsPDS,
		warning.Geom,
		warning.Direction,
		warning.Location,
		warning.Speed,
		warning.SpeedText,
		warning.TMLTime,
		warning.UGC,
		warning.Tornado,
		warning.Damage,
		warning.HailThreat,
		warning.HailTag,
		warning.WindThreat,
		warning.WindTag,
		warning.FlashFlood,
		warning.RainfallTag,
		warning.FloodTagDam,
		warning.SpoutTag,
		warning.SnowSquall,
		warning.SnowSquallTag,
	)
	if err != nil {
		return err
	}

	if rows.Next() {
		if err := ScanWarning(rows, warning); err != nil {
			return err
		}
	}

	return nil
}

func UpdateWarning(db *pgxpool.Pool, warning *models.Warning) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Exec(ctx, `
		UPDATE warnings.warnings SET updated_at = $1, expires = $2, ends = $3, text = $4, action = $5, title = $6, 
    	is_emergency = $7, is_pds = $8, geom = $9, direction = $10, location = $11, speed = $12, 
    	speed_text = $13, tml_time = $14, ugc = $15, tornado = $16, damage = $17, hail_threat = $18, 
    	hail_tag = $19, wind_threat = $20, wind_tag = $21, flash_flood = $22, rainfall_tag = $23, 
    	flood_tag_dam = $24, spout_tag = $25, snow_squall = $26, snow_squall_tag = $27
		WHERE wfo = $28 AND phenomena = $29 AND significance = $30 AND event_number = $31 AND year = $32
		`, time.Now().UTC(), warning.Expires, warning.Ends, warning.Text, warning.Action, warning.Title,
		warning.IsEmergency, warning.IsPDS, warning.Geom, warning.Direction, warning.Location, warning.Speed,
		warning.SpeedText, warning.TMLTime, warning.UGC, warning.Tornado, warning.Damage, warning.HailThreat, warning.HailTag,
		warning.WindThreat, warning.WindTag, warning.FlashFlood, warning.RainfallTag, warning.FloodTagDam,
		warning.SpoutTag, warning.SnowSquall, warning.SnowSquallTag, warning.WFO, warning.Phenomena,
		warning.Significance, warning.EventNumber, warning.Year)
	if err != nil {
		return err
	}

	return nil
}
