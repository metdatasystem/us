package kafka

import (
	"context"
	"encoding/json"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

const EventNew = "NEW"
const EventUpdate = "UPDATE"
const EventDelete = "DELETE"

func PublishEvent(client *kgo.Client, event *EventEnvelope, topic string) error {
	b, err := event.Marshal()
	if err != nil {
		return err
	}

	record := &kgo.Record{
		Topic: topic,
		Value: b,
	}

	result := client.ProduceSync(context.Background(), record)

	return result.FirstErr()
}

type EventEnvelope struct {
	EventType string    `json:"event_type"`
	Product   string    `json:"product"`
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

func (envelope EventEnvelope) Marshal() ([]byte, error) {
	return json.Marshal(envelope)
}
