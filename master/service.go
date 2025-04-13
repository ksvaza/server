package master

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	httprouter "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Service struct {
	StopSignal chan os.Signal
	host       string
	port       int
	username   string
	password   string
	Influxdb   influxdb2.Client
	mqtt       mqtt.Client
	log        Log
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
		logrus.Info("Stopping")
		cancel()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Panic")
			}
		}()

		var err error
		srv.mqtt, err = createMqttClient(srv.host, srv.port, srv.username, srv.password)
		if err != nil {
			logrus.WithError(errors.Wrap(err, "MQTT")).Error("Error")
		}

		token := srv.mqtt.Subscribe("#", 1, srv.handleTopic)
		token.Wait()
		if err := token.Error(); err != nil {
			logrus.WithError(errors.Wrap(err, "MQTT")).Error("Error")
		}

		token = srv.mqtt.Subscribe("Aranet/349681001757/sensors/10341A/json/measurements", 1, func(c mqtt.Client, m mqtt.Message) {
			srv.handleOutdoorTemperature(ctx, c, m)
		})
		token.Wait()
		if err := token.Error(); err != nil {
			logrus.WithError(errors.Wrap(err, "MQTT")).Error("Error")
		}

		token = srv.mqtt.Subscribe("query", 1, func(c mqtt.Client, m mqtt.Message) {
			srv.queryData(ctx, c, m)
		})
		token.Wait()
		if err := token.Error(); err != nil {
			logrus.WithError(errors.Wrap(err, "MQTT")).Error("Error")
		}

		// Subscribe to car-server steam topics

		token = srv.mqtt.Subscribe("PSU_OUT/#", 1, func(c mqtt.Client, m mqtt.Message) {
			srv.handleTopicPSU(ctx, c, m)
		})
		token.Wait()
		if err := token.Error(); err != nil {
			logrus.WithError(errors.Wrap(err, "MQTT")).Error("Error")
		}

		token = srv.mqtt.Subscribe("GPS_OUT/#", 1, func(c mqtt.Client, m mqtt.Message) {
			srv.handleTopicGPS(ctx, c, m)
		})
		token.Wait()
		if err := token.Error(); err != nil {
			logrus.WithError(errors.Wrap(err, "MQTT")).Error("Error")
		}

		token = srv.mqtt.Subscribe("ACCEL_OUT/#", 1, func(c mqtt.Client, m mqtt.Message) {
			srv.handleTopicACCEL(ctx, c, m)
		})
		token.Wait()
		if err := token.Error(); err != nil {
			logrus.WithError(errors.Wrap(err, "ACCEL")).Error("Error")
		}

		token = srv.mqtt.Subscribe("SUS_OUT/#", 1, func(c mqtt.Client, m mqtt.Message) {
			srv.handleTopicSUS(ctx, c, m)
		})
		token.Wait()
		if err := token.Error(); err != nil {
			logrus.WithError(errors.Wrap(err, "SUS")).Error("Error")
		}

		<-ctx.Done()
		srv.mqtt.Disconnect(250)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Panic")
			}
		}()

		mime.AddExtensionType(".js", "application/javascript")
		mime.AddExtensionType(".css", "text/css")
		router := httprouter.New()
		// router.Handler("GET", "/", fs) // , http.StripPrefix("/static", http.FileServer(http.Dir("./static/qa/"))
		router.Handler("GET", "/files/*filepath", http.StripPrefix("/files/", http.FileServer(http.Dir("public"))))
		router.HandlerFunc("GET", "/api/outdoors", srv.getOutdoors)
		router.GET("/api/car/:car/latest", srv.getLatestData)
		router.GET("/api/car/:car/power", srv.setMass)
		router.PUT("/api/mqtt/send/*topic", srv.sendToMqtt)
		router.GET("/api/mqtt/log", srv.getMqttLog)
		router.DELETE("/api/mqtt/log", srv.deleteMqttLogs)

		err := http.ListenAndServe(":1884", router)
		if err != nil {
			logrus.WithError(errors.Wrap(err, "HTTP")).Error("Error")
		}
	}()

	wg.Add(1)
	go func() { // reconnect mqtt client until it is connected
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Panic:%v\n", r)
				logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Panic")
			}
		}()

		// loop needs to happen no more than once per second
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(1 * time.Second)
				if srv.mqtt == nil || srv.mqtt.IsConnected() {
					break
				}
				token := srv.mqtt.Connect()
				token.Wait()
				if err := token.Error(); err != nil {
					logrus.WithError(errors.Wrap(err, "MQTT")).Error("Error")
				}
			}
		}
	}()

	wg.Wait()
}
