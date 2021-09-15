package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
	"github.com/ryszard/sds011/go/sds011"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Topic          string
	SensorPortPath string
	CycleMinutes   uint8
	MqttBroker     string
}

func main() {
	log.SetFormatter(&log.JSONFormatter{})
	godotenv.Load()
	c := Config{
		Topic:          os.Getenv("TOPIC"),
		SensorPortPath: "/dev/ttyUSB0",
		CycleMinutes:   1,
		MqttBroker:     fmt.Sprintf("tcp://%s:1883", os.Getenv("MQTT_BROKER")),
	}

	opts := mqtt.NewClientOptions().AddBroker(c.MqttBroker)
	opts.AutoReconnect = true
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	sensor, err := sds011.New(c.SensorPortPath)
	if err != nil {
		log.Fatalf("ERROR: sds011.New, %v", err)
	}
	defer sensor.Close()

	for {
		point, err := sensor.Get()
		if err != nil {
			log.Errorf("ERROR: sensor.Get: %v", err)
			continue
		}
		log.Printf("Timestamp: %v,PM25: %v,PM10: %v\n", point.Timestamp.Format(time.RFC3339), point.PM25, point.PM10)
		// fmt.Fprintf(os.Stdout, "%v,%v,%v\n", point.Timestamp.Format(time.RFC3339), point.PM25, point.PM10)
		pointJSON, err := json.Marshal(point)
		if err != nil {
			log.Printf("ERROR: Marshal: %v", err)
			continue
		}
		if token := client.Publish(c.Topic, 0, false, pointJSON); token.Wait() && token.Error() != nil {
			log.Info(token.Error())
		}
	}
}
