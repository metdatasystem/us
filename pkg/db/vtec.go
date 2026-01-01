package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metdatasystem/us/pkg/models"
)

// Find a VTEC event based on VTEC parameters
func FindVTECEvent(db *pgxpool.Pool, wfo string, phenomena string, significance string, eventNumber int, year int) (*models.VTECEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Lets check if the VTEC Event is already in the database
	rows, err := db.Query(ctx, `
			SELECT * FROM vtec.events WHERE
			wfo = $1 AND phenomena = $2 AND significance = $3 AND event_number = $4 AND year = $5
			`, wfo, phenomena, significance, eventNumber, year)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan the row into the event struct
	var event *models.VTECEvent
	if rows.Next() {
		event = &models.VTECEvent{}
		if err := ScanVTECEvent(rows, event); err != nil {
			return nil, err
		}
	}

	return event, nil
}

func FindVTECEventTX(tx pgx.Tx, wfo string, phenomena string, significance string, eventNumber int, year int) (*models.VTECEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Lets check if the VTEC Event is already in the database
	rows, err := tx.Query(ctx, `
			SELECT * FROM vtec.events WHERE
			wfo = $1 AND phenomena = $2 AND significance = $3 AND event_number = $4 AND year = $5
			`, wfo, phenomena, significance, eventNumber, year)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan the row into the event struct
	var event *models.VTECEvent
	if rows.Next() {
		event = &models.VTECEvent{}
		if err := ScanVTECEvent(rows, event); err != nil {
			return nil, err
		}
	}

	return event, nil
}

// Scan a row into the VTEC Event struct
func ScanVTECEvent(rows pgx.Rows, event *models.VTECEvent) error {
	return rows.Scan(
		&event.Phenomena,
		&event.Significance,
		&event.WFO,
		&event.EventNumber,
		&event.Year,
		&event.Class,
		&event.Title,
		&event.IsEmergency,
		&event.IsPDS,
		&event.CreatedAt,
		&event.UpdatedAt,
		&event.Issued,
		&event.Starts,
		&event.Expires,
		&event.Ends,
		&event.EndInitial,
	)
}

// Insert a VTEC event into the database
func InsertVTECEvent(db *pgxpool.Pool, event *models.VTECEvent) error {

	rows, err := db.Query(context.Background(), `
				INSERT INTO vtec.events(issued, starts, expires, ends, ends_initial, class, phenomena, wfo, 
				significance, event_number, year, title, is_emergency, is_pds) VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);
				`, event.Issued, event.Starts, event.Expires, event.Ends, event.EndInitial, event.Class,
		event.Phenomena, event.WFO, event.Significance, event.EventNumber, event.Year, event.Title,
		event.IsEmergency, event.IsPDS)

	if err != nil {
		return err
	}
	defer rows.Close()

	// Scan the row into the event struct
	if rows.Next() {
		if err := ScanVTECEvent(rows, event); err != nil {
			return err
		}
	}

	return nil
}

func ScanVTECUpdate(row pgx.Row, update *models.VTECUpdate) error {
	return row.Scan(
		&update.ID,
		&update.CreatedAt,
		&update.Issued,
		&update.Starts,
		&update.Expires,
		&update.Ends,
		&update.Text,
		&update.Product,
		&update.WFO,
		&update.Action,
		&update.Class,
		&update.Phenomena,
		&update.Significance,
		&update.EventNumber,
		&update.Year,
		&update.Title,
		&update.IsEmergency,
		&update.IsPDS,
		&update.Geom,
		&update.Direction,
		&update.Location,
		&update.Speed,
		&update.SpeedText,
		&update.TMLTime,
		&update.UGC,
		&update.Tornado,
		&update.Damage,
		&update.HailThreat,
		&update.HailTag,
		&update.WindThreat,
		&update.WindTag,
		&update.FlashFlood,
		&update.RainfallTag,
		&update.FloodTagDam,
		&update.SpoutTag,
		&update.SnowSquall,
		&update.SnowSquallTag,
	)
}

