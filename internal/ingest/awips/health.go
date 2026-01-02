package awips

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Health struct {
	NWWSReceived prometheus.Counter
	NWWSProduced prometheus.Counter
	NWWSPing     prometheus.Gauge
}

func NewHealth() *Health {
	return &Health{
		NWWSReceived: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nwws_received",
			Help: "Total number of NWWS messages received",
		}),
		NWWSProduced: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nwws_produced",
			Help: "Total number of NWWS messages produced",
		}),
		NWWSPing: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "nwws_ping",
			Help: "Indicates the NWWS ping status",
		}),
	}
}
