package internal

import (
	"errors"
	"time"

	"github.com/metdatasystem/us/pkg/awips"
	"github.com/metdatasystem/us/shared/db"
	"github.com/metdatasystem/us/shared/models"
)

type productHandler struct {
	Handler
}

// Inserts the AWIPS product into the database
func (handler *productHandler) Handle(product awips.Product, receivedAt time.Time) (*models.AWIPSProduct, error) {

	source := product.Office
	bbb := product.WMO.BBB

	if product.Issued.IsZero() {
		return nil, errors.New("product has no issuance time")
	}

	// Generate the product ID
	id := models.GenerateAWIPSProductID(product.Issued, product.Office, product.WMO.Datatype, product.AWIPS.Original, product.WMO.BBB)

	// Build the product
	awipsProduct := &models.AWIPSProduct{
		ProductID:  id,
		ReceivedAt: &receivedAt,
		Issued:     &product.Issued,
		Source:     source,
		Data:       product.Text,
		WMO:        product.WMO.Datatype,
		AWIPS:      product.AWIPS.Original,
		BBB:        bbb,
	}

	awipsProduct, err := db.InsertAWIPSProduct(handler.db, id, awipsProduct)
	if err != nil {
		return nil, err
	}

	return awipsProduct, nil
}
