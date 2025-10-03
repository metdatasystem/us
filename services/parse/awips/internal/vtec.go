package internal

import (
	"context"
	"time"

	"github.com/metdatasystem/us/pkg/awips"
	"github.com/metdatasystem/us/pkg/db"
	"github.com/metdatasystem/us/pkg/models"
	"github.com/twpayne/go-geos"
)

type vtecHandler struct {
	Handler
}

func NewVTECHandler(handler *Handler) *vtecHandler {
	return &vtecHandler{*handler}
}

// Handle a VTEC product
func (handler *vtecHandler) Handle() error {

	product := handler.product
	log := handler.log

	// Go through each segment...
	for _, segment := range handler.product.Segments {
		// ...and each VTEC line in the segment
		for _, vtec := range segment.VTEC {
			// Ignore these
			// TODO: Could be helpful?
			if vtec.Class == "T" || vtec.Action == "ROU" {
				continue
			}

			// Find the year of the VTEC event
			// Some VTECs do not come with a start time so we can assume the year from the product issuance time
			// TODO: Deal with end of year events
			var year int
			if vtec.Start != nil {
				year = vtec.Start.Year()
			} else {
				year = product.Issued.Year()
			}

			// VTECs may not have an end time but we will give them one.
			if vtec.End == nil {
				// Use the expiry of the product for the end time
				vtec.End = &segment.Expires
			}

			// Try and find the event in the database
			event, err := db.FindVTECEvent(handler.db, vtec.WFO, vtec.Phenomena, vtec.Significance, vtec.EventNumber, year)
			if err != nil {
				log.Error().Err(err).Msg("failed to find vtec event")
				continue
			}

			// Create the event if one does not exist
			if event == nil {
				log.Debug().Msg("inserting new vtec event")

				event = &models.VTECEvent{
					Issued:       product.Issued,
					Starts:       vtec.Start,
					Expires:      segment.UGC.Expires,
					Ends:         *vtec.End,
					EndInitial:   *vtec.End,
					Class:        vtec.Class,
					Phenomena:    vtec.Phenomena,
					WFO:          vtec.WFO,
					Significance: vtec.Significance,
					EventNumber:  vtec.EventNumber,
					Year:         year,
					Title:        vtec.Title(segment.IsEmergency()),
					IsEmergency:  segment.IsEmergency(),
					IsPDS:        segment.IsPDS(),
				}

				err = db.InsertVTECEvent(handler.db, event)
				if err != nil {
					log.Error().Err(err).Msg("failed to insert vtec event")
					continue
				}
			}

			// Update event timings if needed
			handler.updateTimes(segment, event, vtec)

			isFire := false
			if vtec.Phenomena == "FW" {
				isFire = true
			}

			ugcs, err := GetUGCs(handler.db, segment.UGC, isFire)
			if err != nil {
				log.Error().Err(err).Msg("failed to get ugcs for vtec event")
				continue
			}

			// Create the VTEC update
			err = handler.createUpdate(&segment, event, vtec, ugcs)
			if err != nil {
				log.Error().Err(err).Msg("failed to create vtec update")
				continue
			}

			err = handler.warning(&segment, event, vtec, ugcs)
			if err != nil {
				log.Error().Err(err).Msg("failed to create/update warning")
				continue
			}

			switch vtec.Action {
			case "NEW", "EXB", "EXA":
				err = handler.ugcNew(&segment, event, vtec, ugcs)
			case "CON", "EXP", "ROU", "CAN", "UPG", "EXT", "COR":
				err = handler.ugcUpdate(&segment, event, vtec, ugcs)
			}

			handler.updateEvent(&segment, event, vtec)

		}

	}

	return nil
}

func (handler *vtecHandler) updateTimes(segment awips.ProductSegment, event *models.VTECEvent, vtec awips.VTEC) {
	product := handler.product
	log := handler.log

	// The product expires at the UGC expiry time
	var end time.Time
	if vtec.End == nil {
		end = segment.UGC.Expires
		log.Debug().Msg("VTEC end time is nil. Defaulting to UGC expiry time.")
	} else {
		end = *vtec.End
	}

	switch vtec.Action {
	case "CAN":
		fallthrough
	case "UPG":
		event.Expires = segment.UGC.Expires
		event.Ends = product.Issued.UTC()
	case "EXP":
		event.Expires = end
		event.Ends = end
	case "EXT":
		fallthrough
	case "EXB":
		event.Ends = end
		event.Expires = segment.UGC.Expires
	default:
		// NEW and CON
		if event.Ends.Before(end) {
			event.Ends = end
		}
		if event.Expires.Before(segment.Expires) {
			event.Expires = segment.Expires
		}
	}
}

