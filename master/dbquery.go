package master

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type dataCarFull struct {
	PSU     *dataPSU     `json:"PSU,omitempty"`
	GPS     *dataGPS     `json:"GPS,omitempty"`
	ACCEL   *dataACCEL   `json:"ACCEL,omitempty"`
	SUS_SPD *dataSUS_SPD `json:"SUS_SPD,omitempty"`
	SUS_RST *dataSUS_RST `json:"SUS_RST,omitempty"`
}

type carResponse struct {
	Data dataCarFull
}

func (srv *Service) queryLatestPSU(ctx context.Context, carID string) (*dataPSU, error) {
	queryAPI := srv.Influxdb.QueryAPI("Kaste")

	// latest data from PSU
	query := `
from(bucket: "CarData")
  |> range(start: 1882-11-18)
  |> last()
  |> filter(fn: (r) => r["_measurement"] == "PSU")
  |> filter(fn: (r) => r["CarID"] == "%s")` // latest data from PSU
	results, err := queryAPI.Query(ctx, fmt.Sprintf(query, carID))
	if err != nil {
		return nil, errors.Wrap(err, "InfluxDB,PSU")
	}

	/*
		tags["CarID"] = carID
		fields["Uop"] = data.Uop
		fields["Iop"] = data.Iop
		fields["Pop"] = data.Pop
		fields["Uip"] = data.Uip
		fields["Wh"] = data.Wh
	*/
	psu := map[time.Time]*dataPSU{}
	for results.Next() {
		raw := results.Record()
		logrus.Debugf("raw: %+v '%T'", raw, raw.Value())
		t := raw.Time()
		if _, found := psu[t]; !found {
			psu[t] = &dataPSU{Time: t}
		}
		switch raw.Field() {
		case "Uop":
			if f, ok := raw.Value().(float64); ok {
				psu[t].Uop = float32(f)
			}
		case "Iop":
			if f, ok := raw.Value().(float64); ok {
				psu[t].Iop = float32(f)
			}
		case "Pop":
			if f, ok := raw.Value().(float64); ok {
				psu[t].Pop = float32(f)
			}
		case "Uip":
			if f, ok := raw.Value().(float64); ok {
				psu[t].Uip = float32(f)
			}
		case "Wh":
			if f, ok := raw.Value().(float64); ok {
				psu[t].Wh = float32(f)
			}
		}
	}
	if err := results.Err(); err != nil {
		return nil, errors.Wrap(err, "InfluxDB,PSU")
	}

	for _, value := range psu {
		return value, nil
	}

	return nil, nil
}

func (srv *Service) queryLatestGPS(ctx context.Context, carID string) (*dataGPS, error) {
	queryAPI := srv.Influxdb.QueryAPI("Kaste")

	// latest data from GPS
	query := `
from(bucket: "CarData")
  |> range(start: 1882-11-18)
  |> last()
  |> filter(fn: (r) => r["_measurement"] == "GPS")
  |> filter(fn: (r) => r["CarID"] == "%s")` // latest data from GPS
	results, err := queryAPI.Query(ctx, fmt.Sprintf(query, carID))
	if err != nil {
		return nil, errors.Wrap(err, "InfluxDB,GPS")
	}
	/*
		tags["CarID"] = carID
		fields["Lat"] = data.Lat
		fields["Lon"] = data.Lon
		fields["Spd"] = data.Spd
	*/
	gps := map[time.Time]*dataGPS{}
	for results.Next() {
		raw := results.Record()
		t := raw.Time()
		if _, found := gps[t]; !found {
			gps[t] = &dataGPS{Time: t}
		}
		switch raw.Field() {
		case "Lat":
			if f, ok := raw.Value().(float64); ok {
				gps[t].Lat = float32(f)
			}
		case "Lon":
			if f, ok := raw.Value().(float64); ok {
				gps[t].Lon = float32(f)
			}
		case "Spd":
			if f, ok := raw.Value().(float64); ok {
				gps[t].Spd = float32(f)
			}
		}
	}
	if err := results.Err(); err != nil {
		return nil, errors.Wrap(err, "InfluxDB,GPS")
	}

	for _, value := range gps {
		return value, nil
	}

	return nil, nil
}

