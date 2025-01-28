package main

import (
	"fmt"
	"strconv"

	"github.com/ksvaza/server/master"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func viperGetString(name string) string {
	value, ok := viper.Get(name).(string)
	if ok {
		return value
	} else {
		return ""
	}
}
func viperGetInt(name string) int {
	s, ok := viper.Get(name).(string)
	if ok {
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0
		} else {
			return n
		}
	} else {
		return 0
	}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Panic")
		}
	}()

	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "02.01.2006 15:04:05.000",
		FullTimestamp:   true,
		//		DisableColors:   true,
		DisableQuote: true,
	})
	logrus.SetOutput(&master.LogWriter{})

	logrus.AddHook(&master.StacktraceHook{})

	logrus.SetLevel(logrus.InfoLevel)

	logrus.Info("Car statistics server started")

	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		logrus.WithError(errors.Wrap(err, "Config read")).Error("Error")
	}

	config := master.Config{
		MqttHost:       viperGetString("MQTT_HOST"),
		MqttPort:       viperGetInt("MQTT_PORT"),
		MqttUsername:   viperGetString("MQTT_USER"),
		MqttPassword:   viperGetString("MQTT_PASSWORD"),
		InfluxdbUrl:    viperGetString("INFLUXDB_URL"),
		InfluxdbApikey: viperGetString("INFLUXDB_APIKEY"),
	}

	fmt.Println(config)

	server := master.NewService(config)
	server.Run()

	logrus.Info("Car statistics server stopped")
}
