package main

import (
	"context"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog/log"
	"github.com/twpayne/go-geos"
	pgxgeos "github.com/twpayne/pgx-geos"
)

type Hub struct {
	mu sync.Mutex

	register        chan *client
	unregister      chan *client
	subscription    chan *subscription
	inboundMessages chan *amqp.Delivery

	connections map[*client]bool
	managers    map[string]Manager

	wsUpgrader websocket.Upgrader
	db         *pgxpool.Pool
	rabbit     *amqp.Channel
	ugcStore   *UGCStore
}

func NewHub() (*Hub, error) {

	dbPool, err := newDatabasePool()
	if err != nil {
		return nil, err
	}

	rabbit, err := newRabbitChannel()
	if err != nil {
		return nil, err
	}

	err = initRabbit(rabbit)
	if err != nil {
		return nil, err
	}

	hub := &Hub{
		connections:  map[*client]bool{},
		register:     make(chan *client),
		unregister:   make(chan *client),
		subscription: make(chan *subscription),
		managers:     make(map[string]Manager),
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		db:     dbPool,
		rabbit: rabbit,
	}

	hub.ugcStore = NewUGCStore(hub)

	return hub, err
}

func (hub *Hub) registerConnection(c *client) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.connections[c] = true
}

func (hub *Hub) unregisterConnection(c *client) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	delete(hub.connections, c)
	for m := range c.subscriptions {
		if manager, ok := hub.managers[m]; ok {
			manager.Unsubscribe(c)
		}
	}
}

func (hub *Hub) subscribeClient(s *subscription) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	if s.Type == UNSUBSCRIBE {
		for _, topic := range s.Topics {
			if manager, ok := hub.managers[topic]; ok {
				manager.Unsubscribe(s.client)
			}
			delete(s.client.subscriptions, topic)
			log.Debug().Str("topic", topic).Msg("unsubscribed from topic")
		}
	} else {
		for _, topic := range s.Topics {
			if manager, ok := hub.managers[topic]; ok {
				manager.Subscribe(s.client)
				s.client.subscriptions[topic] = struct{}{}
				log.Debug().Str("topic", topic).Msg("subscribed to topic")
			}

		}
	}
}

func (hub *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}

	// Upgrade the connection
	ws, err := hub.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade connection")
		return
	}

	// Register the connection
	c := NewClient(ws, hub)
	hub.register <- c

	go c.listenWrite()
	c.listenRead()

}

func (hub *Hub) run() {
	AttachWarningManager(hub)

	err := hub.ugcStore.load()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialise hub UGC store")
		return
	}

	for name, manager := range hub.managers {
		err := manager.Load()
		if err != nil {
			log.Error().Err(err).Str("manager", name).Msg("failed to load manager")
			continue
		}
		go manager.Run()
		log.Info().Msgf("running %s manager", name)
	}

	log.Info().Msg("hub running")
	for {
		select {
		case c := <-hub.register:
			hub.registerConnection(c)
		case c := <-hub.unregister:
			hub.unregisterConnection(c)
		case s := <-hub.subscription:
			hub.subscribeClient(s)
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
