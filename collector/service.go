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
	"os/exec"
	"bytes"
	"os"
	"bufio"
	"strings"
	"strconv"
	"net/url"
	"net/http"
	"io/ioutil"
	"errors"
	"encoding/json"
)

var (
	svrPATH = kingpin.Flag("collector.svr.path", "service info .").Default("./metrics.sh").String()
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

func (c *serviceCollector) checkPathType(path string) string  {
	_, err := os.Stat(path)
	if err == nil || os.IsExist(err) {
		return "FILE"
	}

	_, err = url.Parse(path)
	if err == nil {
		return "HTTP"
	}
	return "NONE"
}

func (c *serviceCollector) getSvrInfoByShell(path string, rst map[string]float64) (error) {
	os.Chmod(path, os.FileMode(0777))
	cmd := exec.Command(*svrPATH)
	var sout, serr bytes.Buffer
	cmd.Stdout = &sout
	cmd.Stderr = &serr
	err := cmd.Run()
	info := sout.String()
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(strings.NewReader(info))
	for scanner.Scan() {
		sp := strings.Split(scanner.Text(), " ")
		rst[sp[0]], err = strconv.ParseFloat(sp[1], 64)
		if err != nil {
			return err
		}
	}
	err = scanner.Err()
	return err
}

func (c *serviceCollector) getSvrInfoByHttp(path string, rst map[string]float64) (error) {
	resp, err := http.Get(path)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := resp.Body.Close(); err != nil {
		return err
	}
	info := string(b)


	var dat map[string]interface{}
	err = json.Unmarshal([]byte(info), &dat)
	if err == nil {
		for k, v := range dat {
			switch v.(type) {
			case string:
				rst[k], err = strconv.ParseFloat(v.(string), 64)
				break;
			case float64:
			case float32:
			case int32:
			case int64:
			case int:
				rst[k] = float64(v.(int))
				break;
			}
		}
	} else {
		scanner := bufio.NewScanner(strings.NewReader(info))
		for scanner.Scan() {
			sp := strings.Split(scanner.Text(), " ")
			if len(sp) != 2 {
				return errors.New("invalid metrics")
			}
			rst[sp[0]], err = strconv.ParseFloat(sp[1], 64)
			if err != nil {
				return err
			}
		}
		err = scanner.Err()
	}


	return err
}

func (c *serviceCollector) getSvrInfo() (map[string]float64, error) {
	path := *svrPATH
	rst := make(map[string]float64)

	pathType := c.checkPathType(path)
	if pathType == "FILE" {
		return rst, c.getSvrInfoByShell(path, rst)
	} else if pathType == "HTTP" {
		return rst, c.getSvrInfoByHttp(path, rst)
	}

	return rst, errors.New("Unsupported path:" + path)
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
