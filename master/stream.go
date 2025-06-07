package master

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// --------------------------------------------------------------------------------------------------------------------------------
// --------------------------------------------------------------------------------------------------------------------------------

// Structures
// ----------------------------------------------------------------

// Json Payload structs
/*
PSU_IN: // Man vajag tikai spriegumu un srāvu jo Spriegums tad ir vnk info no komandas, strāva ir (m×const)/V
{
	"U":4976,
	"I":327
}

PSU_OUT:
{
	"Uop":3124,
	"Iop":327,
	"Pop":8699,
	"Uip":6129,
	"Wh":15356
}

GPS_OUT:
{
	"Lat":569427.00,
	"Lon":242244.53,
	"Spd": 2.00
}
*/
type payloadOutPSU struct {
	PSU struct {
		U  int `json:"U"`
		I  int `json:"I"`
		St int `json:"St"`
	} `json:"PSU"`
}
type payloadPSU struct {
	Uop int `json:"Uop"`
	Iop int `json:"Iop"`
	Pop int `json:"Pop"`
	Uip int `json:"Uip"`
	Wh  int `json:"Wh"`
}
type payloadGPS struct {
	Lat float32 `json:"Lat"`
	Lon float32 `json:"Lon"`
	Spd float32 `json:"Spd"`
}
type payloadAccel struct {
	X float32 `json:"X"`
	Y float32 `json:"Y"`
	Z float32 `json:"Z"`
}
type payloadAll struct {
	PSU   payloadPSU   `json:"PSU"`
	GPS   payloadGPS   `json:"GPS"`
	Accel payloadAccel `json:"Accel"`
}

// Data structs
type dataOutPSU struct {
	U      float32
	I      float32
	Status int
}
type dataPSU struct { // psu dati ir jāreizina/jādala ar 100, piem. "U":4976 (no/uz psu) - 49.76V
	Uop  float32
	Iop  float32
	Pop  float32
	Uip  float32
	Wh   float32
	Time time.Time
}
type dataGPS struct {
	Lat  float32
	Lon  float32
	Spd  float32
	Time time.Time
}
type dataAccel struct {
	X    float32
	Y    float32
	Z    float32
	Time time.Time
}
type dataSUS_SPD struct {
	Spd  float32
	Time time.Time
}
type dataSUS_RST struct {
	Rst  int
	Time time.Time
}

// --------------------------------------------------------------------------------------------------------------------------------
// --------------------------------------------------------------------------------------------------------------------------------

// Tool functions
// ----------------------------------------------------------------

// MQTT Data-in topic handlers
// ----------------------------------------------------------------

// Handle the PSU data
func (srv *Service) handleTopicPSU(ctx context.Context, client mqtt.Client, msg mqtt.Message) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Panic")
		}
	}()

	logrus.Debugf("New PSU value %s", msg.Topic())

	// Unmarshal the payload
	var payload payloadAll
	err := json.Unmarshal(msg.Payload(), &payload)
	if err != nil {
		//fmt.Printf("Error: '%s', payload '%s'\n", err.Error(), string(msg.Payload()))
		logrus.WithError(errors.Wrap(err, "PSU")).Error("Error")
		return
	}

	// Convert the payload to data
	data := dataPSU{
		Uop:  float32(payload.PSU.Uop) / 1000.0,
		Iop:  float32(payload.PSU.Iop) / 1000.0,
		Pop:  float32(payload.PSU.Pop) / 1000.0,
		Uip:  float32(payload.PSU.Uip) / 1000.0,
		Wh:   float32(payload.PSU.Wh) / 1000.0,
		Time: time.Now(),
	}

	// for debugging reasons print the data
	logrus.Debugf("Uop: %f, Iop: %f, Pop: %f, Uip: %f, Wh: %f", data.Uop, data.Iop, data.Pop, data.Uip, data.Wh)

	// Get car ID from subtopic
	var carID string
	carID, err = extractCarID(msg, err)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	org := "Kaste"
	bucket := "CarData"
	writeAPI := srv.Influxdb.WriteAPIBlocking(org, bucket)

	tags := map[string]string{}
	fields := map[string]interface{}{}

	tags["CarID"] = carID
	fields["Uop"] = data.Uop
	fields["Iop"] = data.Iop
	fields["Pop"] = data.Pop
	fields["Uip"] = data.Uip
	//fields["Wh"] = srv.dataCarStorage.MqttMessagePSU(carID, float64(data.Pop))
	fields["Wh"], err = srv.CarTable.MqttMessagePSU(carID, float64(data.Pop))
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	logrus.Debugf("Tags: %v, Fields: %v", tags, fields)

	point := write.NewPoint("PSU", tags, fields, data.Time)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}
}

