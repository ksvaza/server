package main

import (
	"fmt"
	"strconv"

	"github.com/ksvaza/server/master"
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
			fmt.Printf("Panic:%v\n", r)
		}
	}()

	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Error: '%s'\n", err.Error())
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

	print("Hello, world!\n")
}
