package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/dmytroyunyk/mikrotik-defender/internal/storage"
	"github.com/dmytroyunyk/mikrotik-defender/pkg/utils"
)

type Metrics struct {
	db     *storage.DB
	logger *utils.Logger

	totalEvents prometheus.Gauge
	blockedIPs  prometheus.Gauge
	events24h   prometheus.Gauge

	totalBlocked prometheus.Counter
}

func New(db *storage.DB, logger *utils.Logger) *Metrics {
	m := &Metrics{
		db:     db,
		logger: logger,

		totalEvents: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mikrotik_total_events",
			Help: "Total number of attack events recorder",
		}),

		blockedIPs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mikrotik_blocked_ips",
			Help: "Curent number of blocked IP addresses",
		}),

		events24h: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "mikrotik_events_24h",
			Help: "Total number if IP adresses blocked since startup",
		}),

		totalBlocked: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "mikrotik_total_blocked_counter",
			Help: "Total number of IP addresses blocked since startup",
		}),
	}

	prometheus.MustRegister(
		m.totalEvents,
		m.blockedIPs,
		m.events24h,
		m.totalBlocked,
	)

	return m
}

func (m *Metrics) Update() {
	stats, err := m.db.GetStats()
	if err != nil {
		m.logger.Error("failed to update metrics", "error", err)
		return
	}

	m.totalEvents.Set(float64(stats["total_events"]))
	m.blockedIPs.Set((float64(stats["blocked_ips"])))
	m.events24h.Set(float64(stats["events_24h"]))
}

func (m *Metrics) RecordBlock() {
	m.totalBlocked.Inc()
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}
