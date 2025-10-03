package main

import (
	"encoding/json"
	"time"
)

const (
	EnvelopeNew     = "NEW"
	EnvelopeUpdate  = "UPDATE"
	EnvelopeDelete  = "DELETE"
	EnvelopeInitial = "INIT"
)

type Envelope struct {
	Type      string          `json:"type"`
	Product   string          `json:"product"`
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}
