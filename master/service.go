package master

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

type Service struct {
	StopSignal chan os.Signal
	host       string
	port       int
	username   string
	password   string
	Influxdb   influxdb2.Client
}

type Config struct {
	MqttHost       string
	MqttPort       int
	MqttUsername   string
	MqttPassword   string
	InfluxdbUrl    string
	InfluxdbApikey string
}

func NewService(config Config) *Service {
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	return &Service{
		StopSignal: stopSignal,
		host:       config.MqttHost,
		port:       config.MqttPort,
		username:   config.MqttUsername,
		password:   config.MqttPassword,
		Influxdb:   influxdb2.NewClient(config.InfluxdbUrl, config.InfluxdbApikey),
	}
}

func (srv *Service) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-srv.StopSignal
		fmt.Println("Stopping")
		cancel()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		client, err := createMqttClient(srv.host, srv.port, srv.username, srv.password)
		if err != nil {
			fmt.Printf("Failed to create mqtt client '%s'\n", err.Error())
		}

		token := client.Subscribe("topic", 1, srv.handleTopic)
		token.Wait()

		token = client.Subscribe("Aranet/349681001757/sensors/10341A/json/measurements", 1, func(c mqtt.Client, m mqtt.Message) {
			srv.handleOutdoorTemperature(ctx, c, m)
		})
		token.Wait()

		token = client.Subscribe("query", 1, func(c mqtt.Client, m mqtt.Message) {
			srv.queryData(ctx, c, m)
		})
		token.Wait()

		<-ctx.Done()
		client.Disconnect(250)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		mux := http.NewServeMux()
		mux.HandleFunc("/api/outdoors", srv.getOutdoors)
		fs := http.FileServer(http.Dir("./public"))
		mux.Handle("/", fs)

		err := http.ListenAndServe(":1884", mux)
		if err != nil {
			fmt.Printf("HTTP Error: '%s'\n", err.Error())
		}
	}()

	wg.Wait()
}
