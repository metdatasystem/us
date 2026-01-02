package models

import (
	"fmt"
	"time"
)

type AWIPSProduct struct {
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
func GenerateAWIPSProductID(issued time.Time, office string, wmoDatatype string, awips string, bbb string) string {
	id := fmt.Sprintf("%s-%s-%s-%s", issued.UTC().Format("200601021504"), office, wmoDatatype, awips)

	if len(bbb) > 0 {
		id += "-" + bbb
	}

	return id
}
