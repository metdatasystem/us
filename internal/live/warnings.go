package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/metdatasystem/us/shared/streaming"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/geojson"
)

const WarningTopic string = "warnings"

type warningDTO struct {
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
func (warning *warningDTO) GenerateID() string {
	return fmt.Sprintf("%v-%v-%v-%04v-%v", warning.WFO, warning.Phenomena, warning.Significance, warning.EventNumber, warning.Year)
}

// Generates an ID using the warning's generated ID from [warning.GenerateID()], appending the unique integer ID from the database.
//
// Example: KOUN-SV-W-0001-2025-1
func (w *warningDTO) CompositeID() string {
	return fmt.Sprintf("%s-%v", w.GenerateID(), w.ID)
}

type warning struct {
	ID             int                `json:"id"`
	WarningID      string             `json:"warningID"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updatedAt,omitzero"`
	Issued         time.Time          `json:"issued"`
	Starts         *time.Time         `json:"starts,omitzero"`
	Expires        time.Time          `json:"expires"`
	Ends           time.Time          `json:"ends,omitzero"`
	ExpiresInitial time.Time          `json:"expires_initial,omitzero"`
	Current        bool               `json:"current"`
	Product        string             `json:"product"`
	Text           string             `json:"text"`
	WFO            string             `json:"wfo"`
	Action         string             `json:"action"`
	Class          string             `json:"class"`
	Phenomena      string             `json:"phenomena"`
	Significance   string             `json:"significance"`
	EventNumber    int                `json:"eventNumber"`
	Year           int                `json:"year"`
	Title          string             `json:"title"`
	IsEmergency    bool               `json:"isEmergency"`
	IsPDS          bool               `json:"isPDS"`
	Geom           *geom.MultiPolygon `json:"geom,omitempty"`
	Direction      *int               `json:"direction"`
	Locations      *geom.MultiPoint   `json:"locations"`
	Speed          *int               `json:"speed"`
	SpeedText      *string            `json:"speedText"`
	TMLTime        *time.Time         `json:"tmlTime"`
	UGC            map[string]UGC     `json:"ugc"`
	Tornado        string             `json:"tornado,omitempty"`
	Damage         string             `json:"damage,omitempty"`
	HailThreat     string             `json:"hailThreat,omitempty"`
	HailTag        string             `json:"hailTag,omitempty"`
	WindThreat     string             `json:"windThreat,omitempty"`
	WindTag        string             `json:"windTag,omitempty"`
	FlashFlood     string             `json:"flashFlood,omitempty"`
	RainfallTag    string             `json:"rainfallTag,omitempty"`
	FloodTagDam    string             `json:"floodTagDam,omitempty"`
	SpoutTag       string             `json:"spoutTag,omitempty"`
	SnowSquall     string             `json:"snowSquall,omitempty"`
	SnowSquallTag  string             `json:"snowSquall_tag,omitempty"`
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
func (w *warning) CompositeID() string {
	return fmt.Sprintf("%s-%v", w.GenerateID(), w.ID)
}

func (w *warning) MarshalJSON() ([]byte, error) {
	type Alias warning // Use type alias to avoid recursion

	aux := struct {
		Alias
		Geom      string `json:"geom,omitempty"`
		Locations string `json:"locations,omitempty"`
	}{
		Alias: (Alias)(*w),
	}

	if w.Geom != nil {
		b, err := geojson.Marshal(w.Geom)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal geometry: %v", err.Error())
		}
		aux.Geom = string(b)
	}

	if w.Locations != nil {
		b, err := geojson.Marshal(w.Locations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal locations: %v", err.Error())
		}
		aux.Locations = string(b)
	}

	return json.Marshal(aux)
}

type WarningManager struct {
	mu sync.Mutex

	hub         *Hub
	rabbitQueue amqp.Queue

	data        map[string]*warning
	subscribers map[*client]struct{}

	ticker *time.Ticker
}

func AttachWarningManager(hub *Hub) {
	hub.managers["warnings"] = NewWarningManager(hub)
}

func NewWarningManager(hub *Hub) *WarningManager {

	ticker := time.NewTicker(60 * time.Second)

	store := &WarningManager{
		hub:         hub,
		data:        map[string]*warning{},
		subscribers: map[*client]struct{}{},
		ticker:      ticker,
	}

	return store
}

func (manager *WarningManager) Load() error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	rabbit := manager.hub.rabbit

	// Declare and bind the RabbitMQ queues we will be consuming from
	q, err := rabbit.QueueDeclare(
		"live.warning",
		false,
		true,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	manager.rabbitQueue = q

	if err := rabbit.QueueBind(
		q.Name,
		"warning",
		streaming.ExchangeLiveName,
		false,
		nil,
	); err != nil {
		return err
	}

	// Get all the current warnings
	rows, err := manager.hub.db.Query(context.Background(), `
	SELECT * FROM warnings.warnings WHERE action NOT IN ('CAN', 'EXP', 'UPG') AND ends > now() AND current = true
	`)
	if err != nil {
		rows.Close()
		return fmt.Errorf("failed to get active warnings: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		w, err := manager.scanWarning(rows)
		if err != nil {
			return err
		}

		manager.data[w.CompositeID()] = w
	}
	if rows.Err() != nil {
		return err
	}

	log.Debug().Int("size", len(manager.data)).Msg("loaded warning data")

	return nil
}

func (manager *WarningManager) Run() {
	d, err := manager.hub.rabbit.Consume(
		manager.rabbitQueue.Name,
		"live.warning",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to begin consuming warnings")
		return
	}

	go func() {
		for {
			select {
			case t := <-manager.ticker.C:
				manager.ticker.Reset(60 * time.Second)
				manager.checkExpired(t)
			case message := <-d:

				w := &warningDTO{}

				if err := json.Unmarshal(message.Body, &w); err != nil {
					log.Error().Err(err).Msg("failed to unmarshal warning message")
					continue
				}

				err = manager.handleUpdate(w, message.Type)
				if err != nil {
					log.Error().Err(err).Msg("failed to handle warning update")
					continue
				}
				message.Ack(true)
			}
		}
	}()
}

func (manager *WarningManager) Subscribe(c *client) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	manager.subscribers[c] = struct{}{}

	warnings := []*warning{}
	for _, w := range manager.data {
		warnings = append(warnings, w)
	}

	// Marshal the warnings slice to JSON
	warningsBytes, err := json.Marshal(warnings)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal warnings for subscription")
		return
	}

	// Create the envelope
	envelope := Envelope{
		Type:      EnvelopeInitial,
		Product:   WarningTopic,
		ID:        "",
		Timestamp: time.Now(),
		Data:      warningsBytes,
	}

	// Marshal the envelope to JSON if you need to send it
	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal envelope for subscription")
		return
	}

	c.send <- envelopeBytes

	log.Debug().Int("size", len(warnings)).Msg("sent initial warning data to client")
}

