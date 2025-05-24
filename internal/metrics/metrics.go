package metrics

import (
	"net"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type IPCheckStatus string

type IPUpdateStatus string

const (
	IPCheckStatusSuccess  IPCheckStatus  = "success"
	IPCheckStatusFailure  IPCheckStatus  = "failure"
	IPUpdateStatusSuccess IPUpdateStatus = "success"
	IPUpdateStatusFailure IPUpdateStatus = "failure"
)

type AppMetrics struct {
	mutex       *sync.Mutex
	lastKnownIP net.IP

	// Going to use labels to expose currentIP
	currentIP             *prometheus.GaugeVec
	lastIPChangeTimestamp prometheus.Gauge
	ipCheckTotal          *prometheus.CounterVec
	ipUpdateTotal         *prometheus.CounterVec
	ipCheckDuration       *prometheus.HistogramVec
	ipUpdateDuration      *prometheus.HistogramVec
}

func NewAppMetrics() *AppMetrics {
	m := AppMetrics{
		currentIP: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "current_ip",
				Help: "Current IP address",
			},
			[]string{"value"},
		),
		lastIPChangeTimestamp: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "public_ip_last_change_timestamp_seconds",
				Help: "Timestamp of the last detected public IP address change",
			},
		),
		ipCheckTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ip_check_total",
				Help: "Total number of IP check operations",
			},
			[]string{"status"},
		),
		ipUpdateTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ip_update_total",
				Help: "Total number of IP update operations",
			},
			[]string{"status"},
		),
		ipCheckDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ip_check_duration_seconds",
				Help:    "Duration of public IP address check operations",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 13), // From 1ms to ~4096ms
			},
			[]string{"status"},
		),
		ipUpdateDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ip_update_latency_seconds",
				Help:    "Duration of public IP address update operations",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 13), // From 1ms to ~4096ms
			},
			[]string{"status"},
		),
	}

	m.lastKnownIP = net.IPv4zero

	return &m
}

func (m *AppMetrics) UpdateCurrentIP(ip net.IP) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !ip.Equal(m.lastKnownIP) {
		m.currentIP.WithLabelValues(ip.String()).Set(1)
		m.lastIPChangeTimestamp.SetToCurrentTime()
	}
}

func (m *AppMetrics) IncIPCheckTotal(status IPCheckStatus) {
	m.ipCheckTotal.WithLabelValues(string(status)).Inc()
}

func (m *AppMetrics) IncIPUpdateTotal(status IPUpdateStatus) {
	m.ipUpdateTotal.WithLabelValues(string(status)).Inc()
}

func (m *AppMetrics) ObserveIPCheckLatency(duration time.Duration, status IPCheckStatus) {
	m.ipCheckDuration.WithLabelValues(string(status)).Observe(duration.Seconds())
}

func (m *AppMetrics) ObserveIPUpdateLatency(duration time.Duration, status IPUpdateStatus) {
	m.ipUpdateDuration.WithLabelValues(string(status)).Observe(duration.Seconds())
}

func (m *AppMetrics) GetMetrics() []prometheus.Collector {
	return []prometheus.Collector{
		m.currentIP,
		m.lastIPChangeTimestamp,
		m.ipCheckTotal,
		m.ipUpdateTotal,
		m.ipCheckDuration,
		m.ipUpdateDuration,
	}
}
