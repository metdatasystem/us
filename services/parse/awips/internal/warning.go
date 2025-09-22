package internal

import (
	"time"

	"github.com/metdatasystem/us/pkg/awips"
	"github.com/metdatasystem/us/pkg/db"
	"github.com/metdatasystem/us/pkg/kafka"
	"github.com/metdatasystem/us/pkg/models"
	"github.com/twpayne/go-geos"
)

func (handler *vtecHandler) warning(segment *awips.ProductSegment, event *models.VTECEvent, vtec awips.VTEC, ugcs []*models.UGCMinimal) error {
	product := handler.product

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

		updatedUGC := append([]string{}, ugcList...)

		for _, ugc := range warning.UGC {
			switch vtec.Action {
			case "CAN", "UPG", "EXP":
				index := -1
				for i, u := range updatedUGC {
					if u == ugc {
						index = i
					}
				}

				if index > -1 {
					ret := make([]string, 0)
					ret = append(ret, updatedUGC[:index]...)
					updatedUGC = append(ret, updatedUGC[index+1:]...)
				}
			default:
				index := -1
				for i, u := range warning.UGC {
					if u == ugc {
						index = i
					}
				}

				if index == -1 {
					updatedUGC = append(updatedUGC, ugc)
				} else {

				}
			}
		}

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
		warning.UGC = updatedUGC
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

		err = db.InsertWarning(handler.db, warning)
		if err != nil {
			return err
		}

	}

	var eventType string
	switch warning.Action {
	case "NEW", "EXA", "EXB":
		eventType = kafka.EventNew
	case "CAN", "UPG":
		eventType = kafka.EventDelete
	default:
		eventType = kafka.EventUpdate
	}

	kafkaEvent := &kafka.EventEnvelope{
		EventType: eventType,
		Product:   "warning",
		ID:        warning.GenerateID(),
		Timestamp: product.Issued,
		Data:      warning,
	}

	return kafka.PublishEvent(handler.kafka, kafkaEvent, "us-warnings")
}
