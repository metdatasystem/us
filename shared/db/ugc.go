package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metdatasystem/us/shared/models"
	"github.com/twpayne/go-geos"
)

// Find a UGC by its code (e.g. "TXZ123" or "TXF123" for fire zones)
func FindUGCByCode(db *pgxpool.Pool, ugcCode string) (*models.UGC, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := db.QueryRow(ctx, `
	SELECT * FROM postgis.ugcs WHERE ugc = $1 AND valid_to IS NULL
	`, ugcCode)

	ugc := &models.UGC{}
	if err := ScanUGC(row, ugc); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return ugc, nil
}

// Find a UGC by its code (e.g. "TXZ123" or "TXF123" for fire zones)
// but does not return geometry or area data
func FindUGCByCodeMinimal(db *pgxpool.Pool, ugcCode string) (*models.UGCMinimal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	row := db.QueryRow(ctx, `
	SELECT id, ugc, name, state, type, number, cwa, is_marine, is_fire, valid_from, valid_to
	FROM postgis.ugcs WHERE ugc = $1 AND valid_to IS NULL
	`, ugcCode)

	ugc := &models.UGCMinimal{}
	if err := ScanUGCMinimal(row, ugc); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return ugc, nil
}

func GetAllValidUGCMinimal(db *pgxpool.Pool) ([]*models.UGCMinimal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
	SELECT id, ugc, name, state, type, number, cwa, is_marine, is_fire, valid_from, valid_to FROM postgis.ugcs WHERE valid_to IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ugcs []*models.UGCMinimal
	for rows.Next() {
		ugc := &models.UGCMinimal{}
		if err := ScanUGCMinimal(rows, ugc); err != nil {
			return nil, err
		}
		ugcs = append(ugcs, ugc)
	}

	return ugcs, nil
}

// Get all UGCs for a given state
func GetUGCForState(db *pgxpool.Pool, state string, ugcType string) ([]*models.UGC, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
	SELECT * FROM postgis.ugcs WHERE state = $1 AND type = $2 AND valid_to IS NULL
	`, state, ugcType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ugcs []*models.UGC
	for rows.Next() {
		ugc := &models.UGC{}
		if err := ScanUGC(rows, ugc); err != nil {
			return nil, err
		}
		ugcs = append(ugcs, ugc)
	}

	return ugcs, nil
}

// Get all UGCs for a given state but without geometry or area data
func GetUGCForStateMinimal(db *pgxpool.Pool, state string, ugcType string) ([]*models.UGCMinimal, error) {
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

	var ugcs []*models.UGCMinimal
	for rows.Next() {
		ugc := &models.UGCMinimal{}
		if err := ScanUGCMinimal(rows, ugc); err != nil {
			return nil, err
		}
		ugcs = append(ugcs, ugc)
	}

	return ugcs, nil
}

func GetUGCUnionGeomSimplified(db *pgxpool.Pool, ugcs []string) (*geos.Geom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, `
	SELECT ST_Simplify(ST_Union(geom), 0.0025) FROM postgis.ugcs WHERE valid_to IS NULL AND ugc = ANY($1)
	`, ugcs)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var geom *geos.Geom
	if rows.Next() {
		geom = &geos.Geom{}
		if err := rows.Scan(&geom); err != nil {
			return nil, err
		}
	}

	return geom, nil
}

func GetUGCUnionGeomSimplifiedTx(tx pgx.Tx, ugcs []string) (*geos.Geom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := tx.Query(ctx, `
	SELECT ST_Simplify(ST_Union(geom), 0.0025) FROM postgis.ugcs WHERE valid_to IS NULL AND ugc = ANY($1)
	`, ugcs)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var geom *geos.Geom
	if rows.Next() {
		geom = &geos.Geom{}
		if err := rows.Scan(&geom); err != nil {
			return nil, err
		}
	}

	return geom, nil
}

func ScanUGC(row pgx.Row, ugc *models.UGC) error {
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

func ScanUGCMinimal(row pgx.Row, ugc *models.UGCMinimal) error {
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
