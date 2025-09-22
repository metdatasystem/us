package internal

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/metdatasystem/us/pkg/kafka"
	"github.com/rs/zerolog/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Producer struct {
	client   *kgo.Client
	messages chan *kafka.EventEnvelope
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

	err = client.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to ping Kafka client: %v", err)
	}

	producer := &Producer{
		client:   client,
		messages: make(chan *kafka.EventEnvelope),
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

func (p *Producer) SendMessage(message *kafka.EventEnvelope) error {
	data, err := message.Marshal()
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

func (p *Producer) NewMessage(text string, receivedAt time.Time) *kafka.EventEnvelope {
	return &kafka.EventEnvelope{
		EventType: kafka.EventNew,
		Product:   "awips-raw",
		Data:      text,
		Timestamp: time.Now(),
	}
}