// Handle the GPS data
func (srv *Service) handleTopicGPS(ctx context.Context, client mqtt.Client, msg mqtt.Message) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Panic")
		}
	}()

	logrus.Debugf("New GPS value %s", msg.Topic())

	// Unmarshal the payload
	var payload payloadAll
	err := json.Unmarshal(msg.Payload(), &payload)
	if err != nil {
		//fmt.Printf("Error: '%s', payload '%s'\n", err.Error(), string(msg.Payload()))
		logrus.WithError(errors.Wrap(err, "GPS")).Error("Error")
		return
	}

	// Convert the payload to data
	data := dataGPS{
		Lat:  payload.GPS.Lat,
		Lon:  payload.GPS.Lon,
		Spd:  payload.GPS.Spd,
		Time: time.Now(),
	}

	// for debugging reasons print the data
	logrus.Debugf("Lat: %f, Lon: %f, Spd: %f\n", data.Lat, data.Lon, data.Spd)

	// Get car ID from subtopic
	var carID string
	carID, err = extractCarID(msg, err)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	org := "Kaste"
	bucket := "CarData"
	writeAPI := srv.Influxdb.WriteAPIBlocking(org, bucket)

	tags := map[string]string{}
	fields := map[string]interface{}{}

	tags["CarID"] = carID
	fields["Lat"] = data.Lat
	fields["Lon"] = data.Lon
	fields["Spd"] = data.Spd

	err = srv.CarTable.MqttMessageAny(carID)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	point := write.NewPoint("GPS", tags, fields, data.Time)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}
}

// Handle the ACCEL data
func (srv *Service) handleTopicAccel(ctx context.Context, client mqtt.Client, msg mqtt.Message) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Panic")
		}
	}()

	logrus.Debugf("New Accel value %s", msg.Topic())

	// Unmarshal the payload
	var payload payloadAll
	err := json.Unmarshal(msg.Payload(), &payload)
	if err != nil {
		//fmt.Printf("Error: '%s', payload '%s'\n", err.Error(), string(msg.Payload()))
		logrus.WithError(errors.Wrap(err, "Accel")).Error("Error")
		return
	}

	// Convert the payload to data
	data := dataAccel{
		X:    payload.Accel.X,
		Y:    payload.Accel.Y,
		Z:    payload.Accel.Z,
		Time: time.Now(),
	}

	// for debugging reasons print the data
	logrus.Debugf("X: %f, Y: %f, Z: %f\n", data.X, data.Y, data.Z)

	// Get car ID from subtopic
	var carID string
	carID, err = extractCarID(msg, err)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	org := "Kaste"
	bucket := "CarData"
	writeAPI := srv.Influxdb.WriteAPIBlocking(org, bucket)

	tags := map[string]string{}
	fields := map[string]interface{}{}

	tags["CarID"] = carID
	fields["X"] = data.X
	fields["Y"] = data.Y
	fields["Z"] = data.Z

	err = srv.CarTable.MqttMessageAny(carID)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	point := write.NewPoint("Accel", tags, fields, data.Time)

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}
}

// Handle the SUS data
func (srv *Service) handleTopicSUS(ctx context.Context, client mqtt.Client, msg mqtt.Message) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithError(errors.New(fmt.Sprintf("%v", r))).Error("Panic")
		}
	}()

	// Read the speed or reset value
	logrus.Debugf("New SUS value %s", msg.Topic())
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

	// for debugging reasons print the data
	if speed != 0 {
		logrus.Debugf("Speed: %f\n", speed)
	} else {
		logrus.Debugf("Reset: %d\n", rst)
	}

	// Get car ID from subtopic
	var carID string
	carID, err = extractCarID(msg, err)
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	org := "Kaste"
	bucket := "CarData"
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
	if err != nil {
		logrus.WithError(err).Error("Error")
		return
	}

	point := write.NewPoint("SUS", tags, fields, time.Now())

	if err := writeAPI.WritePoint(ctx, point); err != nil {
		logrus.WithError(errors.Wrap(err, "InfluxDB")).Error("Error")
	}
}

// MQTT Data-out topic handlers
// ----------------------------------------------------------------

// Send the PSU data
func (srv *Service) sendPSUData(carID string, data dataOutPSU) error {
	payload := payloadOutPSU{}
	payload.PSU.U = int(math.Round(float64(data.U) * 100.0))
	payload.PSU.I = int(math.Round(float64(data.I) * 100.0))
	payload.PSU.St = data.Status
	bytes, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "JSON")
	}

	token := srv.mqtt.Publish(fmt.Sprintf("PSU_IN/%s", carID), 1, false, bytes)
	token.Wait()
	if err := token.Error(); err != nil {
		return errors.Wrap(err, "MQTT")
	}
	return nil
}

func (srv *Service) sendAnyTopic(topic string, payload []byte) error {
	token := srv.mqtt.Publish(topic, 1, false, payload)
	token.Wait()
	if err := token.Error(); err != nil {
		return errors.Wrap(err, "MQTT")
	}
	return nil
}
