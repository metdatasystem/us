package internal

import (
	"context"
	"encoding/json"
	"time"

	"github.com/metdatasystem/us/pkg/awips"
	"github.com/metdatasystem/us/pkg/db"
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
		return nil
	}

	ugcList := []string{}
	for _, u := range ugcs {
		ugcList = append(ugcList, u.UGC)
	}

	var geom *geos.Geom
	if segment.LatLon != nil {
		coords := segment.LatLon.ToFloatClosing()
		geom = geos.NewPolygon([][][]float64{coords})
	} else {

		g, err := db.GetUGCUnionGeomSimplified(handler.db, ugcList)
		if err != nil {
			return err
		}

		geom = g
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

	warning, err := db.FindWarning(handler.db, event.WFO, event.Phenomena, event.Significance, event.EventNumber, event.Year)
	if err != nil {
		return err
	}

	// Warning exists and can be updated
	if warning != nil {

		warning.Expires = segment.UGC.Expires
		warning.Ends = event.Ends
		warning.Text = segment.Text
		warning.Action = vtec.Action
		warning.Title = vtec.Title(segment.IsEmergency())
		warning.IsEmergency = segment.IsEmergency()
		warning.IsPDS = segment.IsPDS()
		warning.Geom = geom
		warning.Direction = direction
		warning.Location = locations
		warning.Speed = speed
		warning.SpeedText = speedText
		warning.TMLTime = tmlTime
		warning.Tornado = segment.Tags["tornado"]
		warning.Damage = segment.Tags["damage"]
		warning.HailThreat = segment.Tags["hailThreat"]
		warning.HailTag = segment.Tags["hail"]
		warning.WindThreat = segment.Tags["windThreat"]
		warning.WindTag = segment.Tags["wind"]
		warning.FlashFlood = segment.Tags["flashFlood"]
		warning.RainfallTag = segment.Tags["expectedRainfall"]
		warning.FloodTagDam = segment.Tags["damFailure"]
		warning.SpoutTag = segment.Tags["spout"]
		warning.SnowSquall = segment.Tags["snowSquall"]
		warning.SnowSquallTag = segment.Tags["snowSquallImpact"]

		// Publish before we override all the UGC data
		err = handler.publishWarning(warning)
		if err != nil {
			log.Error().Err(err).Msg("failed to publish warning to kafka")
		}

		// Convert warning.UGC into a map for fast lookups
		existing := make(map[string]bool)
		for _, v := range warning.UGC {
			existing[v] = true
		}

		// Handle based on Action
		switch warning.Action {
		case "CAN", "UPG", "EXP":
			// Remove elements that exist in newWarning.UGC
			filtered := []string{}
			toRemove := make(map[string]bool)
			for _, v := range ugcList {
				toRemove[v] = true
			}
			for _, v := range warning.UGC {
				if !toRemove[v] {
					filtered = append(filtered, v)
				}
			}
			warning.UGC = filtered

		default:
			// Add new elements that are not already in warning.UGC
			for _, v := range ugcList {
				if !existing[v] {
					warning.UGC = append(warning.UGC, v)
					existing[v] = true
				}
			}
		}

		if err := db.UpdateWarning(handler.db, warning); err != nil {
			return err
		}
	} else {

		warning = &models.Warning{
			Issued:        product.Issued,
			Starts:        event.Starts,
			Expires:       segment.UGC.Expires,
			Ends:          event.Ends,
			EndInitial:    event.EndInitial,
			Text:          segment.Text,
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
			Geom:          geom,
			Direction:     direction,
			Location:      locations,
			Speed:         speed,
			SpeedText:     speedText,
			TMLTime:       tmlTime,
			UGC:           ugcList,
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

		err = handler.publishWarning(warning)
		if err != nil {
			log.Error().Err(err).Msg("failed to publish warning to kafka")
		}

		err = db.InsertWarning(handler.db, warning)
		if err != nil {
			return err
		}

	}

	return nil
}

func (handler *vtecHandler) publishWarning(warning *models.Warning) error {
	var eventType string
	switch warning.Action {
	case "NEW", "EXA", "EXB":
		eventType = streaming.EventNew
	case "CAN", "UPG":
		eventType = streaming.EventDelete
	default:
		eventType = streaming.EventUpdate
	}

	data, err := json.Marshal(warning)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	return handler.rabbit.PublishWithContext(ctx,
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
