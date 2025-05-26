package master

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
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
	CarTable   CarIDMap
	RaceTable  RaceNameMap
}

type Config struct {
	MqttHost       string
	MqttPort       int
	MqttUsername   string
	MqttPassword   string
	InfluxdbUrl    string
	InfluxdbApikey string
}

var lastMessageTime atomic.Int64

func withCORS(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // or "http://localhost:5173"
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		lastMessageTime.Store(time.Now().UnixMilli())

		h(w, r, ps)
	}
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
		srv.mqtt = nil
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

		srv.SubscribeMQTT(ctx)

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
		// ---------------------------------------------------------------------------------  ieteicamie jaunie nosaukumi implementētajiem api galiem
		router.GET("/api/parameters", withCORS(srv.getParameters))          // cars
		router.POST("/api/parameters", withCORS(srv.postParameters))        // cars
		router.GET("/api/raceconfiguration", withCORS(srv.getRaceConfig))   // races
		router.POST("/api/raceconfiguration", withCORS(srv.postRaceConfig)) // races
		router.POST("/api/start-race", withCORS(srv.startRace))             // race/start
		router.POST("/api/end-race", withCORS(srv.endRace))                 // race/end
		router.POST("/api/whofinished", withCORS(srv.whoFinished))          // car/finish

		// ------------------------
		router.GET("/api/car/:car/latest", withCORS(srv.getLatestData))
		//router.GET("/api/car/:car/power", withCORS(srv.setMass)) // deprecated
		router.PUT("/api/mqtt/send/*topic", withCORS(srv.sendToMqtt))
		router.GET("/api/mqtt/log", withCORS(srv.getMqttLog))
		router.DELETE("/api/mqtt/log", withCORS(srv.deleteMqttLogs))
		router.GET("/api/race/:car/start", withCORS(srv.triggerRaceStart))   // should be POST
		router.GET("/api/race/:car/finish", withCORS(srv.triggerRaceFinish)) // should be POST

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
				if srv.mqtt != nil && srv.mqtt.IsConnected() {
					continue
				}

				client, err := recreateMqttClient()
				if err != nil {
					logrus.WithError(errors.Wrap(err, "MQTT")).Error("Error")
					continue
				}
				srv.mqtt = client
				logrus.Debugf("Reconnecting to MQTT broker %s:%d", srv.host, srv.port)

				srv.SubscribeMQTT(ctx)
			}
		}
	}()

	wg.Add(1)
	go func() {
		for {
			time.Sleep(5 * time.Second)
			last := lastMessageTime.Load()
			if time.Since(time.UnixMilli(last)) > 10*time.Second {
				logrus.Warn("No MQTT data received for 10s — forcing disconnect")
				if srv.mqtt != nil && srv.mqtt.IsConnected() {
					srv.mqtt.Disconnect(250) // trigger reconnect loop
				}
			}
		}
	}()

	wg.Wait()
}
