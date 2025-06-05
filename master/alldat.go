package master

import (
	"fmt"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
)

type AllData struct {
	RaceCoeficient  float64
	CarMap          map[string]Car  // map of [carID]
	Races           map[string]Race // map of [raceName]
	LastLeaderboard []LeaderboardEntry
}

type Car struct {
	Params      Parameters
	CurrentRace *Race
}

type Parameters struct {
	CarID      string  `json:"id"`
	Username   string  `json:"username"`
	Avatar     string  `json:"avatar"`
	SetVoltage float64 `json:"U"`
	MaxCurrent float64 `json:"I"`
	Mass       float64 `json:"m"`
	AgeGroup   string  `json:"ageGroup"`
}

type RaceData struct {
	Position    int
	Points      int
	TotalWh     float64
	RaceTime    time.Duration
	FactualTime time.Duration // time spent in
	RaceMode    bool
	Finished    bool
	timer       time.Time
}

type Result struct {
	RaceName    string        `json:"RaceName"`
	Lap         int           `json:"Lap"`
	CarID       string        `json:"ID"`
	Username    string        `json:"Username"`
	Avatar      string        `json:"avatar"`
	UsedEnergy  float64       `json:"Used energy"`
	Efficiency  float64       `json:"Efficiency"`
	ShellEff    float64       `json:"Shelleficiency"`
	AvgPower    float64       `json:"Average power"`
	AvgSpeed    float64       `json:"Average speed"`
	ElapsedTime time.Duration `json:"Elapsed time"`
}

// [

// {
//  "ID": 69,
//  "username": "Alice",
//  “avatar”: “url/api/images/avatar.png”,
//  "Category names": ["CategoryA", "CategoryAB", "CategoryC"],
//  "Category points": [10, 15, 8]

// "Position": 4

// "Relative Position": -1
// }

// ]
type LeaderboardEntry struct {
	CarID      string   `json:"ID"`
	Username   string   `json:"username"`
	Avatar     string   `json:"avatar"`
	Categories []string `json:"Category names"`
	Points     []int    `json:"Category points"`
	Position   int      `json:"Position"`
	RelPos     int      `json:"Relative Position"`
}

type Race struct {
	RaceName string              `json:"RaceName"`
	Lap      int                 `json:"Lap"`
	Length   float64             `json:"Length"`
	RaceData map[string]RaceData `json:"-"` // map of [carID]
}

type StartInstance struct {
	RaceName string `json:"raceName"`
	CarID    string `json:"ID"`
}

type FinishInstance struct {
	CarID string `json:"ID"`
}

func (a *AllData) GetCars() []Parameters {
	if a.CarMap == nil {
		a.CarMap = make(map[string]Car)
	}
	var cars []Parameters
	for _, car := range a.CarMap {
		cars = append(cars, car.Params)
	}
	return cars
}

func (a *AllData) UpdateCars(cars []Parameters) {
	if a.CarMap == nil {
		a.CarMap = make(map[string]Car)
	}
	found := make(map[string]bool, len(a.CarMap))
	for _, car := range cars {
		if existingCar, ok := a.CarMap[car.CarID]; ok {
			existingCar.Params = car
			a.CarMap[car.CarID] = existingCar
			found[car.CarID] = true
		} else {
			a.CarMap[car.CarID] = Car{Params: car}
			found[car.CarID] = true
			logrus.Debugf("Car %s was not found in CarMap, now registered", car.CarID)
		}
	}
	// Remove cars that are no longer present in the map
	for carID := range a.CarMap {
		if _, ok := found[carID]; !ok {
			delete(a.CarMap, carID)
			logrus.Debugf("Car %s was removed from CarMap", carID)
		}
	}
}

func raceKey(r Race) string {
	return fmt.Sprintf("%s_%d", r.RaceName, r.Lap)
}

func (a *AllData) GetRaces() []Race {
	if a.Races == nil {
		a.Races = make(map[string]Race)
	}
	races := make([]Race, 0, len(a.Races))
	for _, race := range a.Races {
		races = append(races, race)
	}
	return races
}

func (a *AllData) UpdateRaces(races []Race) {
	if a.Races == nil {
		a.Races = make(map[string]Race)
	}
	found := make(map[string]bool, len(a.Races))
	for _, race := range races {
		key := raceKey(race)
		if existingRace, ok := a.Races[key]; ok {
			existingRace.Lap = race.Lap
			existingRace.Length = race.Length
			a.Races[key] = existingRace
		} else {
			race.RaceData = make(map[string]RaceData)
			a.Races[key] = race
			logrus.Debugf("Race %s was not found in Races, now registered", race.RaceName)
		}
		found[key] = true
	}
	// Remove races that are no longer present in the map
	for raceName := range a.Races {
		if _, ok := found[raceName]; !ok {
			delete(a.Races, raceName)
			logrus.Debugf("Race %s was removed from Races", raceName)
		}
	}
}

