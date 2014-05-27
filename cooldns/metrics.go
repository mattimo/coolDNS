package cooldns

import (
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/influxdb"
)

type MetricsHandle interface {
	DnsEvent()
	DatabaseEvent()
	HttpEvent()
	HttpTime(func())
}

// Configuration of the InfluxDB host where all metrics are stored in
type InfluxConfig struct {
	Host     string // Hostname:port of the Database
	Database string // Database name
	Username string
	Password string
}

type InfluxMHandle struct {
	dns     metrics.Meter
	db      metrics.Meter
	http    metrics.Meter
	httpLat metrics.Timer
}

func NewInfluxMetrics(config *InfluxConfig) *InfluxMHandle {
	dnsLoad := metrics.NewMeter()
	metrics.Register("dnsLoad", dnsLoad)

	databaseLoad := metrics.NewMeter()
	metrics.Register("dbLoad", databaseLoad)

	httpLoad := metrics.NewMeter()
	metrics.Register("httpLoad", httpLoad)

	httpLat := metrics.NewTimer()
	metrics.Register("httpTime", httpLat)

	go influxdb.Influxdb(metrics.DefaultRegistry, 10e9, &influxdb.Config{
		Host:     config.Host,
		Database: config.Database,
		Username: config.Username,
		Password: config.Password,
	})
	return &InfluxMHandle{
		dns:     dnsLoad,
		db:      databaseLoad,
		http:    httpLoad,
		httpLat: httpLat,
	}
}

func (m *InfluxMHandle) DnsEvent() {
	m.dns.Mark(1)
}

func (m *InfluxMHandle) DatabaseEvent() {
	m.db.Mark(1)
}

func (m *InfluxMHandle) HttpEvent() {
	m.http.Mark(1)
}

func (m *InfluxMHandle) HttpTime(f func()) {
	m.httpLat.Time(f)
}

type DummyMHandle struct {
}

func NewDummyMetrics() *DummyMHandle {
	return &DummyMHandle{}
}

func (m *DummyMHandle) DnsEvent() {
}

func (m *DummyMHandle) DatabaseEvent() {
}

func (m *DummyMHandle) HttpEvent() {
}

func (m *DummyMHandle) HttpTime(f func()) {
	f()
}
