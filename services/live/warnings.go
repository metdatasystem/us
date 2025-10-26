package main

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/metdatasystem/us/pkg/db"
	"github.com/metdatasystem/us/pkg/models"
	"github.com/metdatasystem/us/pkg/streaming"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
	"github.com/twpayne/go-geos"
)

const WarningTopic string = "warnings"

type Warning struct {
	ID            string         `json:"id"`
	UpdatedAt     time.Time      `json:"updatedAt,omitzero"`
	Issued        time.Time      `json:"issued"`
	Starts        *time.Time     `json:"starts,omitzero"`
	Expires       time.Time      `json:"expires"`
	Ends          time.Time      `json:"ends,omitzero"`
	EndInitial    time.Time      `json:"endInitial,omitzero"`
	Text          string         `json:"text"`
	WFO           string         `json:"wfo"`
	Action        string         `json:"action"`
	Class         string         `json:"class"`
	Phenomena     string         `json:"phenomena"`
	Significance  string         `json:"significance"`
	EventNumber   int            `json:"eventNumber"`
	Year          int            `json:"year"`
	Title         string         `json:"title"`
	IsEmergency   bool           `json:"isEmergency"`
	IsPDS         bool           `json:"isPDS"`
	Geom          *geos.Geom     `json:"geom,omitempty"`
	Direction     *int           `json:"direction"`
	Location      *geos.Geom     `json:"location"`
	Speed         *int           `json:"speed"`
	SpeedText     *string        `json:"speedText"`
	TMLTime       *time.Time     `json:"tmlTime"`
	UGC           map[string]UGC `json:"ugc"`
	Tornado       string         `json:"tornado,omitempty"`
	Damage        string         `json:"damage,omitempty"`
	HailThreat    string         `json:"hailThreat,omitempty"`
	HailTag       string         `json:"hailTag,omitempty"`
	WindThreat    string         `json:"windThreat,omitempty"`
	WindTag       string         `json:"windTag,omitempty"`
	FlashFlood    string         `json:"flashFlood,omitempty"`
	RainfallTag   string         `json:"rainfallTag,omitempty"`
	FloodTagDam   string         `json:"floodTagDam,omitempty"`
	SpoutTag      string         `json:"spoutTag,omitempty"`
	SnowSquall    string         `json:"snowSquall,omitempty"`
	SnowSquallTag string         `json:"snowSquall_tag,omitempty"`
}

func (w *Warning) MarshalJSON() ([]byte, error) {
	type Alias Warning // Use type alias to avoid recursion

	aux := struct {
		*Alias
		Geom     string `json:"geom,omitempty"`
		Location string `json:"location,omitempty"`
	}{
		Alias: (*Alias)(w),
	}

	if w.Geom != nil {
		aux.Geom = w.Geom.ToGeoJSON(1)
	}

	if w.Location != nil {
		aux.Location = w.Location.ToGeoJSON(1)
	}

	return json.Marshal(aux)
}

