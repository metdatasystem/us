package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/metdatasystem/us/pkg/awips"
	"github.com/metdatasystem/us/shared/streaming"
	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
	"github.com/twpayne/go-geom/encoding/ewkb"
)

type warning struct {
	ID             int        `json:"id"`
	Phenomena      string     `json:"phenomena"`
	Significance   string     `json:"significance"`
	WFO            string     `json:"wfo"`
	EventNumber    int        `json:"event_number"`
	Year           int        `json:"year"`
	Action         string     `json:"action"`
	Current        bool       `json:"current"`
	CreatedAt      time.Time  `json:"created_at,omitzero"`
	UpdatedAt      time.Time  `json:"updated_at,omitzero"`
	Issued         time.Time  `json:"issued"`
	Starts         *time.Time `json:"starts,omitzero"`
	Expires        time.Time  `json:"expires"`
	ExpiresInitial time.Time  `json:"expires_initial,omitzero"`
	Ends           time.Time  `json:"ends,omitzero"`
	Class          string     `json:"class"`
	Title          string     `json:"title"`
	IsEmergency    bool       `json:"is_emergency"`
	IsPDS          bool       `json:"is_pds"`
	Text           string     `json:"text"`
	Product        string     `json:"product"`
	Geom           []byte     `json:"geom"`
	Direction      *int       `json:"direction"`
	Locations      []byte     `json:"locations"`
	Speed          *int       `json:"speed"`
	SpeedText      *string    `json:"speed_text"`
	TMLTime        *time.Time `json:"tml_time"`
	UGC            []string   `json:"ugc"`
	Tornado        string     `json:"tornado,omitempty"`
	Damage         string     `json:"damage,omitempty"`
	HailThreat     string     `json:"hail_threat,omitempty"`
	HailTag        string     `json:"hail_tag,omitempty"`
	WindThreat     string     `json:"wind_threat,omitempty"`
	WindTag        string     `json:"wind_tag,omitempty"`
	FlashFlood     string     `json:"flash_flood,omitempty"`
	RainfallTag    string     `json:"rainfall_tag,omitempty"`
	FloodTagDam    string     `json:"flood_tag_dam,omitempty"`
	SpoutTag       string     `json:"spout_tag,omitempty"`
	SnowSquall     string     `json:"snow_squall,omitempty"`
	SnowSquallTag  string     `json:"snow_squall_tag,omitempty"`
}

// Generates an ID using the warning's WFO, phenomena, significance, event number, and year.
//
// Example: KOUN-SV-W-0001-2025
func (warning *warning) GenerateID() string {
	return fmt.Sprintf("%v-%v-%v-%04v-%v", warning.WFO, warning.Phenomena, warning.Significance, warning.EventNumber, warning.Year)
}

// Generates an ID using the warning's generated ID from [warning.GenerateID()], appending the unique integer ID from the database.
//
// Example: KOUN-SV-W-0001-2025-1
func (warning *warning) GenerateCompositeID() string {
	return fmt.Sprintf("%s-%v", warning.GenerateID(), warning.ID)
}

