package internal

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/metdatasystem/us/pkg/streaming"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Server(logLevel zerolog.Level) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	zerolog.SetGlobalLevel(logLevel)

	db, err := newDatabasePool()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise database")
		return
	}

	rabbitChannel, err := newRabbitChannel()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise rabbit channel")
		return
	}

	err = initRabbit(rabbitChannel)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise rabbit declarations")
		return
	}

	messages, err := rabbitChannel.Consume(
		streaming.QueueAWIPS,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	go func() {
		for message := range messages {

			go func(message amqp.Delivery) {
				switch message.ContentType {
				case "text/plain":
					HandleText(string(message.Body), message.Timestamp, db, rabbitChannel)
				case "application/json":
					data := &streaming.AWIPSRaw{}
					if err := data.Unmarshal(message.Body); err != nil {
						log.Error().Err(err).Msg("failed to unmarshal awips raw")
						return
					}
					Handle(data.Text, message.Timestamp, data.TTAAII, data.CCCC, data.AWIPS, db, rabbitChannel)
				}

				message.Ack(false)
			}(message)
		}

	}()

	<-ctx.Done()
	log.Warn().Msg("shutting down")
	err = rabbitChannel.Close()
	if err != nil {
		log.Error().Err(err).Msg("failed to close rabbit channel")
	}
}
