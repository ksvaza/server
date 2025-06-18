package master

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func EnsureBucket(client influxdb2.Client, org, bucket string) (string, error) {
	bucketsAPI := client.BucketsAPI()
	organizationsAPI := client.OrganizationsAPI()
	ctx := context.Background()

	// Check if bucket exists
	b, err := bucketsAPI.FindBucketByName(ctx, bucket)
	if err == nil && b != nil {
		return bucket, nil
	}

	// Find organization
	organization, err := organizationsAPI.FindOrganizationByName(ctx, org)
	if err != nil {
		return "", err
	}

	// Create bucket
	_, err = bucketsAPI.CreateBucketWithName(ctx, organization, bucket)
	if err != nil {
		return "", err
	}
	return bucket, nil
}

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
		Uop:  float32(payload.PSU.Uop) / 100.0,
		Iop:  float32(payload.PSU.Iop) / 100.0,
		Pop:  float32(payload.PSU.Pop) / 100.0,
		Uip:  float32(payload.PSU.Uip) / 100.0,
		Wh:   float32(payload.PSU.Wh),
		Time: time.Now(),
	}

	logrus.Debugf("Uop: %f, Iop: %f, Pop: %f, Uip: %f, Wh: %f", data.Uop, data.Iop, data.Pop, data.Uip, data.Wh)

	var carID string
	carID, err = extractCarID(msg, err)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	srv.AllData.UpdateLiveDataCarPSU(carID, float64(data.Pop), float64(data.Uop))

	org := "Kaste"
	bucket, err := EnsureBucket(srv.Influxdb, org, "AllData/"+srv.AllData.UUID.String())
	if err != nil {
		logrus.WithError(errors.Wrap(err, "EnsureBucket")).Error("Error")
		return
	}
	writeAPI := srv.Influxdb.WriteAPIBlocking(org, bucket)

	consumption, err := srv.AllData.MqttMessagePSU(carID, float64(data.Pop))
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	tags := map[string]string{}
	fields := map[string]interface{}{}
	var race *Race
	var registered bool

	tags["CarID"] = carID
	fields["Uop"] = data.Uop
	fields["Iop"] = data.Iop
	fields["Pop"] = data.Pop
	fields["Uip"] = data.Uip
	fields["Wh"] = consumption
	var car Car
	if car, registered := srv.AllData.CarMap[carID]; registered {
		race = car.CurrentRace
		if race != nil {
			fields["Race"] = race.RaceName
			fields["Lap"] = race.Lap
		} else {
			fields["Race"] = "nil"
			fields["Lap"] = 0
		}
	}

	payloadO := dataOutPSU{
		U:      float32(car.Params.SetVoltage),
		I:      float32(car.Params.MaxCurrent),
		Status: 1,
	}
	srv.sendPSUData(carID, payloadO)

	logrus.Debugf("Tags: %v, Fields: %v", tags, fields)

	point := write.NewPoint("PSU", tags, fields, data.Time)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}

	if race == nil {
		logrus.Error("CurrentRace is nil")
		return
	}
	if !registered {
		logrus.Errorf("Car %s not registered", carID)
		return
	}

	bucket, err = EnsureBucket(srv.Influxdb, org, "RaceData/"+srv.AllData.UUID.String()+"/"+race.RaceName+"/"+strconv.Itoa(race.Lap))
	if err != nil {
		logrus.WithError(errors.Wrap(err, "EnsureBucket")).Error("Error")
		return
	}
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

	srv.AllData.UpdateLiveDataCarGPS(carID, float64(data.Lat), float64(data.Lon), float64(data.Spd))
	srv.AllData.CheckSpeed(carID, data.Spd, srv)

	org := "Kaste"
	bucket, err := EnsureBucket(srv.Influxdb, org, "AllData/"+srv.AllData.UUID.String())
	if err != nil {
		logrus.WithError(errors.Wrap(err, "EnsureBucket")).Error("Error")
		return
	}
	writeAPI := srv.Influxdb.WriteAPIBlocking(org, bucket)

	err = srv.AllData.MqttMessageAny(carID)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	tags := map[string]string{}
	fields := map[string]interface{}{}
	var race *Race
	var registered bool

	tags["CarID"] = carID
	fields["Lat"] = data.Lat
	fields["Lon"] = data.Lon
	fields["Spd"] = data.Spd
	if car, registered := srv.AllData.CarMap[carID]; registered {
		race = car.CurrentRace
		if race != nil {
			fields["Race"] = race.RaceName
			fields["Lap"] = race.Lap
		} else {
			fields["Race"] = "nil"
			fields["Lap"] = 0
		}
	}

	logrus.Debugf("Tags: %v, Fields: %v", tags, fields)

	point := write.NewPoint("GPS", tags, fields, data.Time)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}

	if race == nil {
		logrus.Error("CurrentRace is nil")
		return
	}
	if !registered {
		logrus.Errorf("Car %s not registered", carID)
		return
	}

	bucket, err = EnsureBucket(srv.Influxdb, org, "RaceData/"+srv.AllData.UUID.String()+"/"+race.RaceName+"/"+strconv.Itoa(race.Lap))
	if err != nil {
		logrus.WithError(errors.Wrap(err, "EnsureBucket")).Error("Error")
		return
	}
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

	accel := math.Sqrt(float64(data.X*data.X + data.Y*data.Y + data.Z*data.Z))

	srv.AllData.UpdateLiveDataCarAccel(carID, accel)

	org := "Kaste"
	bucket, err := EnsureBucket(srv.Influxdb, org, "AllData/"+srv.AllData.UUID.String())
	if err != nil {
		logrus.WithError(errors.Wrap(err, "EnsureBucket")).Error("Error")
		return
	}
	writeAPI := srv.Influxdb.WriteAPIBlocking(org, bucket)

	err = srv.AllData.MqttMessageAny(carID)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	tags := map[string]string{}
	fields := map[string]interface{}{}
	var race *Race
	var registered bool

	tags["CarID"] = carID
	fields["X"] = data.X
	fields["Y"] = data.Y
	fields["Z"] = data.Z
	if car, registered := srv.AllData.CarMap[carID]; registered {
		race = car.CurrentRace
		if race != nil {
			fields["Race"] = race.RaceName
			fields["Lap"] = race.Lap
		} else {
			fields["Race"] = "nil"
			fields["Lap"] = 0
		}
	}

	logrus.Debugf("Tags: %v, Fields: %v", tags, fields)

	point := write.NewPoint("Accel", tags, fields, data.Time)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}

	if race == nil {
		logrus.Error("CurrentRace is nil")
		return
	}
	if !registered {
		logrus.Errorf("Car %s not registered", carID)
		return
	}

	bucket, err = EnsureBucket(srv.Influxdb, org, "RaceData/"+srv.AllData.UUID.String()+"/"+race.RaceName+"/"+strconv.Itoa(race.Lap))
	if err != nil {
		logrus.WithError(errors.Wrap(err, "EnsureBucket")).Error("Error")
		return
	}
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
	bucket, err := EnsureBucket(srv.Influxdb, org, "AllData/"+srv.AllData.UUID.String())
	if err != nil {
		logrus.WithError(errors.Wrap(err, "EnsureBucket")).Error("Error")
		return
	}
	writeAPI := srv.Influxdb.WriteAPIBlocking(org, bucket)

	tags := map[string]string{}
	fields := map[string]interface{}{}
	var race *Race
	var registered bool

	tags["CarID"] = carID
	if speed != 0 {
		fields["Spd"] = speed
		err = srv.AllData.MqttMessageAny(carID)
		if err != nil {
			logrus.WithError(err).Error("Error")
			return
		}
	} else {
		fields["Rst"] = rst
		err = srv.AllData.MqttMessageRST(carID, "", srv)
		if err != nil {
			logrus.WithError(err).Error("Error")
			return
		}
	}
	if car, registered := srv.AllData.CarMap[carID]; registered {
		race = car.CurrentRace
		if race != nil {
			fields["Race"] = race.RaceName
			fields["Lap"] = race.Lap
		} else {
			fields["Race"] = "nil"
			fields["Lap"] = 0
		}
	}

	point := write.NewPoint("SUS", tags, fields, time.Now())

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}

	if race == nil {
		logrus.Error("CurrentRace is nil")
		return
	}
	if !registered {
		logrus.Errorf("Car %s not registered", carID)
		return
	}

	bucket, err = EnsureBucket(srv.Influxdb, org, "RaceData/"+srv.AllData.UUID.String()+"/"+race.RaceName+"/"+strconv.Itoa(race.Lap))
	if err != nil {
		logrus.WithError(errors.Wrap(err, "EnsureBucket")).Error("Error")
		return
	}
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
