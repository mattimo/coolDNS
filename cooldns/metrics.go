// The CoolDNS Project. The simple dynamic dns server and update service.
// Copyright (C) 2014 The CoolDNS Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.package main

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
