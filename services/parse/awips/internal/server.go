package internal

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/metdatasystem/us/pkg/streaming"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var monitor = &Monitor{
	Ready: prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "us",
		Subsystem: "parse",
		Name:      "awips_ready",
		Help:      "Indicates if the AWIPS parse server is ready",
	}),
	DBReady: prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "us",
		Subsystem: "parse",
		Name:      "database_ready",
		Help:      "Indicates if the AWIPS parse server is connected to the database",
	}),
	RabbitReady: prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "us",
		Subsystem: "parse",
		Name:      "rabbitmq_ready",
		Help:      "Indicates if the AWIPS parse server is connected to the RabbitMQ broker",
	}),
	ReceivedMessages: prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "us",
		Subsystem: "parse",
		Name:      "received_messages",
		Help:      "How many messages the server has received to process",
	}),
	ProcessedMessages: prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "us",
		Subsystem: "parse",
		Name:      "processed_messages",
		Help:      "How many messages the server has processed",
	}, []string{"result"}),
	MessageProcessTime: prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "us",
		Subsystem: "parse",
		Name:      "message_process_time_seconds",
		Help:      "Time taken to process AWIPS messages",
		Buckets:   prometheus.DefBuckets,
	}),
}

func init() {
	prometheus.MustRegister(monitor.Ready)
	prometheus.MustRegister(monitor.RabbitReady)
	prometheus.MustRegister(monitor.ReceivedMessages)
	prometheus.MustRegister(monitor.ProcessedMessages)
	prometheus.MustRegister(monitor.MessageProcessTime)
}

func Server(logLevel zerolog.Level) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	zerolog.SetGlobalLevel(logLevel)

	db, err := newDatabasePool(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise database")
		return
	}
	monitor.DBReady.Set(1)

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
	monitor.RabbitReady.Set(1)

	go func() {
		log.Info().Msg("consuming messages")

		for message := range messages {
			monitor.ReceivedMessages.Inc()

			go func(message amqp.Delivery) {
				start := prometheus.NewTimer(monitor.MessageProcessTime)
				defer start.ObserveDuration()

				switch message.ContentType {
				case "text/plain":
					HandleText(string(message.Body), message.Timestamp, db, rabbitChannel)
				case "application/json":
					data := &streaming.AWIPSRaw{}
					if err := data.Unmarshal(message.Body); err != nil {
						log.Error().Err(err).Msg("failed to unmarshal awips raw")
						monitor.ProcessedMessages.WithLabelValues("failure").Inc()
						message.Nack(false, false)
						return
					}
					err := Handle(data.Text, message.Timestamp, data.TTAAII, data.CCCC, data.AWIPS, db, rabbitChannel)
					if err != nil {
						monitor.ProcessedMessages.WithLabelValues("failure").Inc()
						message.Nack(false, false)
						return
					}
				}
				monitor.ProcessedMessages.WithLabelValues("success").Inc()
				message.Ack(false)
			}(message)
		}

	}()

	secondTimer := time.NewTimer(15 * time.Second)
	minuteTimer := time.NewTimer(time.Minute)
	go func() {
		for {

			select {
			case <-secondTimer.C:
				monitor.Ready.Set(1)
				secondTimer.Reset(15 * time.Second)
			case <-minuteTimer.C:
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)

				err := db.Ping(ctx)
				if err != nil {
					log.Error().Err(err).Msg("failed to ping database")
					monitor.DBReady.Set(0)
				} else {
					monitor.DBReady.Set(1)
				}

				if rabbitChannel.IsClosed() {
					monitor.RabbitReady.Set(0)
				} else {
					monitor.RabbitReady.Set(1)
				}

				cancel()
				minuteTimer.Reset(time.Minute)
			}
		}
	}()

	go serveMetrics()

	monitor.Ready.Set(1)

	<-ctx.Done()
	log.Warn().Msg("shutting down")
	monitor.Ready.Set(0)
	monitor.DBReady.Set(0)
	monitor.RabbitReady.Set(0)

	db.Close()
	err = rabbitChannel.Close()
	if err != nil {
		log.Error().Err(err).Msg("failed to close rabbit channel")
	}
}

type Monitor struct {
	Ready prometheus.Gauge

	DBReady prometheus.Gauge

	RabbitReady prometheus.Gauge

	ReceivedMessages   prometheus.Counter
	ProcessedMessages  *prometheus.CounterVec
	MessageProcessTime prometheus.Histogram
}

func serveMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
