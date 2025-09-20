package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Message struct {
	Text       string    `json:"text"`
	ReceivedAt time.Time `json:"receivedAt"`
}

type Producer struct {
	client   *kgo.Client
	messages chan *Message
	done     bool
}

func NewProducer() (*Producer, error) {

	// Create Kafka client
	client, err := kgo.NewClient(
		kgo.SeedBrokers(os.Getenv("KAFKA_BROKER")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %v", err)
	}

	producer := &Producer{
		client:   client,
		messages: make(chan *Message),
		done:     false,
	}

	return producer, nil
}

func (p *Producer) Run() {
	for message := range p.messages {

		if p.done {
			p.client.Close()
			return
		}

		err := p.SendMessage(message)
		if err != nil {
			log.Error().Err(err).Msg("failed to send message")
		}

	}
}

func (p *Producer) Stop() {
	p.done = true
}

func (p *Producer) SendMessage(message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err.Error())
	}

	record := &kgo.Record{
		Topic: "us-awips-raw",
		Value: data,
	}
	result := p.client.ProduceSync(context.Background(), record)
	if result.FirstErr() != nil {
		return result.FirstErr()
	}

	return nil
}

func (p *Producer) NewMessage(text string, receivedAt time.Time) *Message {
	return &Message{
		Text:       text,
		ReceivedAt: time.Now(),
	}
}
