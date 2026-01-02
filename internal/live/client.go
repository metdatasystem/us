package main

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var (
	// Time allowed to write a message to the peer.
	WriteWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	PongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	PingPeriod = (PongWait * 9) / 10
	// Maximum message size allowed from peer.
	MaxMessageSize int64 = 64 * 1024
)

const (
	SUBSCRIBE   string = "SUBSCRIBE"
	UNSUBSCRIBE string = "UNSUBSCRIBE"
)

type subscription struct {
	client *client
	Type   string   `json:"type"`
	Topics []string `json:"topics"`
}

type client struct {
	ws            *websocket.Conn
	send          chan []byte
	hub           *Hub
	closed        bool
	subscriptions map[string]struct{}
}

func NewClient(ws *websocket.Conn, hub *Hub) *client {
	return &client{
		ws:            ws,
		send:          make(chan []byte),
		hub:           hub,
		subscriptions: map[string]struct{}{},
	}
}

func (c *client) close() {
	if !c.closed {
		if err := c.ws.Close(); err != nil {
			log.Debug().Err(err).Msg("websocket already closed")
		}
		close(c.send)
		c.closed = true
	}
}

func (c *client) listenRead() {
	defer func() {
		c.hub.unregister <- c
		c.close()
	}()
	c.ws.SetReadLimit(MaxMessageSize)
	if err := c.ws.SetReadDeadline(time.Now().Add(PongWait)); err != nil {
		log.Error().Err(err).Msg("failed to set socket read deadline")
	}
	c.ws.SetPongHandler(func(string) error {
		return c.ws.SetReadDeadline(time.Now().Add(PongWait))
	})
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			log.Debug().Err(err).Msg("ws read message error")
			break
		}

		s := &subscription{client: c}
		if err := json.Unmarshal(message, s); err != nil {
			log.Error().Err(err).Str("data", string(message)).Msg("invalid data sent for subscription")
			continue
		}
		c.hub.subscription <- s
	}
}

func (c *client) listenWrite() {
	write := func(mt int, payload []byte) error {
		if err := c.ws.SetWriteDeadline(time.Now().Add(WriteWait)); err != nil {
			return err
		}
		return c.ws.WriteMessage(mt, payload)
	}
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				if err := write(websocket.CloseMessage, []byte{}); err != nil {
					log.Debug().Err(err).Msg("socket already closed")
				}
				return
			}
			if err := write(websocket.TextMessage, message); err != nil {
				log.Debug().Err(err).Msg("failed to write socket message")
				return
			}
		case <-ticker.C:
			if err := write(websocket.PingMessage, []byte{}); err != nil {
				log.Debug().Err(err).Msg("failed to ping socket")
				return
			}
		}
	}
}
