package main

import (
	"fmt"
	"log"
	"bufio"
	"os"
	"observations"
	"encoding/json"
	"strings"
	"flag"
	"net/http"
	"bytes"
	"io/ioutil"
)

var BluefloodIngestionTtl = flag.Int("bluefloodIngestionTTL", 172800, "How long the data lives in Blueflood")
var BluefloodDumpPath = flag.String("dumpPath", "dump.json", "Where to find the JSON dump, if reading from a file.")
var BluefloodUrl = flag.String("url", "http://qe01.metrics-ingest.api.rackspacecloud.com/v2.0/706456/ingest/multi", "Where Blueflood lives on the Internet")

type BluefloodMetric struct {
	TenantId	*string	`json:"tenantId"`
	CollectionTime	int64	`json:"collectionTime"`
	TtlInSeconds	int	`json:"ttlInSeconds"`
	MetricValue	interface{}	`json:"metricValue"`
	MetricName	string	`json:"metricName"`
}

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

	client := &http.Client{}

	for metricName, metric := range obs.Metrics {
		metricName := getMetricName(metricName, obs)

		v, isNumeric := metric.Value.(float64)
		if !isNumeric {
			continue
		}

		m, err := json.Marshal(&BluefloodMetric{
			TenantId: tenantId,
			CollectionTime: collectionTime,
			TtlInSeconds: *BluefloodIngestionTtl,
			MetricValue: v,
			MetricName: metricName,
		})
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("%s", string(m))

		if metric.Value != nil {
			resp, err := client.Post(*BluefloodUrl, "application/json", bytes.NewReader(m))
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


func main() {
	flag.Parse()

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

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		if err != nil {
			log.Fatal(err)
		}
		relayJsonToBlueflood([]byte(scanner.Text()))
	}
}

