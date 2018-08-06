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

package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"github.com/cinience/node_exporter/collector"
	"gopkg.in/alecthomas/kingpin.v2"
	//"github.com/robfig/cron"
	"os"
	"os/signal"
	"syscall"
	"github.com/robfig/cron"
	"strings"
)

func init() {
	prometheus.MustRegister(version.NewCollector("node_exporter"))
}

func handler(w http.ResponseWriter, r *http.Request) {
	filters := r.URL.Query()["collect[]"]
	log.Debugln("collect query:", filters)

	nc, err := collector.NewNodeCollector(filters...)
	if err != nil {
		log.Warnln("Couldn't create", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Couldn't create %s", err)))
		return
	}

	registry := prometheus.NewRegistry()
	err = registry.Register(nc)
	if err != nil {
		log.Errorln("Couldn't register collector:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Couldn't register collector: %s", err)))
		return
	}

	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		registry,
	}
	// Delegate http serving to Prometheus client library, which will call collector.Collect.
	h := promhttp.InstrumentMetricHandler(
		registry,
		promhttp.HandlerFor(gatherers,
			promhttp.HandlerOpts{
				ErrorLog:      log.NewErrorLogger(),
				ErrorHandling: promhttp.ContinueOnError,
			}),
	)
	h.ServeHTTP(w, r)
}

func collectorPush(gatewayPath string) {
	res := ResponseWriterDelegate{}
	req, err := http.NewRequest("GET", "http://localhost/metrics", nil)
	if err != nil {
		log.Errorln(err)
	}
	req.Header.Set("Accept-Encoding", "")
	handler(&res, req)
	//log.Infoln(resp.Context)

	// http://pushgateway.example.org:9091/metrics/job/some_job/instance/some_instance
	url := gatewayPath
	_, err = http.Post(url, "application/octet-stream", strings.NewReader(res.Context))
	if err != nil {
		log.Errorln(err)
	}
}

func main() {
	var (
		listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9100").String()
		metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		gatewayPath   = kingpin.Flag("pushgateway.uri", "Path under which to expose metrics.").Default("http://localhost:9091/metrics/job/nls/instance/serviceName").String()
		cronInterval = kingpin.Flag("pushgateway.interval", "Run Cron").Default("0").String()
		isWeb = kingpin.Flag("web", "Run Web ").Bool()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("node_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting node_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	// This instance is only used to check collector creation and logging.
	nc, err := collector.NewNodeCollector()
	if err != nil {
		log.Fatalf("Couldn't create collector: %s", err)
	}
	log.Infof("Enabled collectors:")
	collectors := []string{}
	for n := range nc.Collectors {
		collectors = append(collectors, n)
	}
	sort.Strings(collectors)
	for _, n := range collectors {
		log.Infof(" - %s", n)
	}


	http.HandleFunc(*metricsPath, handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Node Exporter</title></head>
			<body>
			<h1>Node Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	if *cronInterval != "0" {
		log.Infoln("Crontab every " + *cronInterval + " second")
		c := cron.New()
		cronRule := fmt.Sprintf("*/%s * * * * *", *cronInterval)
		c.AddFunc(cronRule, func() {
			collectorPush(*gatewayPath)
		})
		c.Start()
	}

	if *isWeb {
		log.Infoln("Listening on", *listenAddress)
		err = http.ListenAndServe(*listenAddress, nil)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGUSR1, syscall.SIGUSR2)
		s := <-c
		log.Infoln("Quit ", s)
	}
}
