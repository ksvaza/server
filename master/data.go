package master

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func extractCarID(msg mqtt.Message, err error) (string, error) {
	topics := strings.Split(msg.Topic(), "/")
	if len(topics) < 2 {
		return "", errors.Wrap(err, "Invalid topic")
	}

	_, err = strconv.ParseInt(topics[1], 10, 0)
	if err != nil {
		return "", errors.Wrap(err, "Subtopic is not a number")
	}
	return topics[1], nil
}

func (srv *Service) mqttReceivePSU(ctx context.Context, client mqtt.Client, msg mqtt.Message) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Panic")
		}
	}()

	logrus.Debugf("New PSU value %s", msg.Topic())

	var payload payloadAll
	err := json.Unmarshal(msg.Payload(), &payload)
	if err != nil {
		logrus.WithError(errors.Wrap(err, "PSU")).Error("Error")
		return
	}

	data := dataPSU{
		Uop:  float32(payload.PSU.Uop) / 1000.0,
		Iop:  float32(payload.PSU.Iop) / 1000.0,
		Pop:  float32(payload.PSU.Pop) / 1000.0,
		Uip:  float32(payload.PSU.Uip) / 1000.0,
		Wh:   float32(payload.PSU.Wh) / 1000.0,
		Time: time.Now(),
	}

	logrus.Debugf("Uop: %f, Iop: %f, Pop: %f, Uip: %f, Wh: %f", data.Uop, data.Iop, data.Pop, data.Uip, data.Wh)

	var carID string
	carID, err = extractCarID(msg, err)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	org := "Kaste"
	bucket := "AllData"
	writeAPI := srv.Influxdb.WriteAPIBlocking(org, bucket)

	consumption, err := srv.CarTable.MqttMessagePSU(carID, float64(data.Pop))
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	tags := map[string]string{}
	fields := map[string]interface{}{}

	tags["CarID"] = carID
	fields["Uop"] = data.Uop
	fields["Iop"] = data.Iop
	fields["Pop"] = data.Pop
	fields["Uip"] = data.Uip
	fields["Wh"] = consumption
	if CurrentRaceConfig != nil {
		tags["Race"] = CurrentRaceConfig.Name
	} else {
		tags["Race"] = "nil"
	}

	logrus.Debugf("Tags: %v, Fields: %v", tags, fields)

	point := write.NewPoint("PSU", tags, fields, data.Time)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}

	if CurrentRaceConfig == nil {
		logrus.Error("CurrentRace is nil")
		return
	}
	if _, exists := srv.CarTable[carID]; !exists {
		logrus.Errorf("Car %s not registered", carID)
		return
	}

	bucket = "RaceData/" + CurrentRaceConfig.Name
	writeAPI = srv.Influxdb.WriteAPIBlocking(org, bucket)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}
}

func (srv *Service) mqttReceiveGPS(ctx context.Context, client mqtt.Client, msg mqtt.Message) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Panic")
		}
	}()

	logrus.Debugf("New GPS value %s", msg.Topic())

	var payload payloadAll
	err := json.Unmarshal(msg.Payload(), &payload)
	if err != nil {
		logrus.WithError(errors.Wrap(err, "GPS")).Error("Error")
		return
	}

	data := dataGPS{
		Lat:  payload.GPS.Lat,
		Lon:  payload.GPS.Lon,
		Spd:  payload.GPS.Spd,
		Time: time.Now(),
	}

	logrus.Debugf("Lat: %f, Lon: %f, Spd: %f", data.Lat, data.Lon, data.Spd)

	var carID string
	carID, err = extractCarID(msg, err)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	org := "Kaste"
	bucket := "AllData"
	writeAPI := srv.Influxdb.WriteAPIBlocking(org, bucket)

	err = srv.CarTable.MqttMessageAny(carID)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	tags := map[string]string{}
	fields := map[string]interface{}{}

	tags["CarID"] = carID
	fields["Lat"] = data.Lat
	fields["Lon"] = data.Lon
	fields["Spd"] = data.Spd
	if CurrentRaceConfig != nil {
		tags["Race"] = CurrentRaceConfig.Name
	} else {
		tags["Race"] = "nil"
	}

	logrus.Debugf("Tags: %v, Fields: %v", tags, fields)

	point := write.NewPoint("GPS", tags, fields, data.Time)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}

	if CurrentRaceConfig == nil {
		logrus.Error("CurrentRace is nil")
		return
	}
	if _, exists := srv.CarTable[carID]; !exists {
		logrus.Errorf("Car %s not registered", carID)
		return
	}

	bucket = "RaceData/" + CurrentRaceConfig.Name
	writeAPI = srv.Influxdb.WriteAPIBlocking(org, bucket)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}
}