func (handler *vtecHandler) warning(segment *awips.ProductSegment, event *vtecEvent, vtec awips.VTEC, ugcs []*ugcMinimal) error {
	product := handler.product

	yesterday := time.Now().Add(time.Hour * -24)

	if event.Ends.Before(yesterday) {
		return nil
	}

	ugcList := []string{}
	for _, u := range ugcs {
		ugcList = append(ugcList, u.UGC)
	}

	var (
		err  error
		geom []byte
	)
	if segment.LatLon != nil {
		polygon, err := segment.LatLon.ToMultiPolygon()
		if err != nil {
			return fmt.Errorf("failed to get latlon polygon: %v", err.Error())
		}
		geom, err = ewkb.Marshal(polygon, ewkb.NDR)
		if err != nil {
			return fmt.Errorf("failed to marshal polygon: %v", err.Error())
		}
	} else if vtec.Action != "CAN" && vtec.Action != "UPG" && vtec.Action != "EXP" {
		rows, err := handler.tx.Query(handler.ctx, `
			SELECT ST_AsBinary(ST_SimplifyPreserveTopology(ST_Union(ST_MakeValid(geom)), 0.0025)) FROM postgis.ugcs WHERE valid_to IS NULL AND ugc = ANY($1)
			`, ugcList)
		if err != nil {
			return err
		}

		if rows.Next() {
			b := []byte{}
			if err := rows.Scan(&b); err != nil {
				return err
			}
			geom = b
		}

		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("error getting simplified geometry: %v", err.Error())
		}

		rows.Close()

	}

	var (
		direction *int
		locations []byte
		speed     *int
		speedText *string
		tmlTime   *time.Time
	)
	if segment.TML != nil {
		direction = &segment.TML.Direction
		locations, err = ewkb.Marshal(segment.TML.Locations, ewkb.NDR)
		if err != nil {
			return fmt.Errorf("failed to marshal locations: %v", err.Error())
		}
		speed = &segment.TML.Speed
		speedText = &segment.TML.SpeedString
		tmlTime = &segment.TML.Time
	}

	warning := &warning{
		Issued:         product.Issued,
		Starts:         event.Starts,
		Expires:        segment.UGC.Expires,
		Ends:           event.Ends,
		ExpiresInitial: segment.UGC.Expires,
		Text:           segment.Text,
		Product:        handler.dbProduct.ProductID,
		WFO:            vtec.WFO,
		Action:         vtec.Action,
		Class:          vtec.Class,
		Phenomena:      vtec.Phenomena,
		Significance:   vtec.Significance,
		EventNumber:    vtec.EventNumber,
		Year:           event.Year,
		Title:          vtec.Title(segment.IsEmergency()),
		IsEmergency:    segment.IsEmergency(),
		IsPDS:          segment.IsPDS(),
		Geom:           geom,
		Direction:      direction,
		Locations:      locations,
		Speed:          speed,
		SpeedText:      speedText,
		TMLTime:        tmlTime,
		UGC:            ugcList,
		Tornado:        segment.Tags["tornado"],
		Damage:         segment.Tags["damage"],
		HailThreat:     segment.Tags["hailThreat"],
		HailTag:        segment.Tags["hail"],
		WindThreat:     segment.Tags["windThreat"],
		WindTag:        segment.Tags["wind"],
		FlashFlood:     segment.Tags["flashFlood"],
		RainfallTag:    segment.Tags["expectedRainfall"],
		FloodTagDam:    segment.Tags["damFailure"],
		SpoutTag:       segment.Tags["spout"],
		SnowSquall:     segment.Tags["snowSquall"],
		SnowSquallTag:  segment.Tags["snowSquallImpact"],
	}

	if _, ok := handler.publishedWarnings[warning.GenerateID()]; !ok {
		rows, err := handler.tx.Query(handler.ctx, `
				UPDATE warnings.warnings SET expires_initial = $1, current = false, updated_at = CURRENT_TIMESTAMP
				WHERE phenomena = $2 AND significance = $3 AND wfo = $4 AND event_number = $5 AND year = $6 AND current = true RETURNING id
			`, warning.Issued, warning.Phenomena, warning.Significance, warning.WFO, warning.EventNumber, warning.Year)
		if err != nil {
			return err
		}

		for rows.Next() {
			var id int
			err := rows.Scan(&id)
			if err != nil {
				log.Error().Err(err).Msg("failed to scan warning ID from update")
				continue
			}

			tempW := *warning
			tempW.ID = id
			data, err := json.Marshal(tempW)
			if err != nil {
				log.Error().Err(err).Msg("failed to marshal warning to publish from update")
				continue
			}

			err = handler.rabbit.PublishWithContext(context.Background(),
				streaming.ExchangeLiveName,
				"warning",
				false,
				false,
				amqp091.Publishing{
					ContentType: "application/json",
					MessageId:   tempW.GenerateCompositeID(),
					Timestamp:   time.Now(),
					Type:        streaming.EventDelete,
					AppId:       "us.parse.awips",
					Body:        data,
				},
			)
			if err != nil {
				log.Error().Err(err).Msg("failed to publish warning delete")
				continue
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("failed to update warning: %v", err.Error())
		}
		rows.Close()
	}

	current := true
	if warning.Action == "CAN" || warning.Action == "UPG" || warning.Action == "EXP" {
		current = false
	}

	rows, err := handler.tx.Query(handler.ctx, `
       INSERT INTO warnings.warnings(
	       issued, starts, expires, ends, expires_initial, text, product, 
		   wfo, action, current, class, phenomena, significance, event_number, year, 
		   title, is_emergency, is_pds, geom, direction, location, speed, speed_text, tml_time, 
		   ugc, tornado, damage, hail_threat, hail_tag, wind_threat, wind_tag, flash_flood, 
		   rainfall_tag, flood_tag_dam, spout_tag, snow_squall, snow_squall_tag
       ) VALUES (
	       $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, 
		   ST_GeomFromWKB($19, 4326), $20, ST_GeomFromWKB($21, 4326), $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37
       ) RETURNING id
       `,
		warning.Issued,
		warning.Starts,
		warning.Expires,
		warning.Ends,
		warning.ExpiresInitial,
		warning.Text,
		warning.Product,
		warning.WFO,
		warning.Action,
		current,
		warning.Class,
		warning.Phenomena,
		warning.Significance,
		warning.EventNumber,
		warning.Year,
		warning.Title,
		warning.IsEmergency,
		warning.IsPDS,
		warning.Geom,
		warning.Direction,
		warning.Locations,
		warning.Speed,
		warning.SpeedText,
		warning.TMLTime,
		warning.UGC,
		warning.Tornado,
		warning.Damage,
		warning.HailThreat,
		warning.HailTag,
		warning.WindThreat,
		warning.WindTag,
		warning.FlashFlood,
		warning.RainfallTag,
		warning.FloodTagDam,
		warning.SpoutTag,
		warning.SnowSquall,
		warning.SnowSquallTag,
	)
	if err != nil {
		return err
	}

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			log.Error().Err(err).Msg("failed to scan warning ID from insert")
			continue
		}
		warning.ID = id
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("failed to insert warning: %v", err.Error())
	}

	if handler.rabbit == nil {
		log.Warn().Msg("handler missing RabbitMQ channel. Not publishing warning")
		return nil
	}

	warningId := warning.GenerateID()
	_, ok := handler.publishedWarnings[warningId]

	switch warning.Action {
	case "CAN", "UPG":
		err = handler.publishWarning(warning, streaming.EventDelete)
	default:
		err = handler.publishWarning(warning, streaming.EventNew)
	}

	if err != nil {
		return err
	}

	if !ok {
		handler.publishedWarnings[warningId] = struct{}{}
	}

	return nil
}

func (handler *vtecHandler) publishWarning(warning *warning, eventType string) error {
	data, err := json.Marshal(warning)
	if err != nil {
		return err
	}

	return handler.rabbit.PublishWithContext(context.Background(),
		streaming.ExchangeLiveName,
		"warning",
		false,
		false,
		amqp091.Publishing{
			ContentType: "application/json",
			MessageId:   warning.GenerateID(),
			Timestamp:   time.Now(),
			Type:        eventType,
			AppId:       "us.parse.awips",
			Body:        data,
		},
	)
}
