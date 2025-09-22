package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/metdatasystem/us/pkg/kafka"
	"github.com/rs/zerolog/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

func Server() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := newDatabasePool()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise database")
		return
	}

	kafkaClient, err := newKafkaClient()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise kafka client")
		return
	}

	messages := make(chan *kafka.EventEnvelope, 100)

	go func() {
		log.Info().Msg("consuming messages")

		for {
			select {
			case <-ctx.Done():
				kafkaClient.Close()
				log.Info().Msg("consumer closed")
				return
			default:
				log.Info().Msg("polling")
				fetches := kafkaClient.PollFetches(context.Background())
				if fetches.IsClientClosed() {
					log.Warn().Msg("kafka client closed")
					<-ctx.Done()
					continue
				}
				fetches.EachError(func(t string, p int32, err error) {
					log.Err(err).Str("topic", t).Int32("partition", p).Msg("error fetching")
				})

				var seen int
				fetches.EachRecord(func(record *kgo.Record) {
					message := &kafka.EventEnvelope{}
					if err := json.Unmarshal(record.Value, message); err != nil {
						log.Err(err).Msg("failed to unmarshal message")
						return
					}
					messages <- message
					seen++
				})
				if err := kafkaClient.CommitUncommittedOffsets(context.Background()); err != nil {
					log.Err(err).Msg("failed to commit offsets")
					continue
				}
				log.Info().Int("seen", seen).Msg("fetched records")
			}
		}
	}()

	go func() {
		for message := range messages {
			text := fmt.Sprintf("%v", message.Data)

			go func() {
				Handle(text, message.Timestamp, db, kafkaClient)
			}()
		}

	}()

	<-ctx.Done()
	log.Warn().Msg("shutting down")
}