type WarningManager struct {
	mu sync.Mutex

	hub         *Hub
	rabbitQueue amqp.Queue

	data        map[string]*Warning
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
		data:        map[string]*Warning{},
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
	warnings, err := db.GetAllActiveWarnings(manager.hub.db)
	if err != nil {
		return err
	}

	for _, warning := range warnings {
		id := warning.GenerateID()

		manager.data[id] = manager.modelToWarning(id, *warning)
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
				manager.checkExpired(t)
			case message := <-d:
				warning := &models.Warning{}
				if err := warning.UnmarshalJSON(message.Body); err != nil {
					log.Error().Err(err).Msg("failed to unmarshal warning message")
					continue
				}
				id := message.MessageId
				err := manager.handleUpdate(id, *warning)
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

	warnings := []*Warning{}
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

func (manager *WarningManager) handleUpdate(id string, w models.Warning) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	newWarning := manager.modelToWarning(id, w)

	var eventType string
	switch newWarning.Action {
	case "NEW", "EXA", "EXB":
		eventType = EnvelopeNew
	case "CAN", "UPG", "EXP":
		eventType = EnvelopeDelete
	default:
		eventType = EnvelopeUpdate
	}

	warningBytes, err := json.Marshal(newWarning)
	if err != nil {
		return err
	}

	envelope := Envelope{
		Type:      eventType,
		Product:   WarningTopic,
		ID:        newWarning.ID,
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
	warning, ok := manager.data[id]
	if !ok {
		if newWarning.Action == "CAN" || newWarning.Action == "UPG" || newWarning.Action == "EXP" {
			return nil
		}
		manager.data[id] = newWarning
		return nil
	}

	if newWarning.Action == "CAN" || newWarning.Action == "UPG" || newWarning.Action == "EXP" {
		codes := []string{}
		for _, ugc := range newWarning.UGC {
			_, ok := warning.UGC[ugc.Code]
			if ok {
				codes = append(codes, ugc.Code)
			}
		}
		for _, code := range codes {
			delete(warning.UGC, code)
		}

		if len(warning.UGC) == 0 {
			delete(manager.data, id)
			return nil
		}
	} else {
		for _, ugc := range newWarning.UGC {
			_, ok := warning.UGC[ugc.Code]
			if !ok {
				warning.UGC[ugc.Code] = ugc
			}
		}
	}

	warning.Expires = newWarning.Expires
	warning.Ends = newWarning.Ends
	warning.Text = newWarning.Text
	warning.Action = newWarning.Action
	warning.Title = newWarning.Title
	warning.IsEmergency = newWarning.IsEmergency
	warning.IsPDS = newWarning.IsPDS
	warning.Geom = newWarning.Geom
	warning.Direction = newWarning.Direction
	warning.Location = newWarning.Location
	warning.Speed = newWarning.Speed
	warning.SpeedText = newWarning.SpeedText
	warning.TMLTime = newWarning.TMLTime
	warning.Tornado = newWarning.Tornado
	warning.Damage = newWarning.Damage
	warning.HailThreat = newWarning.HailThreat
	warning.HailTag = newWarning.HailTag
	warning.WindThreat = newWarning.WindThreat
	warning.WindTag = newWarning.WindTag
	warning.FlashFlood = newWarning.FlashFlood
	warning.RainfallTag = newWarning.RainfallTag
	warning.FloodTagDam = newWarning.FloodTagDam
	warning.SpoutTag = newWarning.SpoutTag
	warning.SnowSquall = newWarning.SnowSquall
	warning.SnowSquallTag = newWarning.SnowSquallTag

	return nil
}

func (manager *WarningManager) checkExpired(t time.Time) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	toDelete := []string{}
	for id, warning := range manager.data {
		if warning.Ends.Before(t) {
			toDelete = append(toDelete, id)
		}
	}

	for _, id := range toDelete {
		delete(manager.data, id)
	}

	if len(toDelete) > 0 {
		log.Debug().Int("deleted", len(toDelete)).Msg("deleted expired warnings")
	}
}

func (manager *WarningManager) modelToWarning(id string, w models.Warning) *Warning {

	ugcs := map[string]UGC{}

	for _, ugc := range w.UGC {
		ugc := manager.hub.ugcStore.findUGC(ugc)
		if ugc != nil {
			ugcs[ugc.Code] = *ugc
		}
	}

	return &Warning{
		ID:            id,
		UpdatedAt:     w.UpdatedAt,
		Issued:        w.Issued,
		Starts:        w.Starts,
		Expires:       w.Expires,
		Ends:          w.Ends,
		EndInitial:    w.EndInitial,
		Text:          w.Text,
		WFO:           w.WFO,
		Action:        w.Action,
		Class:         w.Class,
		Phenomena:     w.Phenomena,
		Significance:  w.Significance,
		EventNumber:   w.EventNumber,
		Year:          w.Year,
		Title:         w.Title,
		IsEmergency:   w.IsEmergency,
		IsPDS:         w.IsPDS,
		Geom:          w.Geom,
		Direction:     w.Direction,
		Location:      w.Location,
		Speed:         w.Speed,
		SpeedText:     w.SpeedText,
		TMLTime:       w.TMLTime,
		UGC:           ugcs,
		Tornado:       w.Tornado,
		Damage:        w.Damage,
		HailThreat:    w.HailThreat,
		HailTag:       w.HailTag,
		WindThreat:    w.WindThreat,
		WindTag:       w.WindTag,
		FlashFlood:    w.FlashFlood,
		RainfallTag:   w.RainfallTag,
		FloodTagDam:   w.FloodTagDam,
		SpoutTag:      w.SpoutTag,
		SnowSquall:    w.SnowSquall,
		SnowSquallTag: w.SnowSquallTag,
	}
}
