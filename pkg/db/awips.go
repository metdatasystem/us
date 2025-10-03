package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metdatasystem/us/pkg/models"
)

// Insert an AWIPS product into the database
func InsertAWIPSProduct(db *pgxpool.Pool, id string, product *models.AWIPSProduct) (*models.AWIPSProduct, error) {

	rows, err := db.Query(context.Background(), `
	INSERT INTO awips.products (product_id, received_at, issued, source, data, wmo, awips, bbb) VALUES
	($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at;
	`, id, product.ReceivedAt, product.Issued, product.Source, product.Data, product.WMO, product.AWIPS, product.BBB)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&product.ID, &product.CreatedAt)
		return product, err
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return nil, errors.New("no rows returned when creating new text product: " + rows.Err().Error())
}

// Find an AWIPS product by its numeric ID. Nil is returned if no product is found.
func FindProductByID(id int, db *pgxpool.Conn) (*models.AWIPSProduct, error) {
	if id <= 0 {
		return nil, errors.New("invalid product ID")
	}

	row := db.QueryRow(context.Background(), `
	SELECT * FROM awips.products WHERE id = $1;
	`, id)

	var product *models.AWIPSProduct

	err := row.Scan()
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return product, nil
}