func InsertVTECUpdate(db *pgxpool.Pool, update *models.VTECUpdate) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
	INSERT INTO vtec.updates(issued, starts, expires, ends, text, product, 
	wfo, action, class, phenomena, significance, event_number, year, title, 
	is_emergency, is_pds, geom, direction, location, speed, speed_text, tml_time, 
	ugc, tornado, damage, hail_threat, hail_tag, wind_threat, wind_tag, flash_flood, 
	rainfall_tag, flood_tag_dam, spout_tag, snow_squall, snow_squall_tag)
	VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, 
	$19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35)
	`, update.Issued, update.Starts, update.Expires, update.Ends, update.Text, update.Product,
		update.WFO, update.Action, update.Class, update.Phenomena, update.Significance, update.EventNumber, update.Year, update.Title,
		update.IsEmergency, update.IsPDS, update.Geom, update.Direction, update.Location, update.Speed, update.SpeedText, update.TMLTime,
		update.UGC, update.Tornado, update.Damage, update.HailThreat, update.HailTag, update.WindThreat, update.WindTag, update.FlashFlood,
		update.RainfallTag, update.FloodTagDam, update.SpoutTag, update.SnowSquall, update.SnowSquallTag)
	if err != nil {
		return err
	}

	if rows.Next() {
		if err := ScanVTECUpdate(rows, update); err != nil {
			return err
		}
	}

	return nil
}

func FindCurrentVTECEventUGCs(db *pgxpool.Pool, wfo string, phenomena string, significance string, eventNumber int, year int, expires time.Time) ([]*models.VTECUGC, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
	SELECT * FROM vtec.ugcs WHERE wfo = $1 AND phenomena = $2 AND significance = $3 AND event_number = $4 AND year = $5 AND action NOT IN ('CAN', 'UPG') AND expires > $6
	`, wfo, phenomena, significance, eventNumber, year, expires)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ugcs []*models.VTECUGC
	for rows.Next() {
		ugc := &models.VTECUGC{}
		if err := ScanVTECUGC(rows, ugc); err != nil {
			return nil, err
		}
		ugcs = append(ugcs, ugc)
	}

	return ugcs, nil
}

func FindCurrentVTECEventUGCsTX(tx pgx.Tx, wfo string, phenomena string, significance string, eventNumber int, year int, expires time.Time) ([]*models.VTECUGC, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := tx.Query(ctx, `
	SELECT * FROM vtec.ugcs WHERE wfo = $1 AND phenomena = $2 AND significance = $3 AND event_number = $4 AND year = $5 AND action NOT IN ('CAN', 'UPG') AND expires > $6
	`, wfo, phenomena, significance, eventNumber, year, expires)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ugcs []*models.VTECUGC
	for rows.Next() {
		ugc := &models.VTECUGC{}
		if err := ScanVTECUGC(rows, ugc); err != nil {
			return nil, err
		}
		ugcs = append(ugcs, ugc)
	}

	return ugcs, nil
}

func ScanVTECUGC(rows pgx.Rows, ugc *models.VTECUGC) error {
	return rows.Scan(
		&ugc.ID,
		&ugc.CreatedAt,
		&ugc.UpdatedAt,
		&ugc.WFO,
		&ugc.Phenomena,
		&ugc.Significance,
		&ugc.EventNumber,
		&ugc.UGC,
		&ugc.Issued,
		&ugc.Starts,
		&ugc.Expires,
		&ugc.Ends,
		&ugc.EndInitial,
		&ugc.Action,
		&ugc.Year,
	)
}

func InsertVTECUGC(db *pgxpool.Pool, ugc *models.VTECUGC) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
		INSERT INTO vtec.ugcs(wfo, phenomena, significance, event_number, ugc, issued, starts, expires, ends, end_initial, action, year) VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);
		`, ugc.WFO, ugc.Phenomena, ugc.Significance, ugc.EventNumber, ugc.UGC,
		ugc.Issued, ugc.Starts, ugc.Expires, ugc.Ends, ugc.EndInitial,
		ugc.Action, ugc.Year)
	if err != nil {
		return err
	}

	defer rows.Close()

	if rows.Next() {
		if err := ScanVTECUGC(rows, ugc); err != nil {
			return err
		}
	}

	return nil
}

func BulkUpdateUGCsById(db *pgxpool.Pool, ugcs []int, expires time.Time, ends time.Time, action string, wfo string, phenomena string, significance string, eventNumber int, year int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Exec(ctx, `
	UPDATE vtec.ugcs SET expires = $1, ends = $2, action = $3 WHERE
	wfo = $4 AND phenomena = $5 AND significance = $6 AND event_number = $7 AND year = $8
	AND ugc = ANY($9)
	`, expires, ends, action, wfo, phenomena, significance, eventNumber,
		year, ugcs)
	return err
}

func DeleteVTECUGC(db *pgxpool.Pool, ugc *models.VTECUGC) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Exec(ctx, `
		DELETE FROM vtec.ugcs WHERE id = $1
		`, ugc.ID)
	return err
}
