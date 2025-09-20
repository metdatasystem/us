package handler

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metdatasystem/us/pkg/awips"
	"github.com/metdatasystem/us/pkg/db/pkg/postgis"
)

func GetUGCs(db *pgxpool.Pool, ugcList *awips.UGC, isFire bool) ([]*postgis.UGCMinimal, error) {
	ugcs := []*postgis.UGCMinimal{}

	// For each state...
	for _, state := range ugcList.States {
		ugcType := state.Type
		// ...and for each area...
		for _, area := range state.Areas {
			if isFire {
				ugcType = "F"
			}

			if area == "000" || area == "ALL" {
				u, err := postgis.GetUGCForStateMinimal(db, state.ID, ugcType)
				if err != nil {
					return nil, err
				}

				ugcs = append(ugcs, u...)
			} else {
				ugcCode := state.ID + ugcType + area
				u, err := postgis.FindUGCByCodeMinimal(db, ugcCode)
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
