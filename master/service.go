package master

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Service struct {
	StopSignal chan os.Signal
	host       string
	port       int
	username   string
	password   string
}

func NewService(host string, port int, username, password string) *Service {
	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	return &Service{
		StopSignal: stopSignal,
		host:       host,
		port:       port,
		username:   username,
		password:   password,
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

		topic := "topic"
		token := client.Subscribe(topic, 1, srv.handleTopic)
		token.Wait()
		fmt.Printf("Subscribed to topic %s\n", topic)

		<-ctx.Done()
		client.Disconnect(250)
	}()

	wg.Wait()
}