func (manager *WarningManager) Unsubscribe(c *client) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	delete(manager.subscribers, c)
}

func (manager *WarningManager) handleUpdate(warningDTO *warningDTO, eventType string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	w := &warning{
		ID:             warningDTO.ID,
		WarningID:      warningDTO.CompositeID(),
		CreatedAt:      warningDTO.CreatedAt,
		UpdatedAt:      warningDTO.UpdatedAt,
		Issued:         warningDTO.Issued,
		Starts:         warningDTO.Starts,
		Expires:        warningDTO.Expires,
		Ends:           warningDTO.Ends,
		ExpiresInitial: warningDTO.ExpiresInitial,
		Current:        warningDTO.Current,
		Product:        warningDTO.Product,
		Text:           warningDTO.Text,
		WFO:            warningDTO.WFO,
		Action:         warningDTO.Action,
		Class:          warningDTO.Class,
		Phenomena:      warningDTO.Phenomena,
		Significance:   warningDTO.Significance,
		EventNumber:    warningDTO.EventNumber,
		Year:           warningDTO.Year,
		Title:          warningDTO.Title,
		IsEmergency:    warningDTO.IsEmergency,
		IsPDS:          warningDTO.IsPDS,
		Direction:      warningDTO.Direction,
		Speed:          warningDTO.Speed,
		SpeedText:      warningDTO.SpeedText,
		TMLTime:        warningDTO.TMLTime,
		Tornado:        warningDTO.Tornado,
		Damage:         warningDTO.Damage,
		HailThreat:     warningDTO.HailThreat,
		HailTag:        warningDTO.HailTag,
		WindThreat:     warningDTO.WindThreat,
		WindTag:        warningDTO.WindTag,
		FlashFlood:     warningDTO.FlashFlood,
		RainfallTag:    warningDTO.RainfallTag,
		FloodTagDam:    warningDTO.FloodTagDam,
		SpoutTag:       warningDTO.SpoutTag,
		SnowSquall:     warningDTO.SnowSquall,
		SnowSquallTag:  warningDTO.SnowSquallTag,
	}

	if len(warningDTO.Geom) > 0 {
		g, err := ewkb.Unmarshal(warningDTO.Geom)
		if err != nil {
			return fmt.Errorf("failed to unmarshal warning geometry: %v", err.Error())
		}

		switch g := g.(type) {
		case *geom.MultiPolygon:
			w.Geom = g
		case *geom.Polygon:
			v := geom.NewMultiPolygon(geom.XY)
			v, err = v.SetCoords([][][]geom.Coord{g.Coords()})
			if err != nil {
				log.Error().Err(err).Msg("failed to convert polygon to multipolygon")
			}
			w.Geom = v
		default:
			log.Warn().Msg("warning geometry was not a polygon or multipolygon")
		}
	}

	if len(warningDTO.Locations) > 0 {
		l, err := ewkb.Unmarshal(warningDTO.Locations)
		if err != nil {
			return fmt.Errorf("failed to unmarshal warning locations: %v", err.Error())
		}

		switch l := l.(type) {
		case *geom.MultiPoint:
			w.Locations = l
		case *geom.Point:
			v := geom.NewMultiPoint(geom.XY)
			v, err = v.SetCoords(v.Coords())
			if err != nil {
				log.Error().Err(err).Msg("failed to convert point to multipoint")
			}
			w.Locations = v
		default:
			log.Warn().Msg("location points was not a point or multipoint")
		}
	}

	ugcs := map[string]UGC{}

	for _, ugc := range warningDTO.UGC {
		ugc := manager.hub.ugcStore.findUGC(ugc)
		if ugc != nil {
			ugcs[ugc.Code] = *ugc
		}
	}

	warningBytes, err := json.Marshal(w)
	if err != nil {
		return err
	}

	envelope := Envelope{
		Type:      eventType,
		Product:   WarningTopic,
		ID:        w.CompositeID(),
		Timestamp: time.Now(),
		Data:      warningBytes,
	}

	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	for client := range manager.subscribers {
		client.send <- envelopeBytes
	}

	// See if we have the warning already
	if _, ok := manager.data[w.CompositeID()]; ok {
		if eventType == streaming.EventDelete {
			delete(manager.data, w.CompositeID())
		} else {
			if _, ok := manager.data[w.CompositeID()]; ok {
				manager.data[w.CompositeID()] = w
			}
		}
	} else if eventType != streaming.EventDelete {
		manager.data[w.CompositeID()] = w
	}

	return nil
}

