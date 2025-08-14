package nwws

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/xmppo/go-xmpp"
)

type Message struct {
	Text       string    `json:"text"`
	ReceivedAt time.Time `json:"receivedAt"`
}

type XmppConfig struct {
	Server   string
	Room     string
	User     string
	Pass     string
	Resource string
}

func Go() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Configure the XMPP client
	xmppConfig := XmppConfig{
		Server:   os.Getenv("NWWSOI_SERVER") + ":5222",
		Room:     os.Getenv("NWWSOI_ROOM"),
		User:     os.Getenv("NWWSOI_USER"),
		Pass:     os.Getenv("NWWSOI_PASS"),
		Resource: os.Getenv("NWWSOI_RESOURCE"),
	}

	err := xmppConfig.check()
	if err != nil {
		slog.Error("NWWS configuration is invalid", "error", err.Error())
		return
	}

	xmpp.DefaultConfig = &tls.Config{
		ServerName:         xmppConfig.serverName(),
		InsecureSkipVerify: false,
	}

	options := xmpp.Options{
		Host:        xmppConfig.Server,
		User:        xmppConfig.User + "@" + xmppConfig.serverName(),
		Password:    xmppConfig.Pass,
		Resource:    xmppConfig.Resource,
		NoTLS:       true,
		StartTLS:    true,
		Debug:       false, // Set to true if you want to see debug information
		Session:     true,
		DialTimeout: 60 * time.Second,
	}

	// Create the XMPP client
	client, err := options.NewClient()
	if err != nil {
		slog.Error("failed to create XMPP client", "error", err.Error())
		return
	}

	slog.Info(fmt.Sprintf("\033[32m *** Connected to %s *** \033[m", xmppConfig.Server))

	// Send presence to the room
	_, err = client.SendOrg(fmt.Sprintf(`<presence xml:lang='en' from='%s@%s' to='%s@%s/%s'><x></x></presence>`, xmppConfig.User, xmppConfig.Server, xmppConfig.Resource, xmppConfig.Room, xmppConfig.User))
	if err != nil {
		slog.Error("failed to send presence", "error", err.Error())
		return
	}

	// Create Kafka client
	kafkaClient, err := kgo.NewClient(
		kgo.SeedBrokers(os.Getenv("KAFKA_BROKER")),
	)
	if err != nil {
		slog.Error("failed to create Kafka client", "error", err.Error())
		return
	}

	// Track time of last received message
	var lastReceived int64
	// Messages per minute
	var messageRate atomic.Uint64
	// Channel to share messages
	messages := make(chan Message)

	// XMPP listening
	go func() {
		defer client.Close()

		slog.Info(fmt.Sprintf("\033[32m *** Listening to %s *** \033[m", xmppConfig.Server))
		for {
			select {
			// Stop
			case <-ctx.Done():
				slog.Info("shutting down XMPP client")
				close(messages)
				return
			// Receive messages
			default:
				chat, err := client.Recv()
				if err != nil {
					slog.Error("failed to receive message", "error", err.Error())
					continue
				}

				switch v := chat.(type) {
				case xmpp.Chat:
					for _, elem := range v.OtherElem {
						// NWWS-OI uses 'x' as the XML element containing the raw text
						if elem.XMLName.Local == "x" {
							// Remove extra newlines
							text := strings.ReplaceAll(elem.String(), "\n\n", "\n")
							// Update last received time
							now := time.Now()
							atomic.StoreInt64(&lastReceived, now.Unix())
							// Share the message
							slog.Debug("received message", "receivedAt", now.String())
							messages <- Message{
								Text:       text,
								ReceivedAt: time.Now(),
							}
							// Increment message rate
							messageRate.Add(1)
						}
					}
				}
			}
		}
	}()

	// Health monitor
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
			last := atomic.LoadInt64(&lastReceived)
			if last == 0 || time.Now().Unix()-last > 60 {
				slog.Warn("no messages received in the last minute", "messagesLastMinute", messageRate.Load())
			} else {
				slog.Debug("healthy", "messagesLastMinute", messageRate.Load())
			}
			// Reset message rate
			messageRate.Swap(0)
		}
	}()

	// Kafka producer
	go func() {
		for message := range messages {

			data, err := json.Marshal(message)
			if err != nil {
				slog.Error("failed to marshal message", "error", err.Error())
			}

			record := &kgo.Record{
				Topic: "us-awips-raw",
				Value: data,
			}
			kafkaClient.Produce(context.Background(), record, func(r *kgo.Record, err error) {
				if err != nil {
					slog.Error("failed to produce message to Kafka", "error", err.Error())
				} else {
					slog.Debug("message produced to Kafka", "topic", r.Topic)
				}
			})
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	kafkaClient.Close()

}

func (conf *XmppConfig) check() error {
	item := ""
	switch "" {
	case conf.Server:
		item = "server"
	case conf.User:
		item = "user"
	case conf.Pass:
		item = "pass"
	case conf.Resource:
		item = "resource"
	case conf.Room:
		item = "room"
	}
	if item != "" {
		return fmt.Errorf("xmpp %s missing in config", item)
	}
	return nil
}

func (conf *XmppConfig) serverName() string {
	return strings.Split(conf.Server, ":")[0]
}
