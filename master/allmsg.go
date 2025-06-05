package master

func (a *AllData) MqttMessageAny(carID string) error {
	if a.CarMap == nil {
		a.CarMap = make(CarIDMap)
	}
	if car, ok := a.CarMap[carID]; ok {
		if raceData, exists := car.RaceData[CurrentRaceConfig.Name]; exists {
			delta := time.Since(raceData.timer)
			raceData.timer = time.Now()
			if raceData.RaceMode {
				raceData.RaceTime += delta
			}
			func (a *AllData) MqttMessageAny(carID string) error {
				if a.CarMap == nil {
					a.CarMap = make(CarIDMap)
				}
				if car, ok := a.CarMap[carID]; ok {
					if raceData, exists := car.RaceData[CurrentRaceConfig.Name]; exists {
						delta := time.Since(raceData.timer)
						raceData.timer = time.Now()
						if raceData.RaceMode {
							raceData.RaceTime += delta
						}
						car.RaceData[CurrentRaceConfig.Name] = raceData
					}
				}
				return nil
			}
		}
	}
}
