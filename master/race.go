package master

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type RaceConfig struct {
	Name   string  `json:"RaceName"`
	Length float64 `json:"Length"`
	Coef   float64 `json:"Coef"`
	Weigth float64 `json:"Weight"`
}
type RaceNameMap map[string]RaceConfig

var CurrentRace *RaceConfig

func (table *RaceNameMap) GetRaceConfig() []RaceConfig {
	if *table == nil {
		*table = make(RaceNameMap)
	}
	var configs []RaceConfig
	for _, instance := range *table {
		configs = append(configs, RaceConfig{
			Name:   instance.Name,
			Length: instance.Length,
			Coef:   instance.Coef,
			Weigth: instance.Weigth,
		})
	}
	return configs
}

func (table *RaceNameMap) UpdateRaceConfig(configs []RaceConfig) {
	if *table == nil {
		*table = make(RaceNameMap)
	}
	found := make(map[string]bool, len(*table))
	for _, instance := range configs {
		if _, ok := (*table)[instance.Name]; ok {
			(*table)[instance.Name] = instance
			found[instance.Name] = true
		} else {
			(*table)[instance.Name] = instance
			found[instance.Name] = true
			logrus.Debugf("Race %s was not found in RaceTable, now registered", instance.Name)
		}
	}
	// Remove races that are no longer present in the configs
	for raceName := range *table {
		if _, ok := found[raceName]; !ok {
			delete(*table, raceName)
			logrus.Debugf("Race %s was removed from RaceTable", raceName)
		}
	}
}

func (table *CarIDMap) StartRace(srv *Service) error {
	if *table == nil {
		*table = make(CarIDMap)
	}
	if CurrentRace == nil {
		return errors.New("CurrentRace is nil")
	}
	for carID, car := range *table {
		raceData, exists := car.RaceData[CurrentRace.Name]
		if !exists {
			raceData = CarRaceData{
				TotalWh:  0,
				RaceTime: 0,
				RaceMode: false,
			}
		}
		raceData.TotalWh = 0
		raceData.RaceTime = 0
		raceData.RaceMode = true
		raceData.Finished = false
		car.RaceData[CurrentRace.Name] = raceData
		(*table)[carID] = car

		table.SendMessagePSU(carID, srv)
	}
	return nil
}

func (table *CarIDMap) EndRace(srv *Service) error {
	if *table == nil {
		*table = make(CarIDMap)
	}
	if CurrentRace == nil {
		return errors.New("CurrentRace is nil")
	}
	for carID, car := range *table {
		raceData, exists := car.RaceData[CurrentRace.Name]
		if !exists {
			raceData = CarRaceData{
				TotalWh:  0,
				RaceTime: 0,
				RaceMode: false,
			}
		}
		raceData.RaceMode = false
		car.RaceData[CurrentRace.Name] = raceData
		(*table)[carID] = car

		table.SendMessagePSU(carID, srv)
	}
	return nil
}

func (table *CarIDMap) FinishRace(srv *Service, carID string) error {
	if *table == nil {
		*table = make(CarIDMap)
	}
	if CurrentRace == nil {
		return errors.New("CurrentRace is nil")
	}
	car, exists := (*table)[carID]
	if !exists {
		return errors.Errorf("Car with ID %s not found", carID)
	}

	raceData, exists := car.RaceData[CurrentRace.Name]
	if !exists {
		return errors.Errorf("Race data for race %s not found for car %s", CurrentRace.Name, carID)
	}
	raceData.RaceMode = false
	raceData.Finished = true
	car.RaceData[CurrentRace.Name] = raceData
	(*table)[carID] = car

	return nil
}
