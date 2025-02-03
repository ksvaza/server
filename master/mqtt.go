package master

import (
	"fmt"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Log struct {
	mu   sync.Mutex
	logs []string
}

func (l *Log) AddLog(topic string, payload []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	log := fmt.Sprintf("%s: %s", topic, string(payload))
	l.logs = append(l.logs, log)
	if len(l.logs) > 1000 {
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
	fmt.Printf("Received message: '%s' from topic: %s\n", msg.Payload(), msg.Topic())
	srv.log.AddLog(msg.Topic(), msg.Payload())
}

func createMqttClient(host string, port int, username, password string) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", host, port))
	opts.SetClientID("server")
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.OnConnect = func(c mqtt.Client) { fmt.Println("Mqtt client connected!") }
	opts.OnConnectionLost = func(c mqtt.Client, err error) { fmt.Printf("Connection lost: '%s'\n", err.Error()) }

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	return client, token.Error()
}
