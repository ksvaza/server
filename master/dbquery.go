package master

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type dataCarFull struct {
	PSU dataPSU `json:"PSU,omitempty"`
	GPS dataGPS `json:"GPS,omitempty"`
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

func (srv *Service) queryLatestData(ctx context.Context, carID string) (*dataCarFull, error) {
	psu, err := srv.queryLatestPSU(ctx, carID)
	if err != nil {
		return nil, errors.Wrap(err, "PSU")
	}

	gps, err := srv.queryLatestGPS(ctx, carID)
	if err != nil {
		return nil, errors.Wrap(err, "GPS")
	}

	result := dataCarFull{}
	if psu != nil {
		result.PSU = *psu
	}
	if gps != nil {
		result.GPS = *gps
	}

	return &result, nil
}
