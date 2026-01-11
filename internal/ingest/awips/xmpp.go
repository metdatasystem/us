package awips

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/muc"
	"mellium.im/xmpp/mux"
)

func X() {
	fmt.Println("yeah")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	xmppConfig := XmppConfig{
		Server:   os.Getenv("NWWSOI_SERVER"),
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

	j, err := jid.Parse(fmt.Sprintf("%s@%s", xmppConfig.User, xmppConfig.Server))
	if err != nil {
		log.Error().Err(err).Msg("failed to parse jid")
		return
	}

	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// dialCtx, dialCtxCancel := context.WithTimeout(ctx, 30*time.Second)
	// session, err := xmpp.DialClientSession(
	// 	dialCtx, j,
	// 	xmpp.BindResource(),
	// 	xmpp.StartTLS(&tls.Config{
	// 		ServerName: j.Domain().String(),
	// 		MinVersion: tls.VersionTLS12,
	// 	}),
	// 	xmpp.SASL("", xmppConfig.Pass, sasl.ScramSha1Plus, sasl.ScramSha1, sasl.Plain),
	// )
	// dialCtxCancel()
	// if err != nil {
	// 	log.Error().Err(err).Msg("failed to create client session")
	// }

	// defer func() {
	// 	if err := session.Close(); err != nil {
	// 		log.Error().Err(err).Msg("failed to close session")
	// 	}
	// 	if err := session.Conn().Close(); err != nil {
	// 		log.Error().Err(err).Msg("failed to close connection")
	// 	}
	// }()

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dialCancel()

	log.Info().Msg("dialling tcp...")

	conn, err := dial.Client(dialCtx, "tcp", j)
	if err != nil {
		log.Error().Err(err).Msg("could not dial tcp")
		return
	}

	negotiator := xmpp.NewNegotiator(func(s *xmpp.Session, sc *xmpp.StreamConfig) xmpp.StreamConfig {
		return xmpp.StreamConfig{
			Features: []xmpp.StreamFeature{
				xmpp.BindResource(),
				xmpp.StartTLS(&tls.Config{
					ServerName: j.Domain().String(),
					MinVersion: tls.VersionTLS12,
				}),
				xmpp.SASL(xmppConfig.User, xmppConfig.Pass,
					sasl.ScramSha256Plus, sasl.ScramSha256,
					sasl.ScramSha1Plus, sasl.ScramSha1,
					sasl.Plain,
				),
			},
		}
	})

	log.Info().Msg("attempting to connect")

	session, err := xmpp.NewSession(dialCtx, j.Domain(), j, conn, 0, negotiator)
	dialCancel()
	if err != nil {
		log.Error().Err(err).Msg("failed to create session")
		return
	}

	mucClient := &muc.Client{}
	muxx := mux.New("jabber:client", muc.HandleClient(mucClient))
	go func() {
		err := session.Serve(muxx)
		if err != nil {
			log.Error().Err(err).Msg("failed to serve")
		}
	}()

	log.Info().Msg("yes?")

	<-ctx.Done()
	log.Info().Msg("shutting down")
}
