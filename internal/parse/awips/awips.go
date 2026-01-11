package internal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/metdatasystem/us/pkg/awips"
)

type awipsProduct struct {
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

// Create a new product ID based on the product's issuance time, office, WMO datatype, and AWIPS identifier.
// An empty BBB  field can be provided if not applicable.
func generateAWIPSProductID(issued time.Time, office string, wmoDatatype string, awips string, bbb string) string {
	id := fmt.Sprintf("%s-%s-%s-%s", issued.UTC().Format("200601021504"), office, wmoDatatype, awips)

	if len(bbb) > 0 {
		id += "-" + bbb
	}

	return id
}

type productHandler struct {
	Handler
}

// Inserts the AWIPS product into the database
func (handler *productHandler) Handle(product awips.Product, receivedAt time.Time) (*awipsProduct, error) {

	source := product.Office
	bbb := product.WMO.BBB

	if product.Issued.IsZero() {
		return nil, errors.New("product has no issuance time")
	}

	// Generate the product ID
	id := generateAWIPSProductID(product.Issued, product.Office, product.WMO.Datatype, product.AWIPS.Original, product.WMO.BBB)

	// Build the product
	awipsProduct := &awipsProduct{
		ProductID:  id,
		ReceivedAt: &receivedAt,
		Issued:     &product.Issued,
		Source:     source,
		Data:       product.Text,
		WMO:        product.WMO.Datatype,
		AWIPS:      product.AWIPS.Original,
		BBB:        bbb,
	}

	rows, err := handler.db.Query(context.Background(), `
	INSERT INTO awips.products (product_id, received_at, issued, source, data, wmo, awips, bbb) VALUES
	($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at;
	`, id, awipsProduct.ReceivedAt, product.Issued, awipsProduct.Source, awipsProduct.Data, awipsProduct.WMO, awipsProduct.AWIPS, awipsProduct.BBB)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("no rows returned after creating awips product")
	}
	err = rows.Scan(&awipsProduct.ID, &awipsProduct.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan awips product: %v", err.Error())
	}
	if !rows.Next() && rows.Err() != nil {
		return nil, rows.Err()
	}

	return awipsProduct, nil
}
