package internal

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/metdatasystem/us/pkg/streaming"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xmppo/go-xmpp"
)

type XmppConfig struct {
	Server   string
	Room     string
	User     string
	Pass     string
	Resource string
}

func NWWS(logLevel zerolog.Level) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	zerolog.SetGlobalLevel(logLevel)

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
		log.Error().Err(err).Msg("NWWS configuration is invalid")
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
		log.Error().Err(err).Msg("failed to create XMPP client")
		return
	}

	log.Info().Msgf("connected to %s", xmppConfig.Server)

	// Send presence to the room
	_, err = client.SendOrg(fmt.Sprintf(`<presence xml:lang='en' from='%s@%s' to='%s@%s/%s'><x></x></presence>`, xmppConfig.User, xmppConfig.Server, xmppConfig.Resource, xmppConfig.Room, xmppConfig.User))
	if err != nil {
		log.Error().Err(err).Msg("failed to send presence")
		return
	}

	// Track time of last received message
	var lastReceived int64
	// Messages per minute
	var messageRate atomic.Uint64
	// Channel to share messages

	producer, err := NewProducer()
	if err != nil {
		log.Error().Err(err).Msg("failed to create producer")
		return
	}

	// XMPP listening
	go func() {
		defer client.Close()

		log.Info().Msgf("listening to %s", xmppConfig.Server)
		for {
			select {
			// Stop
			case <-ctx.Done():
				log.Warn().Msg("shutting down XMPP client")
				close(producer.messages)
				return
			// Receive messages
			default:
				chat, err := client.Recv()
				if err != nil {
					log.Error().Err(err).Msg("failed to receive messgae")
					continue
				}

				switch v := chat.(type) {
				case xmpp.Chat:
					for _, elem := range v.OtherElem {
						// NWWS-OI uses 'x' as the XML element containing the raw text
						if elem.XMLName.Local == "x" {
							now := time.Now()
							log.Debug().Time("received", now).Msg("received message")
							// Get attributes
							var issued *time.Time = nil
							var ttaaii string
							var cccc string
							var awips string
							for _, attr := range elem.Attr {
								switch attr.Name.Local {
								case "issue":
									t, err := time.Parse("2006-01-02T15:04:05Z", attr.Value)
									if err != nil {
										log.Error().Err(err).Msg("failed to parse x element issue time")
										continue
									}
									issued = &t
								case "ttaaii":
									ttaaii = attr.Value
								case "cccc":
									cccc = attr.Value
								case "awipsid":
									awips = attr.Value
								}
							}

							// Remove extra newlines
							text := strings.ReplaceAll(elem.String(), "\n\n", "\n")

							// Build the message
							message := streaming.AWIPSRaw{
								Issued: *issued,
								TTAAII: ttaaii,
								CCCC:   cccc,
								AWIPS:  awips,
								Text:   text,
							}
							data, err := json.Marshal(message)
							if err != nil {
								log.Error().Err(err).Msg("failed to marshal awips raw message")
								continue
							}

							// Update last received time
							atomic.StoreInt64(&lastReceived, now.Unix())
							// Share the message
							producer.messages <- Message{"application/json", data}
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
				log.Warn().Uint64("messagesLastMinute", messageRate.Load()).Msg("no messages received in the last minute")
			} else {
				log.Debug().Uint64("messagesLastMinute", messageRate.Load()).Msg("healthy")
			}
			// Reset message rate
			messageRate.Swap(0)
		}
	}()

	go producer.Run()

	<-ctx.Done()
	log.Warn().Msg("shutting down")
	producer.Stop()

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
