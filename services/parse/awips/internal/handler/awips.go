package handler

import (
	"errors"
	"time"

	"github.com/metdatasystem/us/pkg/awips"
	dbAwips "github.com/metdatasystem/us/pkg/db/pkg/awips"
)

type productHandler struct {
	Handler
}

// Inserts the AWIPS product into the database
func (handler *productHandler) Handle(product awips.Product, receivedAt time.Time) (*dbAwips.Product, error) {

	source := product.Office
	bbb := product.WMO.BBB

	if product.Issued.IsZero() {
		return nil, errors.New("product has no issuance time")
	}

	// Generate the product ID
	id := dbAwips.GenerateProductID(product.Issued, product.Office, product.WMO.Datatype, product.AWIPS.Original, product.WMO.BBB)

	// Build the product
	dbProduct := &dbAwips.Product{
		ProductID:  id,
		ReceivedAt: &receivedAt,
		Issued:     &product.Issued,
		Source:     source,
		Data:       product.Text,
		WMO:        product.WMO.Datatype,
		AWIPS:      product.AWIPS.Original,
		BBB:        bbb,
	}

	// Insert the product into the database
	dbProduct, err := dbProduct.Insert(id, handler.db)
	if err != nil {
		return nil, err
	}

	return dbProduct, nil
}
