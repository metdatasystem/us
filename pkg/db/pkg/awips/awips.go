package awips

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Create a new product ID based on the product's issuance time, office, WMO datatype, and AWIPS identifier.
// An empty BBB  field can be provided if not applicable.
func GenerateProductID(issued time.Time, office string, wmoDatatype string, awips string, bbb string) string {
	id := fmt.Sprintf("%s-%s-%s-%s", issued.UTC().Format("200601021504"), office, wmoDatatype, awips)

	if len(bbb) > 0 {
		id += "-" + bbb
	}

	return id
}

// AWIPS product
type Product struct {
	ID         int        `json:"id"`
	ProductID  string     `json:"product_id"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
	ReceivedAt *time.Time `json:"received_at"`
	Issued     *time.Time `json:"issued"` // Product issuance time
	Source     string     `json:"source"` // Issuing office or centre
	Data       string     `json:"data"`   // Text data of the product
	WMO        string     `json:"wmo"`
	AWIPS      string     `json:"awips"`
	BBB        string     `json:"bbb"`
}

// Insert an AWIPS product into the database
func (product *Product) Insert(id string, db *pgxpool.Pool) (*Product, error) {

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
func FindProductByID(id int, db *pgxpool.Conn) (*Product, error) {
	if id <= 0 {
		return nil, errors.New("invalid product ID")
	}

	row := db.QueryRow(context.Background(), `
	SELECT * FROM awips.products WHERE id = $1;
	`, id)

	var product *Product

	err := row.Scan()
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return product, nil
}