func (manager *WarningManager) checkExpired(t time.Time) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	toDelete := []*warning{}
	for _, w := range manager.data {
		if w.Ends.Before(t) {
			toDelete = append(toDelete, w)
		}
	}

	if len(toDelete) > 0 {
		for _, w := range toDelete {
			delete(manager.data, w.WarningID)

			warningBytes, err := json.Marshal(w)
			if err != nil {
				log.Error().Err(err).Msg("failed to marshal warning for expired warning")
			}

			envelope := Envelope{
				Type:      EnvelopeDelete,
				Product:   WarningTopic,
				ID:        w.CompositeID(),
				Timestamp: time.Now(),
				Data:      warningBytes,
			}

			envelopeBytes, err := json.Marshal(envelope)
			if err != nil {
				log.Error().Err(err).Msg("failed to marshal envelope for expired warning")
			}

			for client := range manager.subscribers {
				client.send <- envelopeBytes
			}
		}

		log.Debug().Int("deleted", len(toDelete)).Msg("deleted expired warnings")
	}
}

func (manager *WarningManager) scanWarning(row pgx.Row) (*warning, error) {

	g := ewkb.MultiPolygon{}
	locs := ewkb.MultiPoint{}
	u := []string{}

	w := warning{}

	row.Scan(
		&w.ID,
		&w.Phenomena,
		&w.Significance,
		&w.WFO,
		&w.EventNumber,
		&w.Year,
		&w.Action,
		&w.Current,
		&w.CreatedAt,
		&w.UpdatedAt,
		&w.Issued,
		&w.Starts,
		&w.Expires,
		&w.ExpiresInitial,
		&w.Ends,
		&w.Class,
		&w.Title,
		&w.IsEmergency,
		&w.IsPDS,
		&w.Text,
		&w.Product,
		&g,
		&w.Direction,
		&locs,
		&w.Speed,
		&w.SpeedText,
		&w.TMLTime,
		&u,
		&w.Tornado,
		&w.Damage,
		&w.HailThreat,
		&w.HailTag,
		&w.WindThreat,
		&w.WindTag,
		&w.FlashFlood,
		&w.RainfallTag,
		&w.FloodTagDam,
		&w.SpoutTag,
		&w.SnowSquall,
		&w.SnowSquallTag,
	)

	ugcs := map[string]UGC{}

	for _, ugc := range u {
		ugc := manager.hub.ugcStore.findUGC(ugc)
		if ugc != nil {
			ugcs[ugc.Code] = *ugc
		}
	}

	w.WarningID = w.CompositeID()
	w.Geom = g.MultiPolygon
	w.Locations = locs.MultiPoint
	w.UGC = ugcs

	return &w, nil
}
