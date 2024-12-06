package master

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func (srv *Service) handleTopic(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
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
