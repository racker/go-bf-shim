package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"observations"
	"os"
	"strings"
	"sync"
)

var BluefloodIngestionTtl = flag.Int("bluefloodIngestionTTL", 172800, "How long the data lives in Blueflood")
var BluefloodDumpPath = flag.String("dumpPath", "dump.json", "Where to find the JSON dump, if reading from a file.")
var BluefloodUrl = flag.String("url", "http://qe01.metrics-ingest.api.rackspacecloud.com/v2.0/706456/ingest/multi", "Where Blueflood lives on the Internet")
var Jobs = flag.Int("jobs", 1, "How many concurrent connections to establish to Blueflood")

type BluefloodMetric struct {
	TenantId       *string `json:"tenantId"`
	CollectionTime int64   `json:"collectionTime"`
	TtlInSeconds   int     `json:"ttlInSeconds"`
	MetricValue    float64 `json:"metricValue"`
	MetricName     string  `json:"metricName"`
}

type BluefloodBuffer struct {
	queue []*BluefloodMetric
}

func (bfb *BluefloodBuffer) enqueue(m *BluefloodMetric) {
	if m != nil {
		bfb.queue = append(bfb.queue, m)
	}
	if len(bfb.queue) > 1500 {
		bfb.send()
	}
}

func (bfb *BluefloodBuffer) send() {
	m, err := json.Marshal(bfb.queue)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%s", string(m))

	buf := bytes.NewBuffer(m)
	req, err := http.NewRequest("POST", *BluefloodUrl, buf)
	req.Header.Set("Content-Type", "application/json")

	tr := &http.Transport{
		DisableCompression: true,
	}
	c := &http.Client{Transport: tr}

	resp, err := c.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	if (resp.StatusCode < 200) || (300 <= resp.StatusCode) {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Error trying to read body of HTTP error response: %s", err)
		}
		log.Printf("Error body: %q", string(body))
		log.Fatalf("Expected 2xx response code; got %d", resp.StatusCode)
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
}

var bluefloodBuffer *BluefloodBuffer

func getMetricName(name string, obs *observations.Observation) string {
	sl := []string{
		"rackspace.monitoring.entities",
		obs.EntityId,
		"checks",
	}

	if obs.CheckType != nil {
		sl = append(sl, *obs.CheckType)
	}

	sl = append(sl, obs.CheckId)

	if obs.MonitoringZoneId != nil {
		sl = append(sl, *obs.MonitoringZoneId)
	}

	prefix := strings.Join(sl, ".")
	return fmt.Sprintf("%s%s", prefix, name)
}

func forwardToBlueflood(obs *observations.Observation) {
	tenantId := obs.TenantId
	collectionTime := obs.Timestamp

	for metricName, metric := range obs.Metrics {
		metricName := getMetricName(metricName, obs)

		v, isNumeric := (metric.Value).(float64)
		if !isNumeric {
			continue
		}

		if metric.Value != nil {
			bluefloodBuffer.enqueue(&BluefloodMetric{
				TenantId:       tenantId,
				CollectionTime: collectionTime,
				TtlInSeconds:   *BluefloodIngestionTtl,
				MetricValue:    v,
				MetricName:     metricName,
			})
		}
	}
}

func relayJsonToBlueflood(js []byte) {
	var o *observations.Observation

	err := json.Unmarshal(js, &o)
	if err != nil {
		log.Fatal(err)
	}

	forwardToBlueflood(o)
}

type Job struct {
	jsonInput chan []byte
	fin chan bool
	wg *sync.WaitGroup
}

func newJob(waitGroup *sync.WaitGroup) *Job {
	return &Job {
		jsonInput: make(chan []byte),
		fin: make(chan bool),
		wg: waitGroup,
	}
}

func (j *Job) Start() {
	log.Print("Job started")
	for {
		select {
		case js := <-j.jsonInput:
			relayJsonToBlueflood(js)
		case _ = <-j.fin:
			log.Print("Job terminating")
			j.wg.Done()
		}
	}
}

func main() {
	flag.Parse()

	bluefloodBuffer = new(BluefloodBuffer)

	log.Print("Blueflood shim started.")
	defer log.Print("Blueflood shim terminated.")

	if BluefloodDumpPath == nil {
		log.Fatal("I need a -dumpPath")
	}

	if BluefloodUrl == nil {
		log.Fatal("I need a -url")
	}

	f, err := os.Open(*BluefloodDumpPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	wg := &sync.WaitGroup{}
	wg.Add(*Jobs)
	workers := make([]*Job, *Jobs)

	for i := 0; i < *Jobs; i++ {
		j := newJob(wg)
		go j.Start()
		workers[i] = j
	}

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	j := 0
	for scanner.Scan() {
		if err != nil {
			log.Fatal(err)
		}
		t := scanner.Text()
		workers[j].jsonInput <- []byte(t)
		j = (j + 1) % *Jobs
	}
	j = 0
	for j < *Jobs {
		workers[j].fin <- true
	}
	wg.Wait()
}
