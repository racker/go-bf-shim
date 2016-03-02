package main

import (
	"bufio"
	"encoding/json"
	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"log"
	"observations"
	"os"
	"testing"
)

func TestKafkaRead(t *testing.T) {

	log.Print("Loading test data...")

	file, err := os.Open("../../dump.json")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	consumer := mocks.NewConsumer(t, &sarama.Config{ChannelBufferSize: 85000})

	for scanner.Scan() {
		if err != nil {
			log.Fatal(err)
		}
		json := scanner.Text()
		expectation := consumer.ExpectConsumePartition("observations.json", 0, sarama.OffsetOldest)
		expectation.YieldMessage(&sarama.ConsumerMessage{Value: []byte(json)})
	}

	log.Print("Consuming test data...")
	reader, err := consumer.ConsumePartition("observations.json", 0, sarama.OffsetOldest)
	if err != nil {
		t.Fatal(err)
	}

	for {
		message := <-reader.Messages()
		var obj *observations.Observation

		log.Print(string(message.Value))
		err := json.Unmarshal(message.Value, &obj)
		if err != nil {
			log.Fatal(err)
		}

		// log.Print(obj)
	}

}