func (handler *vtecHandler) createUpdate(segment *awips.ProductSegment, event *models.VTECEvent, vtec awips.VTEC, ugcs []*models.UGCMinimal) error {

	product := handler.product
	dbProduct := handler.dbProduct

	ugc := []string{}
	for _, u := range ugcs {
		ugc = append(ugc, u.UGC)
	}

	var polygon *geos.Geom
	if segment.LatLon != nil {
		coords := segment.LatLon.ToFloatClosing()
		polygon = geos.NewPolygon([][][]float64{coords})
	}

	var direction *int
	var locations *geos.Geom
	var speed *int
	var speedText *string
	var tmlTime *time.Time
	if segment.TML != nil {
		direction = &segment.TML.Direction

		points := []*geos.Geom{}
		for _, location := range segment.TML.Locations {
			point := geos.NewPoint(location.FlatCoords())
			points = append(points, point)
		}
		locations = geos.NewCollection(geos.TypeIDMultiPoint, points)

		speed = &segment.TML.Speed
		speedText = &segment.TML.SpeedString
		tmlTime = &segment.TML.Time
	}

	update := &models.VTECUpdate{
		Issued:        product.Issued,
		Starts:        event.Starts,
		Expires:       segment.UGC.Expires,
		Ends:          event.Ends,
		Text:          segment.Text,
		Product:       dbProduct.ProductID,
		WFO:           vtec.WFO,
		Action:        vtec.Action,
		Class:         vtec.Class,
		Phenomena:     vtec.Phenomena,
		Significance:  vtec.Significance,
		EventNumber:   vtec.EventNumber,
		Year:          event.Year,
		Title:         vtec.Title(segment.IsEmergency()),
		IsEmergency:   segment.IsEmergency(),
		IsPDS:         segment.IsPDS(),
		Geom:          polygon,
		Direction:     direction,
		Location:      locations,
		Speed:         speed,
		SpeedText:     speedText,
		TMLTime:       tmlTime,
		UGC:           ugc,
		Tornado:       segment.Tags["tornado"],
		Damage:        segment.Tags["damage"],
		HailThreat:    segment.Tags["hailThreat"],
		HailTag:       segment.Tags["hail"],
		WindThreat:    segment.Tags["windThreat"],
		WindTag:       segment.Tags["wind"],
		FlashFlood:    segment.Tags["flashFlood"],
		RainfallTag:   segment.Tags["expectedRainfall"],
		FloodTagDam:   segment.Tags["damFailure"],
		SpoutTag:      segment.Tags["spout"],
		SnowSquall:    segment.Tags["snowSquall"],
		SnowSquallTag: segment.Tags["snowSquallImpact"],
	}

	err := db.InsertVTECUpdate(handler.db, update)
	if err != nil {
		return err
	}

	return nil
}

func (handler *vtecHandler) ugcNew(segment *awips.ProductSegment, event *models.VTECEvent, vtec awips.VTEC, ugcs []*models.UGCMinimal) error {

	dbProduct := handler.dbProduct
	log := handler.log

	start := event.Starts
	if start != nil {
		if start.Equal(*dbProduct.Issued) {
			start = dbProduct.Issued
		}
	}

	// The product expires at the UGC expiry time
	expires := segment.UGC.Expires
	var end time.Time
	if vtec.End == nil {
		end = expires
		log.Info().Msg("VTEC end time is nil. Defaulting to UGC expiry time.")
	} else {
		end = *vtec.End
	}

	currentUGCs, err := db.FindCurrentVTECEventUGCs(handler.db, event.WFO, event.Phenomena, event.Significance, event.EventNumber, event.Year, expires)
	if err != nil {
		return err
	}

	duplicates := 0
	deleted := 0

	approved := []*models.UGCMinimal{}

	for _, ugc := range ugcs {
		var current *models.VTECUGC
		for _, c := range currentUGCs {
			if ugc.ID == c.UGC {
				current = c
				break
			}
		}
		if current != nil {
			// If the product was reissued as a correction, delete the existing UGC since it may not be valid anymore
			if handler.product.IsCorrection() && current.Action == vtec.Action {
				db.DeleteVTECUGC(handler.db, current)
				deleted++
			}
			duplicates++
		} else {
			approved = append(approved, ugc)
		}
	}

	if duplicates > 0 {
		log.Warn().Int("duplicates", duplicates).Int("deleted", deleted).Bool("correction", handler.product.IsCorrection()).Str("vtec", vtec.Original).Msg("found existing ugcs for vtec event")
	}

	for _, ugc := range approved {
		newUGC := &models.VTECUGC{
			WFO:          event.WFO,
			Phenomena:    event.Phenomena,
			Significance: event.Significance,
			EventNumber:  event.EventNumber,
			UGC:          ugc.ID,
			Issued:       event.Issued,
			Starts:       start,
			Expires:      expires,
			Ends:         end,
			EndInitial:   end,
			Action:       vtec.Action,
			Year:         event.Year,
		}

		err = db.InsertVTECUGC(handler.db, newUGC)
		if err != nil {
			log.Error().Err(err).Msg("failed to insert new vtec ugc")
		}
	}

	return nil
}

func (handler *vtecHandler) ugcUpdate(segment *awips.ProductSegment, event *models.VTECEvent, vtec awips.VTEC, ugcs []*models.UGCMinimal) error {

	expires := segment.UGC.Expires
	end := event.Ends

	u := []int{}
	for _, ugc := range ugcs {
		u = append(u, ugc.ID)
	}

	return db.BulkUpdateUGCsById(handler.db, u, expires, end, vtec.Action, event.WFO, event.Phenomena, event.Significance, event.EventNumber, event.Year)
}

func (handler *vtecHandler) updateEvent(segment *awips.ProductSegment, event *models.VTECEvent, vtec awips.VTEC) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := handler.db.Exec(ctx, `
	UPDATE vtec.events SET updated_at = CURRENT_TIMESTAMP, is_emergency = $6, is_pds = $7 WHERE
			wfo = $1 AND phenomena = $2 AND significance = $3 AND event_number = $4 AND year = $5
			`, vtec.WFO, vtec.Phenomena, vtec.Significance, vtec.EventNumber, event.Year, segment.IsEmergency(), segment.IsPDS())
	if err != nil {
		handler.log.Warn().Err(err).Str("vtec", vtec.Original).Msg("failed to update vtec event")
		return
	}
}
