package internal

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metdatasystem/us/pkg/awips"
	"github.com/metdatasystem/us/shared/db"
	"github.com/metdatasystem/us/shared/models"
)

func GetUGCs(dbPool *pgxpool.Pool, ugcList *awips.UGC, isFire bool) ([]*models.UGCMinimal, error) {
	ugcs := []*models.UGCMinimal{}

	// For each state...
	for _, state := range ugcList.States {
		ugcType := state.Type
		// ...and for each area...
		for _, area := range state.Areas {
			if isFire {
				ugcType = "F"
			}

			if area == "000" || area == "ALL" {
				u, err := db.GetUGCForStateMinimal(dbPool, state.ID, ugcType)
				if err != nil {
					return nil, err
				}

				ugcs = append(ugcs, u...)
			} else {
				ugcCode := state.ID + ugcType + area
				u, err := db.FindUGCByCodeMinimal(dbPool, ugcCode)
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
