package master

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

type mqttMeasurement struct {
	Battery     string `json:"battery"`
	Time        string `json:"time"`
	Temperature string `json:"temperature"`
	Humidity    string `json:"humidity"`
	Rssi        string `json:"rssi"`
}

type measurement struct {
	Time        time.Time
	Temperature float64
	Humidity    float64
	Battery     float64
	Rssi        float64
}

func (srv *Service) queryData(ctx context.Context, client mqtt.Client, msg mqtt.Message) {
	queryAPI := srv.Influxdb.QueryAPI("Aranet")
	query := `from(bucket: "Outdoors")
            |> range(start: -1m)
            |> filter(fn: (r) => r._measurement == "measurement")`
	results, err := queryAPI.Query(ctx, query)
	if err != nil {
		fmt.Printf("Error: '%s'\n", err.Error())
	}
	data := map[time.Time]*measurement{}
	for results.Next() {
		raw := results.Record()
		t := raw.Time()
		if _, found := data[t]; !found {
			data[t] = &measurement{Time: t}
		}
		switch raw.Field() {
		case "battery":
			if f, ok := raw.Value().(float64); ok {
				data[t].Battery = f
			}
		case "temperature":
			if f, ok := raw.Value().(float64); ok {
				data[t].Temperature = f
			}
		case "humidity":
			if f, ok := raw.Value().(float64); ok {
				data[t].Humidity = f
			}
		case "rssi":
			if f, ok := raw.Value().(float64); ok {
				data[t].Rssi = f
			}
		}
	}
	if err := results.Err(); err != nil {
		fmt.Printf("Error: '%s'\n", err.Error())
	}

	result := []measurement{}
	for _, value := range data {
		result = append(result, *value)
	}

	slices.SortFunc(result, func(a, b measurement) int {
		return a.Time.Compare(b.Time)
	})

	fmt.Printf("Data: ")
	fmt.Println(result)
}

func (srv *Service) queryMeasurements(ctx context.Context) ([]measurement, error) {
	queryAPI := srv.Influxdb.QueryAPI("Aranet")
	query := `from(bucket: "Outdoors")
            |> range(start: -10m)
            |> filter(fn: (r) => r._measurement == "measurement")`
	results, err := queryAPI.Query(ctx, query)
	if err != nil {
		//fmt.Printf("Error: '%s'\n", err.Error())
		return nil, err
	}
	data := map[time.Time]*measurement{}
	for results.Next() {
		raw := results.Record()
		t := raw.Time()
		if _, found := data[t]; !found {
			data[t] = &measurement{Time: t}
		}
		switch raw.Field() {
		case "battery":
			if f, ok := raw.Value().(float64); ok {
				data[t].Battery = f
			}
		case "temperature":
			if f, ok := raw.Value().(float64); ok {
				data[t].Temperature = f
			}
		case "humidity":
			if f, ok := raw.Value().(float64); ok {
				data[t].Humidity = f
			}
		case "rssi":
			if f, ok := raw.Value().(float64); ok {
				data[t].Rssi = f
			}
		}
	}
	if err := results.Err(); err != nil {
		//fmt.Printf("Error: '%s'\n", err.Error())
		return nil, err
	}

	result := []measurement{}
	for _, value := range data {
		result = append(result, *value)
	}

	slices.SortFunc(result, func(a, b measurement) int {
		return a.Time.Compare(b.Time)
	})

	return result, nil
}

func (srv *Service) handleOutdoorTemperature(ctx context.Context, client mqtt.Client, msg mqtt.Message) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic:%v\n", r)
		}
	}()

	fmt.Println("New value")

	var data mqttMeasurement
	err := json.Unmarshal(msg.Payload(), &data)
	if err != nil {
		fmt.Printf("Error: '%s', payload '%s'\n", err.Error(), string(msg.Payload()))
		return
	}

	org := "Aranet"
	bucket := "Outdoors"
	writeAPI := srv.Influxdb.WriteAPIBlocking(org, bucket)

	tags := map[string]string{}

	fields := map[string]interface{}{}

	if s, err := strconv.ParseFloat(data.Battery, 32); err == nil {
		fields["battery"] = s
	}
	if s, err := strconv.ParseFloat(data.Temperature, 32); err == nil {
		fields["temperature"] = s
	}
	if s, err := strconv.ParseFloat(data.Humidity, 32); err == nil {
		fields["humidity"] = s
	}
	if s, err := strconv.ParseFloat(data.Rssi, 32); err == nil {
		fields["rssi"] = s
	}

	var t time.Time
	if s, err := strconv.ParseInt(data.Battery, 10, 64); err == nil {
		t = time.Unix(s, 0)
	} else {
		t = time.Now()
	}

	point := write.NewPoint("measurement", tags, fields, t)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		fmt.Printf("Error: '%s'\n", err.Error())
	}
}
