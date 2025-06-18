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
	AllData    AllData
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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

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
	srv := &Service{
		StopSignal: stopSignal,
		host:       config.MqttHost,
		port:       config.MqttPort,
		username:   config.MqttUsername,
		password:   config.MqttPassword,
		Influxdb:   influxdb2.NewClient(config.InfluxdbUrl, config.InfluxdbApikey),
	}

	srv.AllData.LiveData = map[string]LiveDataInstance{}

	return srv
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

		// err := srv.CarTable.LoadFromFile()
		// if err != nil {
		// 	logrus.WithError(errors.Wrap(err, "CarTable")).Error("Error loading car data")
		// }
		// err = srv.RaceTable.LoadFromFile()
		// if err != nil {
		// 	logrus.WithError(errors.Wrap(err, "RaceTable")).Error("Error loading race data")
		// }
		err := srv.AllData.LoadFromFile()
		if err != nil {
			logrus.WithError(errors.Wrap(err, "AllData")).Error("Error loading all data")
		}

		mime.AddExtensionType(".js", "application/javascript")
		mime.AddExtensionType(".css", "text/css")
		router := httprouter.New()
		// router.Handler("GET", "/", fs) // , http.StripPrefix("/static", http.FileServer(http.Dir("./static/qa/"))
		//router.Handler("GET", "/files/*filepath", http.StripPrefix("/files/", http.FileServer(http.Dir("public"))))
		//router.HandlerFunc("GET", "/api/outdoors", srv.getOutdoors)
		router.HandlerFunc("GET", "/ws", srv.wsHandler)

		router.GET("/api/cars", withCORS(srv.getCars))
		router.POST("/api/cars", withCORS(srv.postCars))
		router.GET("/api/races", withCORS(srv.getRaces))
		router.POST("/api/races", withCORS(srv.postRaces))
		router.GET("/api/results/:racename", withCORS(srv.getResults))
		router.GET("/api/leaderboard/:agegroup", withCORS(srv.getLeaderboard))
		router.DELETE("/api/leaderboard/:agegroup", withCORS(srv.deleteLeaderboard))
		router.POST("/api/race/start", withCORS(srv.postStartRace))
		router.POST("/api/race/finish", withCORS(srv.postRaceFinish))
		router.POST("/api/car/finish", withCORS(srv.postCarFinish))
		router.POST("/api/points", withCORS(srv.postPoints))
		router.DELETE("/api/points", withCORS(srv.deletePoints))
		router.DELETE("/api/delete", withCORS(srv.deleteData))
		router.GET("/api/settings", withCORS(srv.getSettings))
		router.POST("/api/settings", withCORS(srv.postSettings))

		// ------------------------
		router.GET("/api/car/:car/latest", withCORS(srv.getLatestData))
		//router.GET("/api/car/:car/power", withCORS(srv.setMass)) // deprecated
		router.PUT("/api/mqtt/send/:topic", withCORS(srv.sendToMqtt))
		router.GET("/api/mqtt/log", withCORS(srv.getMqttLog))
		router.DELETE("/api/mqtt/log", withCORS(srv.deleteMqttLogs))
		router.GET("/api/race/:car/start", withCORS(srv.triggerRaceStart))   // should be POST
		router.GET("/api/race/:car/finish", withCORS(srv.triggerRaceFinish)) // should be POST

		// //router.Handler("GET", "/*filepath", http.FileServer(http.Dir("public")))
		// http.Handle("/", http.FileServer(http.Dir("public")))

		// err = http.ListenAndServe(":1884", router)
		// if err != nil {
		// 	logrus.WithError(errors.Wrap(err, "HTTP")).Error("Error")
		// }

		fileServer := http.FileServer(http.Dir("public"))

		// Custom handler to delegate /api and /ws to httprouter, everything else to fileServer
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/ws" || len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
				router.ServeHTTP(w, r)
			} else {
				fileServer.ServeHTTP(w, r)
			}
		})

		err = http.ListenAndServe(":1884", handler)
		if err != nil {
			logrus.WithError(errors.Wrap(err, "HTTP")).Error("Error")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Panic")
			}
		}()

		for {
			for carID, car := range srv.AllData.CarMap {
				// Calculate max current
				if car.Params.SetVoltage <= 0 {
					logrus.Warnf("SetVoltage for car %s is zero or negative, skipping PSU update", carID)
					continue
				}
				maxCurrent := car.Params.Mass * srv.AllData.Settings.RaceCoeficient / car.Params.SetVoltage
				car.Params.MaxCurrent = maxCurrent

				payload := dataOutPSU{
					U:      float32(car.Params.SetVoltage),
					I:      float32(car.Params.MaxCurrent),
					Status: 1,
				}

				srv.sendPSUData(carID, payload)
			}
			time.Sleep(1 * time.Second) // send PSU data every 5 seconds
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			srv.SendLiveData()
			time.Sleep(1 * time.Second) // send live data every second
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
