package metrics

import (
	"net"

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
	// Going to use labels to expose currentIP
	currentIP *prometheus.GaugeVec

	ipCheckTotal  *prometheus.CounterVec
	ipUpdateTotal *prometheus.CounterVec
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
		ipCheckTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ip_check_total",
				Help: "Total number of IP checks",
			},
			[]string{"status"},
		),
		ipUpdateTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ip_update_total",
				Help: "Total number of IP updates",
			},
			[]string{"status"},
		),
	}

	return &m
}

func (m *AppMetrics) UpdateCurrentIP(ip net.IP) {
	m.currentIP.WithLabelValues(ip.String()).Set(1)
}

func (m *AppMetrics) IncIPCheckTotal(status IPCheckStatus) {
	m.ipCheckTotal.WithLabelValues(string(status)).Inc()
}

func (m *AppMetrics) IncIPUpdateTotal(status IPUpdateStatus) {
	m.ipUpdateTotal.WithLabelValues(string(status)).Inc()
}

func (m *AppMetrics) GetMetrics() []prometheus.Collector {
	return []prometheus.Collector{
		m.currentIP,
		m.ipCheckTotal,
		m.ipUpdateTotal,
	}
}
