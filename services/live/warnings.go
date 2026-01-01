package main

import (
	"encoding/json"
	"fmt"
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
	ID             int            `json:"id"`
	WarningID      string         `json:"warningID"`
	UpdatedAt      time.Time      `json:"updatedAt,omitzero"`
	Issued         time.Time      `json:"issued"`
	Starts         *time.Time     `json:"starts,omitzero"`
	Expires        time.Time      `json:"expires"`
	Ends           time.Time      `json:"ends,omitzero"`
	ExpiresInitial time.Time      `json:"expires_initial,omitzero"`
	Text           string         `json:"text"`
	WFO            string         `json:"wfo"`
	Action         string         `json:"action"`
	Class          string         `json:"class"`
	Phenomena      string         `json:"phenomena"`
	Significance   string         `json:"significance"`
	EventNumber    int            `json:"eventNumber"`
	Year           int            `json:"year"`
	Title          string         `json:"title"`
	IsEmergency    bool           `json:"isEmergency"`
	IsPDS          bool           `json:"isPDS"`
	Geom           *geos.Geom     `json:"geom,omitempty"`
	Direction      *int           `json:"direction"`
	Location       *geos.Geom     `json:"location"`
	Speed          *int           `json:"speed"`
	SpeedText      *string        `json:"speedText"`
	TMLTime        *time.Time     `json:"tmlTime"`
	UGC            map[string]UGC `json:"ugc"`
	Tornado        string         `json:"tornado,omitempty"`
	Damage         string         `json:"damage,omitempty"`
	HailThreat     string         `json:"hailThreat,omitempty"`
	HailTag        string         `json:"hailTag,omitempty"`
	WindThreat     string         `json:"windThreat,omitempty"`
	WindTag        string         `json:"windTag,omitempty"`
	FlashFlood     string         `json:"flashFlood,omitempty"`
	RainfallTag    string         `json:"rainfallTag,omitempty"`
	FloodTagDam    string         `json:"floodTagDam,omitempty"`
	SpoutTag       string         `json:"spoutTag,omitempty"`
	SnowSquall     string         `json:"snowSquall,omitempty"`
	SnowSquallTag  string         `json:"snowSquall_tag,omitempty"`
}

func (w *Warning) CompositeID() string {
	return fmt.Sprintf("%s-%v", w.WarningID, w.ID)
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

	data        map[string]map[int]*Warning
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
		data:        map[string]map[int]*Warning{},
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

		_, ok := manager.data[id]
		if !ok {
			manager.data[id] = map[int]*Warning{warning.ID: manager.modelToWarning(*warning)}
		} else {
			manager.data[id][warning.ID] = manager.modelToWarning(*warning)
		}
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
				warning := &models.Warning{}
				if err := warning.UnmarshalJSON(message.Body); err != nil {
					log.Error().Err(err).Msg("failed to unmarshal warning message")
					continue
				}
				err := manager.handleUpdate(*warning, message.Type)
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
	for _, list := range manager.data {
		for _, w := range list {
			warnings = append(warnings, w)
		}
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

func (manager *WarningManager) handleUpdate(w models.Warning, eventType string) error {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	warning := manager.modelToWarning(w)

	warningBytes, err := json.Marshal(warning)
	if err != nil {
		return err
	}

	envelope := Envelope{
		Type:      eventType,
		Product:   WarningTopic,
		ID:        warning.CompositeID(),
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
	if _, ok := manager.data[warning.WarningID]; ok {
		if eventType == streaming.EventDelete {
			delete(manager.data, warning.WarningID)
		} else {
			if _, ok := manager.data[warning.WarningID][warning.ID]; ok {
				manager.data[warning.WarningID][warning.ID] = warning
			}
		}
	} else if eventType != streaming.EventDelete {
		manager.data[warning.WarningID] = map[int]*Warning{warning.ID: warning}
	}

	return nil
}

func (manager *WarningManager) checkExpired(t time.Time) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	toDelete := []*Warning{}
	for _, bucket := range manager.data {
		for _, warning := range bucket {
			if warning.Ends.Before(t) {
				toDelete = append(toDelete, warning)
			}
		}
	}

	if len(toDelete) > 0 {
		for _, warning := range toDelete {
			delete(manager.data[warning.WarningID], warning.ID)

			warningBytes, err := json.Marshal(warning)
			if err != nil {
				log.Error().Err(err).Msg("failed to marshal warning for expired warning")
			}

			envelope := Envelope{
				Type:      EnvelopeDelete,
				Product:   WarningTopic,
				ID:        warning.CompositeID(),
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

func (manager *WarningManager) modelToWarning(w models.Warning) *Warning {

	ugcs := map[string]UGC{}

	for _, ugc := range w.UGC {
		ugc := manager.hub.ugcStore.findUGC(ugc)
		if ugc != nil {
			ugcs[ugc.Code] = *ugc
		}
	}

	return &Warning{
		ID:             w.ID,
		WarningID:      w.GenerateCompositeID(),
		UpdatedAt:      w.UpdatedAt,
		Issued:         w.Issued,
		Starts:         w.Starts,
		Expires:        w.Expires,
		Ends:           w.Ends,
		ExpiresInitial: w.ExpiresInitial,
		Text:           w.Text,
		WFO:            w.WFO,
		Action:         w.Action,
		Class:          w.Class,
		Phenomena:      w.Phenomena,
		Significance:   w.Significance,
		EventNumber:    w.EventNumber,
		Year:           w.Year,
		Title:          w.Title,
		IsEmergency:    w.IsEmergency,
		IsPDS:          w.IsPDS,
		Geom:           w.Geom,
		Direction:      w.Direction,
		Location:       w.Location,
		Speed:          w.Speed,
		SpeedText:      w.SpeedText,
		TMLTime:        w.TMLTime,
		UGC:            ugcs,
		Tornado:        w.Tornado,
		Damage:         w.Damage,
		HailThreat:     w.HailThreat,
		HailTag:        w.HailTag,
		WindThreat:     w.WindThreat,
		WindTag:        w.WindTag,
		FlashFlood:     w.FlashFlood,
		RainfallTag:    w.RainfallTag,
		FloodTagDam:    w.FloodTagDam,
		SpoutTag:       w.SpoutTag,
		SnowSquall:     w.SnowSquall,
		SnowSquallTag:  w.SnowSquallTag,
	}
}
