package master

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

func (a *AllData) MqttMessagePSU(carID string, power float64) (float64, error) {
	if car, ok := a.CarMap[carID]; ok {
		race := car.CurrentRace
		if race == nil {
			return 0, errors.New(fmt.Sprintf("CurrentRace is nil for car %s", carID))
		}
		if raceData, exists := race.RaceData[carID]; exists {
			delta := time.Since(raceData.timer)
			err := a.MqttMessageAny(carID)
			if err != nil {
				return 0, err
			}
			if raceData.RaceMode {
				raceData.TotalWh += power * float64(delta.Hours())
			}
			race.RaceData[carID] = raceData
			return raceData.TotalWh, nil
		}
		return 0, errors.New(fmt.Sprintf("Race data not found for car %s", carID))
	}
	return 0, errors.New(fmt.Sprintf("Car %s not found", carID))
}

func (a *AllData) MqttMessageAny(carID string) error {
	if car, ok := a.CarMap[carID]; ok {
		race := car.CurrentRace
		if race == nil {
			return errors.New(fmt.Sprintf("CurrentRace is nil for car %s", carID))
		}
		if raceData, exists := race.RaceData[carID]; exists {
			delta := time.Since(raceData.timer)
			raceData.timer = time.Now()
			if raceData.RaceMode {
				raceData.RaceTime += delta
			}
			race.RaceData[carID] = raceData
			return nil
		}
		return errors.New(fmt.Sprintf("Race data not found for car %s", carID))
	}
	return errors.New(fmt.Sprintf("Car %s not found", carID))
}

func (a *AllData) MqttMessageRST(carID string, porCode string, srv *Service) error {
	if car, ok := a.CarMap[carID]; ok {
		race := car.CurrentRace
		if race == nil {
			return errors.New(fmt.Sprintf("CurrentRace is nil for car %s", carID))
		}
		if raceData, exists := race.RaceData[carID]; exists {
			raceData.timer = time.Now()

			payload := dataOutPSU{
				U:      float32(car.Params.SetVoltage),
				I:      float32(car.Params.MaxCurrent),
				Status: 1,
			}

			srv.sendPSUData(carID, payload)

			return nil
		}
		return errors.New(fmt.Sprintf("Race data not found for car %s", carID))
	}
	return errors.New(fmt.Sprintf("Car %s not found", carID))
}
