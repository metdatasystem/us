package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/metdatasystem/us/pkg/awips"
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/ewkbhex"
)

type vtecEvent struct {
	ID           int        `json:"id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	Issued       time.Time  `json:"issued"`
	Starts       *time.Time `json:"starts"`
	Expires      time.Time  `json:"expires"`
	Ends         time.Time  `json:"ends"`
	EndInitial   time.Time  `json:"end_initial"`
	Class        string     `json:"class"`
	Phenomena    string     `json:"phenomena"`
	WFO          string     `json:"wfo"`
	Significance string     `json:"significance"`
	EventNumber  int        `json:"event_number"`
	Year         int        `json:"year"`
	Title        string     `json:"title"`
	IsEmergency  bool       `json:"is_emergency"`
	IsPDS        bool       `json:"is_pds"`
}

type vtecUpdate struct {
	ID            int        `json:"id"`
	CreatedAt     time.Time  `json:"created_at,omitempty"`
	Issued        time.Time  `json:"issued"`
	Starts        *time.Time `json:"starts,omitempty"`
	Expires       time.Time  `json:"expires"`
	Ends          time.Time  `json:"ends,omitempty"`
	Text          string     `json:"text"`
	Product       string     `json:"product"`
	WFO           string     `json:"wfo"`
	Action        string     `json:"action"`
	Class         string     `json:"class"`
	Phenomena     string     `json:"phenomena"`
	Significance  string     `json:"significance"`
	EventNumber   int        `json:"event_number"`
	Year          int        `json:"year"`
	Title         string     `json:"title"`
	IsEmergency   bool       `json:"is_emergency"`
	IsPDS         bool       `json:"is_pds"`
	Geom          []byte     `json:"geom,omitempty"`
	Direction     *int       `json:"direction"`
	Location      []byte     `json:"location,omitempty"`
	Speed         *int       `json:"speed"`
	SpeedText     *string    `json:"speed_text"`
	TMLTime       *time.Time `json:"tml_time"`
	UGC           []string   `json:"ugc"`
	Tornado       string     `json:"tornado,omitempty"`
	Damage        string     `json:"damage,omitempty"`
	HailThreat    string     `json:"hail_threat,omitempty"`
	HailTag       string     `json:"hail_tag,omitempty"`
	WindThreat    string     `json:"wind_threat,omitempty"`
	WindTag       string     `json:"wind_tag,omitempty"`
	FlashFlood    string     `json:"flash_flood,omitempty"`
	RainfallTag   string     `json:"rainfall_tag,omitempty"`
	FloodTagDam   string     `json:"flood_tag_dam,omitempty"`
	SpoutTag      string     `json:"spout_tag,omitempty"`
	SnowSquall    string     `json:"snow_squall,omitempty"`
	SnowSquallTag string     `json:"snow_squall_tag,omitempty"`
}

type vtecUGC struct {
	ID           int        `json:"id"`
	CreatedAt    time.Time  `json:"created_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at,omitempty"`
	WFO          string     `json:"wfo"`
	Phenomena    string     `json:"phenomena"`
	Significance string     `json:"significance"`
	EventNumber  int        `json:"event_number"`
	UGC          int        `json:"ugc"`
	Issued       time.Time  `json:"issued"`
	Starts       *time.Time `json:"starts,omitempty"`
	Expires      time.Time  `json:"expires"`
	Ends         time.Time  `json:"ends,omitempty"`
	EndInitial   time.Time  `json:"end_initial,omitempty"`
	Action       string     `json:"action"`
	Year         int        `json:"year"`
}

type vtecHandler struct {
	Handler
	ctx context.Context
	tx  pgx.Tx

	publishedWarnings map[string]struct{}
	// errorCollecting *ErrorCollector
}

func NewVTECHandler(handler *Handler) *vtecHandler {
	return &vtecHandler{*handler, context.Background(), nil, map[string]struct{}{}}
}

// Handle a VTEC product
func (handler *vtecHandler) Handle() error {

	product := handler.product
	log := handler.log

	var err error

	// Initialise transaction
	handler.tx, err = handler.db.BeginTx(handler.ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err.Error())
	}
	defer handler.tx.Rollback(handler.ctx)

	// Go through each segment...
	for _, segment := range handler.product.Segments {
		// ...and each VTEC in the segment
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
			event, err := handler.findEvent(vtec.WFO, vtec.Phenomena, vtec.Significance, vtec.EventNumber, year)
			if err != nil {
				log.Error().Err(err).Msg("failed to find vtec event")
				continue
			}

			// Create the event if one does not exist
			if event == nil {
				log.Debug().Msg("inserting new vtec event")

				event, err = handler.createEvent(vtec, segment, year)
				if err != nil {
					log.Error().Err(err).Msg("failed to insert vtec event")
					continue
				}
			}

			// Update event timings if needed
			handler.updateTimes(segment, event, vtec)

			// Some special stuff needs to be done for fire weather events
			isFire := false
			if vtec.Phenomena == "FW" {
				isFire = true
			}

			ugcs, err := getUGCs(handler.ctx, handler.tx, segment.UGC, isFire)
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
			if err != nil {
				log.Error().Err(err).Msg("failed to create/update ugcs")
				continue
			}

			handler.updateEvent(&segment, event, vtec)

		}

	}

	if err := handler.tx.Commit(handler.ctx); err != nil {
		return err
	}

	return nil
}

func (handler *vtecHandler) updateTimes(segment awips.ProductSegment, event *vtecEvent, vtec awips.VTEC) {
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

func (handler *vtecHandler) findEvent(wfo string, phenomena string, significance string, eventNumber int, year int) (*vtecEvent, error) {
	rows, err := handler.tx.Query(handler.ctx, `
			SELECT * FROM vtec.events WHERE
			wfo = $1 AND phenomena = $2 AND significance = $3 AND event_number = $4 AND year = $5
			`, wfo, phenomena, significance, eventNumber, year)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	// Scan the row into the event struct
	var event *vtecEvent
	if rows.Next() {
		event = &vtecEvent{}

		if err :=
			rows.Scan(
				&event.Phenomena,
				&event.Significance,
				&event.WFO,
				&event.EventNumber,
				&event.Year,
				&event.Class,
				&event.Title,
				&event.IsEmergency,
				&event.IsPDS,
				&event.CreatedAt,
				&event.UpdatedAt,
				&event.Issued,
				&event.Starts,
				&event.Expires,
				&event.Ends,
				&event.EndInitial,
			); err != nil {
			return nil, fmt.Errorf("failed to scan vtec event: %v", err.Error())
		}
	}

	return event, nil
}

func (handler *vtecHandler) createEvent(vtec awips.VTEC, segment awips.ProductSegment, year int) (*vtecEvent, error) {
	product := handler.product

	event := &vtecEvent{
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

	_, err := handler.tx.Exec(handler.ctx, `
				INSERT INTO vtec.events(issued, starts, expires, ends, ends_initial, class, phenomena, wfo, 
				significance, event_number, year, title, is_emergency, is_pds) VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);
				`, event.Issued, event.Starts, event.Expires, event.Ends, event.EndInitial, event.Class,
		event.Phenomena, event.WFO, event.Significance, event.EventNumber, event.Year, event.Title,
		event.IsEmergency, event.IsPDS)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func (handler *vtecHandler) updateEvent(segment *awips.ProductSegment, event *vtecEvent, vtec awips.VTEC) {

	_, err := handler.tx.Exec(handler.ctx, `
	UPDATE vtec.events SET updated_at = CURRENT_TIMESTAMP, is_emergency = $6, is_pds = $7 WHERE
			wfo = $1 AND phenomena = $2 AND significance = $3 AND event_number = $4 AND year = $5
			`, vtec.WFO, vtec.Phenomena, vtec.Significance, vtec.EventNumber, event.Year, segment.IsEmergency(), segment.IsPDS())
	if err != nil {
		handler.log.Warn().Err(err).Str("vtec", vtec.Original).Msg("failed to update vtec event")
		return
	}
}

func (handler *vtecHandler) findCurrentVTECEventUGCs(wfo string, phenomena string, significance string, eventNumber int, year int, expires time.Time) ([]*vtecUGC, error) {

	rows, err := handler.tx.Query(handler.ctx, `
	SELECT * FROM vtec.ugcs WHERE wfo = $1 AND phenomena = $2 AND significance = $3 AND event_number = $4 AND year = $5 AND action NOT IN ('CAN', 'UPG') AND expires > $6
	`, wfo, phenomena, significance, eventNumber, year, expires)
	if err != nil {
		return nil, fmt.Errorf("failed to query vtec ugc: %v", err.Error())
	}
	defer rows.Close()

	var ugcs []*vtecUGC
	for rows.Next() {
		ugc := &vtecUGC{}
		if err := rows.Scan(
			&ugc.ID,
			&ugc.CreatedAt,
			&ugc.UpdatedAt,
			&ugc.Phenomena,
			&ugc.Significance,
			&ugc.WFO,
			&ugc.EventNumber,
			&ugc.Year,
			&ugc.UGC,
			&ugc.Issued,
			&ugc.Starts,
			&ugc.Expires,
			&ugc.Ends,
			&ugc.EndInitial,
			&ugc.Action,
		); err != nil {
			return nil, fmt.Errorf("failed to scan vtec ugc: %v", err.Error())
		}
		ugcs = append(ugcs, ugc)
	}

	return ugcs, rows.Err()
}

// Assembles and creates a VTEC Update.
func (handler *vtecHandler) createUpdate(segment *awips.ProductSegment, event *vtecEvent, vtec awips.VTEC, ugcs []*ugcMinimal) error {

	product := handler.product
	dbProduct := handler.dbProduct

	ugc := []string{}
	for _, u := range ugcs {
		ugc = append(ugc, u.UGC)
	}

	var err error
	var polygon []byte
	if segment.LatLon != nil {
		p, err := segment.LatLon.ToPolygon()
		if err != nil {
			return fmt.Errorf("failed to get latlon polygon: %v", err.Error())
		}
		polygon, err = ewkb.Marshal(p, ewkbhex.NDR)
		if err != nil {
			return fmt.Errorf("failed to encode polygon: %v", err.Error())
		}
	}

	var direction *int
	var locations []byte
	var speed *int
	var speedText *string
	var tmlTime *time.Time
	if segment.TML != nil {
		direction = &segment.TML.Direction
		l := segment.TML.Locations
		locations, err = ewkb.Marshal(l, ewkb.NDR)
		if err != nil {
			return fmt.Errorf("failed to encode tml locations: %v", err.Error())
		}
		speed = &segment.TML.Speed
		speedText = &segment.TML.SpeedString
		tmlTime = &segment.TML.Time
	}

	update := &vtecUpdate{
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

	_, err = handler.tx.Exec(handler.ctx, `
	INSERT INTO vtec.updates(issued, starts, expires, ends, text, product, 
	wfo, action, class, phenomena, significance, event_number, year, title, 
	is_emergency, is_pds, geom, direction, location, speed, speed_text, tml_time, 
	ugc, tornado, damage, hail_threat, hail_tag, wind_threat, wind_tag, flash_flood, 
	rainfall_tag, flood_tag_dam, spout_tag, snow_squall, snow_squall_tag)
	VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, ST_GeomFromWKB($17, 4326), $18, 
	ST_GeomFromWKB($19, 4326), $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35)
	`, update.Issued, update.Starts, update.Expires, update.Ends, update.Text, update.Product,
		update.WFO, update.Action, update.Class, update.Phenomena, update.Significance, update.EventNumber, update.Year, update.Title,
		update.IsEmergency, update.IsPDS, update.Geom, update.Direction, update.Location, update.Speed, update.SpeedText, update.TMLTime,
		update.UGC, update.Tornado, update.Damage, update.HailThreat, update.HailTag, update.WindThreat, update.WindTag, update.FlashFlood,
		update.RainfallTag, update.FloodTagDam, update.SpoutTag, update.SnowSquall, update.SnowSquallTag)
	if err != nil {
		return fmt.Errorf("vtec update query failed: %v", err.Error())
	}

	return nil
}

func (handler *vtecHandler) ugcNew(segment *awips.ProductSegment, event *vtecEvent, vtec awips.VTEC, ugcs []*ugcMinimal) error {

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

	currentUGCs, err := handler.findCurrentVTECEventUGCs(event.WFO, event.Phenomena, event.Significance, event.EventNumber, event.Year, expires)
	if err != nil {
		return fmt.Errorf("failed to find current vtec ugcs: %v", err.Error())
	}

	duplicates := 0
	deleted := 0

	approved := []*ugcMinimal{}

	for _, ugc := range ugcs {
		var current *vtecUGC
		for _, c := range currentUGCs {
			if ugc.ID == c.UGC {
				current = c
				break
			}
		}
		if current != nil {
			// If the product was reissued as a correction, delete the existing UGC since it may not be valid anymore
			if handler.product.IsCorrection() && current.Action == vtec.Action {
				_, err := handler.tx.Exec(handler.ctx, `
							DELETE FROM vtec.ugcs WHERE id = $1
							`, ugc.ID)
				if err != nil {
					log.Error().Err(err).Msg("failed to delete vtec.ugc entry")
				}
				deleted++
			}
			duplicates++
		} else {
			approved = append(approved, ugc)
		}
	}

	if duplicates > 0 {
		log.Warn().Int("duplicates", duplicates).Int("deleted", deleted).Bool("correction", handler.product.IsCorrection()).Str("vtec", vtec.Original).Msg("found existing ugcs for new vtec event")
		return nil
	}

	for _, ugc := range approved {
		newUGC := &vtecUGC{
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

		_, err := handler.tx.Exec(handler.ctx, `
		INSERT INTO vtec.ugcs(wfo, phenomena, significance, event_number, ugc, issued, starts, expires, ends, end_initial, action, year) VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);
		`, newUGC.WFO, newUGC.Phenomena, newUGC.Significance, newUGC.EventNumber, newUGC.UGC,
			newUGC.Issued, newUGC.Starts, newUGC.Expires, newUGC.Ends, newUGC.EndInitial,
			newUGC.Action, newUGC.Year)
		if err != nil {
			log.Error().Err(err).Msg("failed to insert new vtec ugc")
		}
	}

	return nil
}

func (handler *vtecHandler) ugcUpdate(segment *awips.ProductSegment, event *vtecEvent, vtec awips.VTEC, ugcs []*ugcMinimal) error {

	expires := segment.UGC.Expires
	end := event.Ends

	u := []int{}
	for _, ugc := range ugcs {
		u = append(u, ugc.ID)
	}

	_, err := handler.tx.Exec(handler.ctx, `
	UPDATE vtec.ugcs SET expires = $1, ends = $2, action = $3 WHERE
	wfo = $4 AND phenomena = $5 AND significance = $6 AND event_number = $7 AND year = $8
	AND ugc = ANY($9)
	`, expires, end, vtec.Action, event.WFO, event.Phenomena, event.Significance, event.EventNumber,
		event.Year, u)

	return err
}
