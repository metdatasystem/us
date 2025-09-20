package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/metdatasystem/us/services/parse/awips/internal/handler"
	"github.com/rs/zerolog/log"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twpayne/go-geos"
	pgxgeos "github.com/twpayne/pgx-geos"
)

type Server struct {
	ctx         context.Context
	db          *pgxpool.Pool
	kafkaClient *kgo.Client
	messages    chan *Message
}

func New(ctx context.Context) (*Server, error) {
	db, err := newDatabasePool()
	if err != nil {
		return nil, fmt.Errorf("failed to initialise database: %v", err)
	}

	kafkaClient, err := newKafkaClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialise kafka client: %v", err)
	}

	server := &Server{
		ctx:         ctx,
		db:          db,
		kafkaClient: kafkaClient,
		messages:    make(chan *Message, 100),
	}

	return server, nil
}

func (server *Server) Run() {
	// Consume from Kafka
	go server.consume()

	// Process messages
	go func() {
		for message := range server.messages {
			go func() {
				handler.Handle(message.Text, message.ReceivedAt, server.db, server.kafkaClient)
			}()
		}

	}()

}

// Send the shutdown signal to the server.
func (server *Server) Stop() {
	log.Info().Msg("shutting down server")
	<-server.ctx.Done()
}

type Message struct {
	Text       string    `json:"text"`
	ReceivedAt time.Time `json:"receivedAt"`
}

func (server *Server) consume() {
	log.Info().Msg("consuming messages")

	for {
		select {
		case <-server.ctx.Done():
			server.kafkaClient.Close()
			log.Info().Msg("consumer closed")
			return
		default:
			log.Info().Msg("polling")
			fetches := server.kafkaClient.PollFetches(context.Background())
			if fetches.IsClientClosed() {
				log.Warn().Msg("kafka client closed")
				server.Stop()
				return
			}
			fetches.EachError(func(t string, p int32, err error) {
				log.Err(err).Str("topic", t).Int32("partition", p).Msg("error fetching")
			})

			var seen int
			fetches.EachRecord(func(record *kgo.Record) {
				message := &Message{}
				if err := json.Unmarshal(record.Value, message); err != nil {
					log.Err(err).Msg("failed to unmarshal message")
					return
				}
				server.messages <- message
				seen++
			})
			if err := server.kafkaClient.CommitUncommittedOffsets(context.Background()); err != nil {
				log.Err(err).Msg("failed to commit offsets")
				continue
			}
			log.Info().Int("seen", seen).Msg("fetched records")
		}
	}
}

func newDatabasePool() (*pgxpool.Pool, error) {
	ctx := context.Background()

	config, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		if err := pgxgeos.Register(ctx, conn, geos.NewContext()); err != nil {
			return err
		}
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	return pool, nil
}

func newKafkaClient() (*kgo.Client, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(os.Getenv("KAFKA_BROKERS")),
		kgo.ConsumerGroup("us-awips-raw"),
		kgo.ConsumeTopics("us-awips-raw"),
		kgo.DisableAutoCommit(),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	return client, nil
}