func (a *AllData) GetResults(raceName string) ([]Result, error) {
	race, ok := a.Races[raceName]
	if !ok {
		return nil, fmt.Errorf("race '%s' not found", raceName)
	}

	results := make([]Result, 0, len(race.RaceData))
	for carID, data := range race.RaceData {
		if car, exists := a.CarMap[carID]; exists {
			// Calculate efficiency metrics
			efficiency := data.TotalWh / (race.Length / 1000) / car.Params.Mass // Wh/km/kg
			shellEff := (race.Length / 1000) / (data.TotalWh / 1000)            // km/kWh
			avgPower := data.TotalWh / data.RaceTime.Hours()                    // W
			avgSpeed := (race.Length / 1000) / data.RaceTime.Hours()            // km/h

			result := Result{
				RaceName:    race.RaceName,
				Lap:         race.Lap,
				CarID:       car.Params.CarID,
				Username:    car.Params.Username,
				Avatar:      car.Params.Avatar,
				UsedEnergy:  data.TotalWh,
				Efficiency:  efficiency,
				ShellEff:    shellEff,
				AvgPower:    avgPower,
				AvgSpeed:    avgSpeed,
				ElapsedTime: data.RaceTime,
			}
			results = append(results, result)
		}
	}
	return results, nil
}

// [

// {
//  "ID": 69,
//  "username": "Alice",
//  “avatar”: “url/api/images/avatar.png”,
//  "Category names": ["CategoryA", "CategoryAB", "CategoryC"],
//  "Category points": [10, 15, 8]

// "Position": 4

// "Relative Position": -1
// }

// ]
func (a *AllData) UpdateLeaderboard() ([]LeaderboardEntry, error) {
	entries := make([]LeaderboardEntry, 0, len(a.CarMap))

	for carID, car := range a.CarMap {
		var categories []string
		var points []int
		totalPoints := 0

		// Collect race data for this car
		for raceName, race := range a.Races {
			if raceData, participated := race.RaceData[carID]; participated {
				categories = append(categories, raceName)
				points = append(points, raceData.Points)
				totalPoints += raceData.Points
			}
		}

		// Create leaderboard entry
		entry := LeaderboardEntry{
			CarID:      carID,
			Username:   car.Params.Username,
			Avatar:     car.Params.Avatar,
			Categories: categories,
			Points:     points,
		}
		entries = append(entries, entry)
	}

	// Sort entries by total points to determine position
	sort.Slice(entries, func(i, j int) bool {
		sumI := 0
		sumJ := 0
		for _, p := range entries[i].Points {
			sumI += p
		}
		for _, p := range entries[j].Points {
			sumJ += p
		}
		return sumI > sumJ
	})
	// Set positions and calculate relative positions from last leaderboard
	for i := range entries {
		entries[i].Position = i + 1

		// Find car's position in last leaderboard
		for _, lastEntry := range a.LastLeaderboard {
			if lastEntry.CarID == entries[i].CarID {
				entries[i].RelPos = lastEntry.Position - entries[i].Position
				break
			}
		}
	}

	// Update LastLeaderboard with current entries
	a.LastLeaderboard = make([]LeaderboardEntry, len(entries))
	copy(a.LastLeaderboard, entries)

	return entries, nil
}

func (a *AllData) GetLeaderboard() ([]LeaderboardEntry, error) {
	a.UpdateLeaderboard()
	if a.LastLeaderboard == nil {
		return nil, fmt.Errorf("leaderboard not initialized")
	}
	return a.LastLeaderboard, nil
}

func (a *AllData) StartRace(s StartInstance, srv *Service) error {
	car, ok := a.CarMap[s.CarID]
	if !ok {
		return fmt.Errorf("car with ID '%s' not found", s.CarID)
	}

	race, ok := a.Races[s.RaceName]
	if !ok {
		return fmt.Errorf("race '%s' not found", s.RaceName)
	}

	if car.CurrentRace != nil {
		return fmt.Errorf("car '%s' is already in race '%s'", s.CarID, car.CurrentRace.RaceName)
	}

	// Initialize race data for this car
	race.RaceData[s.CarID] = RaceData{
		Position: 0,
		Points:   0,
		TotalWh:  0,
		RaceTime: 0,
		Finished: false,
		timer:    time.Now(),
		RaceMode: true,
	}

	// Calculate max current
	maxCurrent := car.Params.Mass * a.RaceCoeficient / car.Params.SetVoltage
	car.Params.MaxCurrent = maxCurrent

	payload := dataOutPSU{
		U: float32(car.Params.SetVoltage),
		I: float32(car.Params.MaxCurrent),
	}

	srv.sendPSUData(s.CarID, payload)

	// Update car's current race
	car.CurrentRace = &race
	a.CarMap[s.CarID] = car

	return nil
}

func (a *AllData) CarRaceFinish(s FinishInstance, srv *Service) error {
	car, ok := a.CarMap[s.CarID]
	if !ok {
		return fmt.Errorf("car with ID '%s' not found", s.CarID)
	}

	if car.CurrentRace == nil {
		return fmt.Errorf("car '%s' is not in any race", s.CarID)
	}

	// Get race data for this car
	raceData := car.CurrentRace.RaceData[s.CarID]
	raceData.FactualTime = time.Since(raceData.timer)
	raceData.Finished = true
	raceData.RaceMode = false
	car.CurrentRace.RaceData[s.CarID] = raceData

	// Clear car's current race
	car.CurrentRace = nil
	a.CarMap[s.CarID] = car

	return nil
}