func (srv *Service) mqttReceiveAccel(ctx context.Context, client mqtt.Client, msg mqtt.Message) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Panic")
		}
	}()

	logrus.Debugf("New Accel value %s", msg.Topic())

	var payload payloadAll
	err := json.Unmarshal(msg.Payload(), &payload)
	if err != nil {
		logrus.WithError(errors.Wrap(err, "Accel")).Error("Error")
		return
	}

	data := dataAccel{
		X:    payload.Accel.X,
		Y:    payload.Accel.Y,
		Z:    payload.Accel.Z,
		Time: time.Now(),
	}

	logrus.Debugf("X: %f, Y: %f, Z: %f", data.X, data.Y, data.Z)

	var carID string
	carID, err = extractCarID(msg, err)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	org := "Kaste"
	bucket := "AllData"
	writeAPI := srv.Influxdb.WriteAPIBlocking(org, bucket)

	err = srv.CarTable.MqttMessageAny(carID)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	tags := map[string]string{}
	fields := map[string]interface{}{}

	tags["CarID"] = carID
	fields["X"] = data.X
	fields["Y"] = data.Y
	fields["Z"] = data.Z
	if CurrentRaceConfig != nil {
		tags["Race"] = CurrentRaceConfig.Name
	} else {
		tags["Race"] = "nil"
	}

	logrus.Debugf("Tags: %v, Fields: %v", tags, fields)

	point := write.NewPoint("Accel", tags, fields, data.Time)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}

	if CurrentRaceConfig == nil {
		logrus.Error("CurrentRace is nil")
		return
	}
	if _, exists := srv.CarTable[carID]; !exists {
		logrus.Errorf("Car %s not registered", carID)
		return
	}

	bucket = "RaceData/" + CurrentRaceConfig.Name
	writeAPI = srv.Influxdb.WriteAPIBlocking(org, bucket)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}
}

func (srv *Service) mqttReceiveSUS(ctx context.Context, client mqtt.Client, msg mqtt.Message) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Panic")
		}
	}()

	logrus.Debugf("New SUS value %s", msg.Topic())

	// Read the speed or reset value
	var speed float32
	var rst int
	_, err := fmt.Sscanf(string(msg.Payload()), "SPD: %f", &speed)
	if err != nil {
		logrus.Debugf("No speed argument")
		_, err = fmt.Sscanf(string(msg.Payload()), "RST: %d", &rst)
		if err != nil {
			logrus.WithError(errors.Wrap(err, "SUS")).Error("Error")
			return
		}
	}

	// Get car ID from subtopic
	var carID string
	carID, err = extractCarID(msg, err)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	org := "Kaste"
	bucket := "AllData"
	writeAPI := srv.Influxdb.WriteAPIBlocking(org, bucket)

	tags := map[string]string{}
	fields := map[string]interface{}{}

	tags["CarID"] = carID
	if speed != 0 {
		fields["Spd"] = speed
		err = srv.CarTable.MqttMessageAny(carID)
	} else {
		fields["Rst"] = rst
		err = srv.CarTable.MqttMessageRST(carID, "", srv)
	}
	if CurrentRaceConfig != nil {
		tags["Race"] = CurrentRaceConfig.Name
	} else {
		tags["Race"] = "nil"
	}
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	point := write.NewPoint("SUS", tags, fields, time.Now())

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}

	if CurrentRaceConfig == nil {
		logrus.Error("CurrentRace is nil")
		return
	}
	if _, exists := srv.CarTable[carID]; !exists {
		logrus.Errorf("Car %s not registered", carID)
		return
	}

	bucket = "RaceData/" + CurrentRaceConfig.Name
	writeAPI = srv.Influxdb.WriteAPIBlocking(org, bucket)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}
}

func (table *CarIDMap) SaveToFile() error {
	if *table == nil {
		*table = make(CarIDMap)
	}
	data, err := json.Marshal(*table)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal CarIDMap")
	}
	err = os.WriteFile("caridmap.json", data, 0644)
	if err != nil {
		return errors.Wrap(err, "Failed to write CarIDMap to file")
	}
	return nil
}

func (table *CarIDMap) LoadFromFile() error {
	if *table == nil {
		*table = make(CarIDMap)
	}
	data, err := os.ReadFile("caridmap.json")
	if err != nil {
		return errors.Wrap(err, "Failed to read CarIDMap from file")
	}
	err = json.Unmarshal(data, table)
	if err != nil {
		return errors.Wrap(err, "Failed to unmarshal CarIDMap")
	}
	return nil
}

func (table *RaceNameMap) SaveToFile() error {
	if *table == nil {
		*table = make(RaceNameMap)
	}
	data, err := json.Marshal(*table)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal RaceNameMap")
	}
	err = os.WriteFile("racenamemap.json", data, 0644)
	if err != nil {
		return errors.Wrap(err, "Failed to write RaceNameMap to file")
	}
	return nil
}

func (table *RaceNameMap) LoadFromFile() error {
	if *table == nil {
		*table = make(RaceNameMap)
	}
	data, err := os.ReadFile("racenamemap.json")
	if err != nil {
		return errors.Wrap(err, "Failed to read RaceNameMap from file")
	}
	err = json.Unmarshal(data, table)
	if err != nil {
		return errors.Wrap(err, "Failed to unmarshal RaceNameMap")
	}
	return nil
}
