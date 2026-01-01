package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/metdatasystem/us/pkg/awips"
	"github.com/metdatasystem/us/pkg/models"
	"github.com/metdatasystem/us/pkg/streaming"
	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
	"github.com/twpayne/go-geos"
)

func (handler *vtecHandler) warning(segment *awips.ProductSegment, event *models.VTECEvent, vtec awips.VTEC, ugcs []*models.UGCMinimal) error {
	product := handler.product

	yesterday := time.Now().Add(time.Hour * -24)

	if event.Ends.Before(yesterday) {
		// return nil
	}

	ugcList := []string{}
	for _, u := range ugcs {
		ugcList = append(ugcList, u.UGC)
	}

	fmt.Println(vtec.Action)
	fmt.Println(ugcList)

	var geom *geos.Geom
	if segment.LatLon != nil {
		coords := segment.LatLon.ToFloatClosing()
		geom = geos.NewPolygon([][][]float64{coords})
	} else if vtec.Action != "CAN" && vtec.Action != "UPG" && vtec.Action != "EXP" {
		rows, err := handler.tx.Query(handler.ctx, `
			SELECT ST_SimplifyPreserveTopology(ST_Union(geom), 0.0025) FROM postgis.ugcs WHERE valid_to IS NULL AND ugc = ANY($1)
			`, ugcList)
		if err != nil {
			return err
		}

		if rows.Next() {
			g := &geos.Geom{}
			if err := rows.Scan(&g); err != nil {
				return err
			}
			geom = g
		}

		rows.Close()

	}

	fmt.Println(geom)

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

	warning := &models.Warning{
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
		Location:       locations,
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
			data, err := tempW.MarshalJSON()
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
		   $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37
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
		warning.Location,
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

	err = handler.handleWarningPublishing(warning)
	if err != nil {
		log.Error().Err(err).Msg("failed to publish warning")
	}

	return nil
}

func (handler *vtecHandler) handleWarningPublishing(warning *models.Warning) error {
	if handler.rabbit == nil {
		log.Warn().Msg("handler missing RabbitMQ channel. Not publishing warning")
		return nil
	}

	warningId := warning.GenerateID()
	_, ok := handler.publishedWarnings[warningId]

	var err error
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

func (handler *vtecHandler) publishWarning(warning *models.Warning, eventType string) error {
	data, err := warning.MarshalJSON()
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