func (srv *Service) queryLatestACCEL(ctx context.Context, carID string) (*dataACCEL, error) {
	queryAPI := srv.Influxdb.QueryAPI("Kaste")

	// latest data from ACCEL
	query := `
from(bucket: "CarData")
  |> range(start: 1882-11-18)
  |> last()
  |> filter(fn: (r) => r["_measurement"] == "ACCEL")
  |> filter(fn: (r) => r["CarID"] == "%s")` // latest data from ACCEL
	results, err := queryAPI.Query(ctx, fmt.Sprintf(query, carID))
	if err != nil {
		return nil, errors.Wrap(err, "InfluxDB,ACCEL")
	}
	/*
		tags["CarID"] = carID
		fields["X"] = data.X
		fields["Y"] = data.Y
		fields["Z"] = data.Z
	*/
	accel := map[time.Time]*dataACCEL{}
	for results.Next() {
		raw := results.Record()
		t := raw.Time()
		if _, found := accel[t]; !found {
			accel[t] = &dataACCEL{Time: t}
		}
		switch raw.Field() {
		case "X":
			if f, ok := raw.Value().(float64); ok {
				accel[t].X = float32(f)
			}
		case "Y":
			if f, ok := raw.Value().(float64); ok {
				accel[t].Y = float32(f)
			}
		case "Z":
			if f, ok := raw.Value().(float64); ok {
				accel[t].Z = float32(f)
			}
		}
	}
	if err := results.Err(); err != nil {
		return nil, errors.Wrap(err, "InfluxDB,ACCEL")
	}

	for _, value := range accel {
		return value, nil
	}

	return nil, nil
}

func (srv *Service) querySUS_SPD(ctx context.Context, carID string) (*dataSUS_SPD, error) {
	queryAPI := srv.Influxdb.QueryAPI("Kaste")

	// latest data from SUS
	query := `
from(bucket: "CarData")
  |> range(start: 1882-11-18)
  |> last()
  |> filter(fn: (r) => r["_measurement"] == "SUS")
  |> filter(fn: (r) => r["CarID"] == "%s")` // latest data from SUS
	results, err := queryAPI.Query(ctx, fmt.Sprintf(query, carID))
	if err != nil {
		return nil, errors.Wrap(err, "InfluxDB,SUS")
	}
	/*
		tags["CarID"] = carID
		fields["SPD"] = data.Spd
	*/
	sus := map[time.Time]*dataSUS_SPD{}
	for results.Next() {
		raw := results.Record()
		t := raw.Time()
		if _, found := sus[t]; !found {
			sus[t] = &dataSUS_SPD{Time: t}
		}
		switch raw.Field() {
		case "Spd":
			if f, ok := raw.Value().(float64); ok {
				sus[t].Spd = float32(f)
			}
		default:
			logrus.Warnf("Unknown field: %s", raw.Field())
		}
	}
	if err := results.Err(); err != nil {
		return nil, errors.Wrap(err, "InfluxDB,SUS")
	}

	for _, value := range sus {
		return value, nil
	}

	return nil, nil
}

func (srv *Service) querySUS_RST(ctx context.Context, carID string) (*dataSUS_RST, error) {
	queryAPI := srv.Influxdb.QueryAPI("Kaste")

	// latest data from SUS
	query := `
from(bucket: "CarData")
  |> range(start: 1882-11-18)
  |> last()
  |> filter(fn: (r) => r["_measurement"] == "SUS")
  |> filter(fn: (r) => r["CarID"] == "%s")` // latest data from SUS
	results, err := queryAPI.Query(ctx, fmt.Sprintf(query, carID))
	if err != nil {
		return nil, errors.Wrap(err, "InfluxDB,SUS")
	}
	/*
		tags["CarID"] = carID
		fields["RST"] = data.Rst
	*/
	sus := map[time.Time]*dataSUS_RST{}
	for results.Next() {
		raw := results.Record()
		t := raw.Time()
		if _, found := sus[t]; !found {
			sus[t] = &dataSUS_RST{Time: t}
		}
		switch raw.Field() {
		case "Rst":
			if f, ok := raw.Value().(int); ok {
				sus[t].Rst = f
			}
		default:
			logrus.Warnf("Unknown field: %s", raw.Field())
		}
	}
	if err := results.Err(); err != nil {
		return nil, errors.Wrap(err, "InfluxDB,SUS")
	}

	for _, value := range sus {
		return value, nil
	}

	return nil, nil
}

func (srv *Service) queryLatestData(ctx context.Context, carID string) (*dataCarFull, error) {
	psu, err := srv.queryLatestPSU(ctx, carID)
	if err != nil {
		return nil, errors.Wrap(err, "PSU")
	}

	gps, err := srv.queryLatestGPS(ctx, carID)
	if err != nil {
		return nil, errors.Wrap(err, "GPS")
	}

	accel, err := srv.queryLatestACCEL(ctx, carID)
	if err != nil {
		return nil, errors.Wrap(err, "ACCEL")
	}

	sus_spd, err := srv.querySUS_SPD(ctx, carID)
	if err != nil {
		return nil, errors.Wrap(err, "SUS")
	}

	sus_rst, err := srv.querySUS_RST(ctx, carID)
	if err != nil {
		return nil, errors.Wrap(err, "SUS")
	}

	result := dataCarFull{}
	result.PSU = psu
	result.GPS = gps
	result.ACCEL = accel
	result.SUS_SPD = sus_spd
	result.SUS_RST = sus_rst

	return &result, nil
}
