package master

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type CarParameters struct {
	CarID      string  `json:"id"`
	Username   string  `json:"username"`
	Avatar     string  `json:"avatar"`
	SetVoltage float64 `json:"U"`
	MaxCurrent float64 `json:"I"`
	Mass       float64 `json:"m"`
}
type CarRaceData struct {
	TotalWh  float64
	RaceTime time.Duration
	RaceMode bool
	Finished bool
	timer    time.Time
}

type CarData struct {
	Params   CarParameters
	RaceData map[string]CarRaceData // map of [race name]
}
type CarIDMap map[string]CarData

// ----------------------------------------------------------------

func (table *CarIDMap) GetCarParameters() []CarParameters {
	if *table == nil {
		*table = make(CarIDMap)
	}
	logrus.Printf("%+v", *table)
	var parameters []CarParameters
	for _, instance := range *table {
		parameters = append(parameters, CarParameters{
			CarID:      instance.Params.CarID,
			Username:   instance.Params.Username,
			Avatar:     instance.Params.Avatar,
			SetVoltage: instance.Params.SetVoltage,
			MaxCurrent: instance.Params.MaxCurrent,
			Mass:       instance.Params.Mass,
		})
	}
	return parameters
}

func (table *CarIDMap) UpdateCarParameters(parameters []CarParameters) {
	if *table == nil {
		*table = make(CarIDMap)
	}
	found := make(map[string]bool, len(*table))
	for _, instance := range parameters {
		if car, ok := (*table)[instance.CarID]; ok {
			if car.RaceData == nil {
				car.RaceData = make(map[string]CarRaceData)
			}
			car.Params.CarID = instance.CarID
			car.Params.Username = instance.Username
			car.Params.Avatar = instance.Avatar
			car.Params.SetVoltage = instance.SetVoltage
			car.Params.MaxCurrent = instance.MaxCurrent
			car.Params.Mass = instance.Mass
			(*table)[instance.CarID] = car
			found[instance.CarID] = true
		} else {
			// Register the car if not found
			(*table)[instance.CarID] = CarData{
				Params: CarParameters{
					CarID:      instance.CarID,
					Username:   instance.Username,
					Avatar:     instance.Avatar,
					SetVoltage: instance.SetVoltage,
					MaxCurrent: instance.MaxCurrent,
					Mass:       instance.Mass,
				},
				RaceData: make(map[string]CarRaceData),
			}
			found[instance.CarID] = true
			logrus.Debugf("Car %s was not found in DataCarStorage, now registered", instance.CarID)
		}
	}
	// Remove cars that are no longer present in the parameters
	for carID := range *table {
		if _, ok := found[carID]; !ok {
			delete(*table, carID)
			logrus.Debugf("Car %s was removed from DataCarStorage", carID)
		}
	}
}

// ----------------------------------------------------------------

func (table *CarIDMap) RaceStart(srv *Service) error {
	if *table == nil {
		*table = make(CarIDMap)
	}
	if CurrentRaceConfig == nil {
		return errors.New("CurrentRace is nil")
	}
	for carID, car := range *table {
		raceData, exists := car.RaceData[CurrentRaceConfig.Name]
		if !exists {
			raceData = CarRaceData{
				TotalWh:  0,
				RaceTime: 0,
				RaceMode: false,
				Finished: false,
			}
		}
		raceData.RaceTime = 0
		raceData.RaceMode = true
		car.RaceData[CurrentRaceConfig.Name] = raceData
		(*table)[carID] = car

		table.SendMessagePSU(carID, srv)
	}
	return nil
}

func (table *CarIDMap) CarRaceFinish(carID string) error {
	if *table == nil {
		*table = make(CarIDMap)
	}
	if car, ok := (*table)[carID]; ok {
		if raceData, exists := car.RaceData[CurrentRaceConfig.Name]; exists {
			raceData.RaceMode = false
			car.RaceData[CurrentRaceConfig.Name] = raceData
			(*table)[carID] = car
			return nil
		}
		return errors.New(fmt.Sprintf("Race data not found for car %s", carID))
	} else {
		return errors.New(fmt.Sprintf("Car %s not found", carID))
	}
}

func (table *CarIDMap) MqttMessagePSU(carID string, power float64, consumption float64) (float64, error) {
	if *table == nil {
		*table = make(CarIDMap)
	}
	if car, ok := (*table)[carID]; ok {
		if raceData, exists := car.RaceData[CurrentRaceConfig.Name]; exists {
			// delta := time.Since(raceData.timer)
			err := table.MqttMessageAny(carID)
			if err != nil {
				return 0, err
			}
			delta := consumption - raceData.TotalWh
			if delta > 0 {
				raceData.TotalWh += delta
			} else {
				logrus.Warnf("Negative consumption value for car %s: %f", carID, delta)
			}
			// if raceData.RaceMode {
			// 	raceData.TotalWh += power * float64(delta.Hours())
			// }
			car.RaceData[CurrentRaceConfig.Name] = raceData
			(*table)[carID] = car
			return raceData.TotalWh, nil
		}
		return 0, errors.New(fmt.Sprintf("Race data not found for car %s", carID))
	}
	return 0, errors.New(fmt.Sprintf("Car %s not found", carID))
}

func (table *CarIDMap) MqttMessageAny(carID string) error {
	if *table == nil {
		*table = make(CarIDMap)
	}
	if car, ok := (*table)[carID]; ok {
		if raceData, exists := car.RaceData[CurrentRaceConfig.Name]; exists {
			delta := time.Since(raceData.timer)
			raceData.timer = time.Now()
			if raceData.RaceMode {
				raceData.RaceTime += delta
			}
			car.RaceData[CurrentRaceConfig.Name] = raceData
			(*table)[carID] = car
			return nil
		}
		return errors.New(fmt.Sprintf("Race data not found for car %s", carID))
	} else {
		return errors.New(fmt.Sprintf("Car %s not found", carID))
	}
}

func (table *CarIDMap) MqttMessageRST(carID string, porCode string, srv *Service) error {
	if *table == nil {
		*table = make(CarIDMap)
	}
	if car, ok := (*table)[carID]; ok {
		if raceData, exists := car.RaceData[CurrentRaceConfig.Name]; exists {
			raceData.timer = time.Now()
			car.RaceData[CurrentRaceConfig.Name] = raceData
			(*table)[carID] = car
			table.SendMessagePSU(carID, srv)
			return nil
		}
		return errors.New(fmt.Sprintf("Race data not found for car %s", carID))
	}
	return errors.New(fmt.Sprintf("Car %s not found", carID))
}

func (table *CarIDMap) SendMessagePSU(carID string, srv *Service) {
	if *table == nil {
		*table = make(CarIDMap)
	}
	if car, ok := (*table)[carID]; ok {
		payload := dataOutPSU{U: float32(car.Params.SetVoltage), I: float32(car.Params.MaxCurrent)}
		srv.sendPSUData(carID, payload)
	}
}
