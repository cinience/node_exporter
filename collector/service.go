// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !notime

package collector

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	svrURL = kingpin.Flag("collector.svr.url", "service url.").Default("http://localhost:7001/metrics").String()
)

type serviceCollector struct {
	desc *prometheus.Desc
}

func init() {
	registerCollector("svr", defaultEnabled, NewServiceCollector)
}

func NewServiceCollector() (Collector, error) {
	return &serviceCollector{}, nil
}

func (c *serviceCollector) getSvrInfo() (map[string]float64, error) {
	return map[string]float64{
		"service_tps":      float64(0.1),
	}, nil
}

func (c *serviceCollector) Update(ch chan<- prometheus.Metric) error {
	svrInfo, err := c.getSvrInfo()
	if err != nil {
		return fmt.Errorf("couldn't get service info: %s", err)
	}
	log.Debugf("Set service: %#v", svrInfo)
	for k, v := range svrInfo {
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "svr", k),
				fmt.Sprintf("Memory information field %s.", k),
				nil, nil,
			),
			prometheus.GaugeValue, v,
		)
	}
	return nil
}
