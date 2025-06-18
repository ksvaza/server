package master

import (
	"context"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Log struct {
	mu   sync.Mutex
	logs []string
}

func (l *Log) AddLog(topic string, payload []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	t := time.Now().Format("02.01.2006 15:04:05")
	log := fmt.Sprintf("%s - %s: %s", t, topic, string(payload))
	l.logs = append(l.logs, log)
	if len(l.logs) > 10000 {
		l.logs = l.logs[1:]
	}
}

func (l *Log) ClearLogs() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logs = []string{}
}

func (l *Log) GetLogs() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	var result string
	for i := len(l.logs) - 1; i >= 0; i-- {
		result += l.logs[i] + "\n"
	}

	return result
}

func (srv *Service) handleTopic(client mqtt.Client, msg mqtt.Message) {
	if len(msg.Topic()) >= 6 && msg.Topic()[:6] == "Aranet" {
		return
	}
	fmt.Printf("Received message: '%s' from topic: %s\n", msg.Payload(), msg.Topic())
	srv.log.AddLog(msg.Topic(), msg.Payload())
}

var MqttClientOptions *mqtt.ClientOptions

func createMqttClient(host string, port int, username, password string) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", host, port))
	//opts.AddBroker(fmt.Sprintf("ssl://%s:%s@%s:%d", username, password, host, port))
	opts.SetClientID("server")
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	opts.AutoReconnect = false
	opts.CleanSession = false
	//opts.SetTLSConfig(&tls.Config{InsecureSkipVerify: true})
	opts.OnConnect = func(c mqtt.Client) { fmt.Println("Mqtt client connected!") }
	opts.OnConnectionLost = func(c mqtt.Client, err error) { fmt.Printf("Connection lost: '%s'\n", err.Error()) }

	MqttClientOptions = opts
	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	return client, token.Error()
}

func recreateMqttClient() (mqtt.Client, error) {
	if MqttClientOptions == nil {
		return nil, errors.New("MqttClientOptions is nil")
	}
	client := mqtt.NewClient(MqttClientOptions)
	token := client.Connect()
	token.Wait()
	return client, token.Error()
}

func (srv *Service) SubscribeMQTT(ctx context.Context) {
	logrus.Debugf("Subscribing to MQTT topics")
	// Subscribe to all topics
	token := srv.mqtt.Subscribe("#", 1, srv.handleTopic)
	token.Wait()
	if err := token.Error(); err != nil {
		logrus.WithError(errors.Wrap(err, "MQTT")).Error("Error")
	}

	// token = srv.mqtt.Subscribe("Aranet/349681001757/sensors/10341A/json/measurements", 1, func(c mqtt.Client, m mqtt.Message) {
	// 	srv.handleOutdoorTemperature(ctx, c, m)
	// })
	// token.Wait()
	// if err := token.Error(); err != nil {
	// 	logrus.WithError(errors.Wrap(err, "MQTT")).Error("Error")
	// }

	// token = srv.mqtt.Subscribe("query", 1, func(c mqtt.Client, m mqtt.Message) {
	// 	srv.queryData(ctx, c, m)
	// })
	// token.Wait()
	// if err := token.Error(); err != nil {
	// 	logrus.WithError(errors.Wrap(err, "MQTT")).Error("Error")
	// }

	// Subscribe to car-server steam topics

	token = srv.mqtt.Subscribe("PSU_OUT/#", 1, func(c mqtt.Client, m mqtt.Message) {
		//srv.handleTopicPSU(ctx, c, m)
		srv.mqttReceivePSU(ctx, c, m)
	})
	token.Wait()
	if err := token.Error(); err != nil {
		logrus.WithError(errors.Wrap(err, "MQTT")).Error("Error")
	}

	token = srv.mqtt.Subscribe("GPS_OUT/#", 1, func(c mqtt.Client, m mqtt.Message) {
		//srv.handleTopicGPS(ctx, c, m)
		srv.mqttReceiveGPS(ctx, c, m)
	})
	token.Wait()
	if err := token.Error(); err != nil {
		logrus.WithError(errors.Wrap(err, "MQTT")).Error("Error")
	}

	token = srv.mqtt.Subscribe("Accel_OUT/#", 1, func(c mqtt.Client, m mqtt.Message) {
		//srv.handleTopicAccel(ctx, c, m)
		srv.mqttReceiveAccel(ctx, c, m)
	})
	token.Wait()
	if err := token.Error(); err != nil {
		logrus.WithError(errors.Wrap(err, "Accel")).Error("Error")
	}

	token = srv.mqtt.Subscribe("SUS_OUT/#", 1, func(c mqtt.Client, m mqtt.Message) {
		//srv.handleTopicSUS(ctx, c, m)
		srv.mqttReceiveSUS(ctx, c, m)
	})
	token.Wait()
	if err := token.Error(); err != nil {
		logrus.WithError(errors.Wrap(err, "SUS")).Error("Error")
	}
}
