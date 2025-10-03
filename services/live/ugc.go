package main

import (
	"sync"

	"github.com/metdatasystem/us/pkg/db"
	"github.com/metdatasystem/us/pkg/models"
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

	data, err := db.GetAllValidUGCMinimal(store.hub.db)
	if err != nil {
		return err
	}

	for _, u := range data {
		store.data[u.UGC] = modelToUGC(u)
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

func modelToUGC(u *models.UGCMinimal) *UGC {
	return &UGC{
		ID:     u.ID,
		Code:   u.UGC,
		State:  u.State,
		Type:   u.Type,
		Number: u.Number,
		Name:   u.Name,
	}
}
