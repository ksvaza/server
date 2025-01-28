package master

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

type dataCarFull struct {
	PSU dataPSU
	GPS dataGPS
}

type carResponse struct {
	Data dataCarFull
}

func (srv *Service) queryLatestData(ctx context.Context, carID string) (*dataCarFull, error) {
	queryAPI := srv.Influxdb.QueryAPI("Kaste")
	// latest data from PSU
	query := `from(bucket: "CarData")` // latest data from PSU
	results, err := queryAPI.Query(ctx, query)
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
		t := raw.Time()
		if _, found := psu[t]; !found {
			psu[t] = &dataPSU{Time: t}
		}
		switch raw.Field() {
		case "Uop":
			if f, ok := raw.Value().(float32); ok {
				psu[t].Uop = f
			}
		case "Iop":
			if f, ok := raw.Value().(float32); ok {
				psu[t].Iop = f
			}
		case "Pop":
			if f, ok := raw.Value().(float32); ok {
				psu[t].Pop = f
			}
		case "Uip":
			if f, ok := raw.Value().(float32); ok {
				psu[t].Uip = f
			}
		case "Wh":
			if f, ok := raw.Value().(float32); ok {
				psu[t].Wh = f
			}
		}
	}
	if err := results.Err(); err != nil {
		return nil, errors.Wrap(err, "InfluxDB,PSU")
	}

	query = `from(bucket: "CarData")` // latest data from GPS
	results, err = queryAPI.Query(ctx, query)
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
			if f, ok := raw.Value().(float32); ok {
				gps[t].Lat = f
			}
		case "Lon":
			if f, ok := raw.Value().(float32); ok {
				gps[t].Lon = f
			}
		case "Spd":
			if f, ok := raw.Value().(float32); ok {
				gps[t].Spd = f
			}
		}
	}
	if err := results.Err(); err != nil {
		return nil, errors.Wrap(err, "InfluxDB,GPS")
	}

	result := &dataCarFull{}
	for _, value := range psu {
		result.PSU = *value
		break
	}
	for _, value := range gps {
		result.GPS = *value
		break
	}

	return result, nil
}
