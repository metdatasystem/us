package main

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
)

type UGC struct {
	ID     int    `json:"id"`
	Code   string `json:"code"`
	State  string `json:"state"`
	Type   string `json:"type"`
	Number int    `json:"number"`
	Name   string `json:"name"`
}

type UGCStore struct {
	mu sync.Mutex

	data map[string]*UGC
	hub  *Hub
}

func NewUGCStore(hub *Hub) *UGCStore {
	return &UGCStore{
		data: map[string]*UGC{},
		hub:  hub,
	}
}

func (store *UGCStore) load() error {
	store.mu.Lock()
	defer store.mu.Unlock()

	rows, err := store.hub.db.Query(context.Background(), `
	SELECT id, ugc, name, state, type, number FROM postgis.ugcs WHERE valid_to IS NULL
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		ugc := UGC{}
		if err := rows.Scan(
			&ugc.ID,
			&ugc.Code,
			&ugc.Name,
			&ugc.State,
			&ugc.Type,
			&ugc.Number,
		); err != nil {
			return err
		}
		store.data[ugc.Code] = &ugc
	}

	log.Debug().Int("size", len(store.data)).Msg("loaded UGC data")

	return nil
}

func (store *UGCStore) findUGC(code string) *UGC {
	store.mu.Lock()
	defer store.mu.Unlock()

	ugc, ok := store.data[code]
	if !ok {
		return nil
	}

	return ugc
}
