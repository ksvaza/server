package master

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type AllData struct {
	UUID          uuid.UUID // Unique identifier for this instance
	LastSave      time.Time // Timestamp of last file save
	Settings      Settings
	CarMap        map[string]Car                // map of [carID]
	Races         map[string]Race               // map of [raceName_Lap]
	Leaderboards  map[string][]LeaderboardEntry // map of [ageGroup]
	LiveData      map[string]LiveDataInstance   // map of [carID]
	LiveDataMutex sync.Mutex                    // Mutex to protect LiveData access
}

type Car struct {
	Params            Parameters
	CurrentRace       *Race
	SpeedTestIterator int
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
	RaceData map[string]RaceData `json:"RaceData"` // map of [carID]
}

type StartInstance struct {
	RaceName string `json:"raceName"`
	Lap      int    `json:"Lap"`
	CarID    string `json:"ID"`
}

type FinishInstance struct {
	CarID string `json:"ID"`
}

type PointsInstance struct {
	CarID  string `json:"ID"`
	Points int    `json:"Points"`
}

type Points struct {
	CategoryName string           `json:"CategoryName"`
	Points       []PointsInstance `json:"Points"`
}

type Settings struct {
	RaceCoeficient float64 `json:"PowerCoef"`
	MaxSpeed       float64 `json:"MaxSpd"`
}

//	{
//	    "key": "69",
//	    "ID": 69,
//	    "username": "Alice",
//	    "avatar": "https://m.media-amazon.com/images/S/pv-target-images/16627900db04b76fae3b64266ca161511422059cd24062fb5d900971003a0b70.jpg",
//	    "CategoryNames": ["CategoryA","CategoryB","CategoryC","CategoryD"],
//	    "CategoryPoints": [10,15,8,1],
//	    "status": "online",
//	    "position": 1,
//	    "lat": 56.660211339105715,
//	    "lon": 23.744293111677134,
//	    "spd": 12.3,
//	    "power": 98,
//	    "acceleration": 0.2,
//	    "voltage": 62,
//	    "updatedAt": "2025-05-14T22:36:00Z"
//	  },
type LiveDataInstance struct {
	Key        string    `json:"key"`
	CarID      string    `json:"ID"`
	Username   string    `json:"username"`
	Avatar     string    `json:"avatar"`
	Categories []string  `json:"CategoryNames"`
	Points     []int     `json:"CategoryPoints"`
	Position   int       `json:"position"`
	Lat        float64   `json:"lat"`
	Lon        float64   `json:"lon"`
	Speed      float64   `json:"spd"`
	Power      float64   `json:"power"`
	Accel      float64   `json:"acceleration"`
	Voltage    float64   `json:"voltage"`
	UpdatedAt  time.Time `json:"updatedAt"`
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

func (a *AllData) UpdateCars(cars []Parameters, srv *Service) {
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
	// Update PSU data for all registered cars based on new settings
	for carID, car := range a.CarMap {
		// Calculate max current
		if car.Params.SetVoltage <= 0 {
			logrus.Warnf("SetVoltage for car %s is zero or negative, skipping PSU update", carID)
			continue
		}
		maxCurrent := car.Params.Mass * a.Settings.RaceCoeficient / car.Params.SetVoltage
		car.Params.MaxCurrent = maxCurrent

		payload := dataOutPSU{
			U:      float32(car.Params.SetVoltage),
			I:      float32(car.Params.MaxCurrent),
			Status: 1,
		}

		srv.AllData.AddCarToLiveData(carID, car.Params.Username, car.Params.Avatar)

		srv.sendPSUData(carID, payload)
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
			var efficiency, shellEff, avgPower, avgSpeed float64
			// Calculate efficiency metrics
			if race.Length > 0 && data.RaceTime > 0 {
				efficiency = data.TotalWh / (race.Length / 1000) / car.Params.Mass
				shellEff = (race.Length / 1000) / (data.TotalWh / 1000)
				avgPower = data.TotalWh / data.RaceTime.Hours()
				avgSpeed = (race.Length / 1000) / data.RaceTime.Hours()
			}
			// efficiency := data.TotalWh / (race.Length / 1000) / car.Params.Mass // Wh/km/kg
			// shellEff := (race.Length / 1000) / (data.TotalWh / 1000)            // km/kWh
			// avgPower := data.TotalWh / data.RaceTime.Hours()                    // W
			// avgSpeed := (race.Length / 1000) / data.RaceTime.Hours()            // km/h

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

func (a *AllData) UpdateLeaderboard() error {
	// Group cars by age group
	ageGroups := make(map[string][]Car)
	allCars := []Car{}

	for _, car := range a.CarMap {
		ageGroup := car.Params.AgeGroup
		ageGroups[ageGroup] = append(ageGroups[ageGroup], car)
		allCars = append(allCars, car) // Add to overall group
	}
	ageGroups["all"] = allCars // Add "all" group for overall leaderboard

	categs := map[string][]string{}
	pts := map[string][]int{}

	// Process each age group (including "all" string for overall)
	for ageGroup := range ageGroups {
		entries := make([]LeaderboardEntry, 0)
		cars := ageGroups[ageGroup]

		// Create entries for each car
		for _, car := range cars {
			var categories []string
			var points []int
			validPoints := false

			// Collect race data
			for _, race := range a.Races {
				if raceData, participated := race.RaceData[car.Params.CarID]; participated {
					if raceData.Points >= 0 {
						categories = append(categories, race.RaceName)
						points = append(points, raceData.Points)
						validPoints = true
					}
				}
			}

			// Only add cars with valid points
			if validPoints {
				entry := LeaderboardEntry{
					CarID:      car.Params.CarID,
					Username:   car.Params.Username,
					Avatar:     car.Params.Avatar,
					Categories: categories,
					Points:     points,
				}
				categs[car.Params.CarID] = categories
				pts[car.Params.CarID] = points
				entries = append(entries, entry)
			}
		}

		// Sort entries by total points
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

		// Set positions
		for i := range entries {
			entries[i].Position = i + 1
			if prev, ok := a.Leaderboards[ageGroup]; ok {
				// Find previous position for relative position calculation
				for _, lastEntry := range prev {
					a.UpdateLiveDataCarPositionCategory(lastEntry.CarID, categs[lastEntry.CarID], pts[lastEntry.CarID], entries[i].Position)
					if lastEntry.CarID == entries[i].CarID {
						entries[i].RelPos = lastEntry.Position - entries[i].Position
						break
					}
				}
			}
		}

		// Update leaderboard for this age group
		if a.Leaderboards == nil {
			a.Leaderboards = make(map[string][]LeaderboardEntry)
		}
		a.Leaderboards[ageGroup] = entries
	}

	return nil
}

func (a *AllData) GetLeaderboard(ageGroup string) ([]LeaderboardEntry, error) {
	if a.Leaderboards == nil {
		a.Leaderboards = make(map[string][]LeaderboardEntry)
	}
	entries, ok := a.Leaderboards[ageGroup]
	if !ok {
		return nil, fmt.Errorf("no leaderboard found for age group %s", ageGroup)
	}
	return entries, nil
}

func (a *AllData) DeleteLeaderboard(ageGroup string) error {
	if a.Leaderboards == nil {
		a.Leaderboards = make(map[string][]LeaderboardEntry)
	}
	if _, ok := a.Leaderboards[ageGroup]; !ok {
		return fmt.Errorf("no leaderboard found for age group %s", ageGroup)
	}
	if entries, ok := a.Leaderboards[ageGroup]; ok {
		for i := range entries {
			entries[i].RelPos = 0
		}
		a.Leaderboards[ageGroup] = entries
	}
	return nil
}

func (a *AllData) StartRace(s StartInstance) error {
	car, ok := a.CarMap[s.CarID]
	if !ok {
		return fmt.Errorf("car with ID '%s' not found", s.CarID)
	}

	race, ok := a.Races[s.RaceName+"_"+strconv.Itoa(s.Lap)]
	if !ok {
		return fmt.Errorf("race '%s' not found", s.RaceName)
	}

	if car.CurrentRace != nil {
		return fmt.Errorf("car '%s' is already in race '%s'", s.CarID, car.CurrentRace.RaceName)
	}

	// Ensure RaceData map is initialized
	if race.RaceData == nil {
		race.RaceData = make(map[string]RaceData)
	}

	// Initialize race data for this car-+
	race.RaceData[s.CarID] = RaceData{
		Position: 0,
		Points:   0,
		TotalWh:  0,
		RaceTime: 0,
		Finished: false,
		timer:    time.Now(),
		RaceMode: true,
	}

	// Update car's current race
	car.CurrentRace = &race
	a.CarMap[s.CarID] = car
	return nil
}

func (a *AllData) RaceFinish(r Race, srv *Service) error {
	if _, ok := a.Races[raceKey(r)]; !ok {
		return fmt.Errorf("race '%s' not found", r.RaceName)
	}

	// Check all cars for unfinished races
	for carID, car := range a.CarMap {
		if car.CurrentRace != nil && car.CurrentRace.RaceName == r.RaceName && car.CurrentRace.Lap == r.Lap {
			if car.CurrentRace != nil && car.CurrentRace.RaceName == r.RaceName && car.CurrentRace.Lap == r.Lap {
				// Get race data for this car
				raceData := car.CurrentRace.RaceData[carID]
				raceData.FactualTime = time.Since(raceData.timer)
				raceData.RaceMode = false
				car.CurrentRace.RaceData[carID] = raceData

				// Clear car's current race
				car.CurrentRace = nil
				a.CarMap[carID] = car
			}
		}
	}
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

	// Ensure RaceData map is initialized
	if car.CurrentRace.RaceData == nil {
		car.CurrentRace.RaceData = make(map[string]RaceData)
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

func (a *AllData) UpdatePoints(p Points) error {
	// Iterate through all races to find matching race names
	found := false
	for raceKey, race := range a.Races {
		if race.RaceName == p.CategoryName {
			// Initialize RaceData map if nil
			if race.RaceData == nil {
				race.RaceData = make(map[string]RaceData)
			}
			// Update points for each car in the points instance
			for _, pi := range p.Points {
				if raceData, exists := race.RaceData[pi.CarID]; exists {
					raceData.Points = pi.Points
					race.RaceData[pi.CarID] = raceData
				} else {
					// If car not found, create new RaceData entry
					raceData = RaceData{
						Position: 0,
						Points:   pi.Points,
					}
					race.RaceData[pi.CarID] = raceData
				}
				// Store updated race back in races map
				a.Races[raceKey] = race
				found = true
			}
		}
	}
	if !found {
		// create a new race (category) if not found
		newRace := Race{
			RaceName: p.CategoryName,
			RaceData: make(map[string]RaceData),
		}
		// Add points for each car
		for _, pi := range p.Points {
			newRace.RaceData[pi.CarID] = RaceData{
				Position: 0,
				Points:   pi.Points,
			}
		}
		// Add new race to races map
		a.Races[raceKey(newRace)] = newRace
	}

	a.UpdateLeaderboard()
	return nil
}

func (a *AllData) ResetPoints(raceName string) error {
	// Iterate through all races with given name
	found := false
	for raceKey, race := range a.Races {
		if race.RaceName == raceName {
			found = true
			// Initialize RaceData if nil
			if race.RaceData == nil {
				race.RaceData = make(map[string]RaceData)
			}
			// Reset points for all cars in the race
			for carID, raceData := range race.RaceData {
				raceData.Points = 0 // Set default point value to -1
				race.RaceData[carID] = raceData
			}
			a.Races[raceKey] = race
		}
	}
	if !found {
		return fmt.Errorf("race '%s' not found", raceName)
	}
	a.UpdateLeaderboard()
	return nil
}

func (a *AllData) SaveToFile() error {
	// Update the timestamp before saving
	a.LastSave = time.Now()

	data, err := json.MarshalIndent(*a, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal AllData: %w", err)
	}
	lastSave, err := a.LastSave.MarshalText()
	if err != nil {
		return fmt.Errorf("failed to marshal LastSave: %w", err)
	}
	folderPath := fmt.Sprintf("home/%s", a.UUID.String())
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	filename := fmt.Sprintf("%s/alldata_%s.json", folderPath, lastSave)
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write AllData to custom file: %w", err)
	}

	// Save copy to default location
	err = os.WriteFile("home/alldata.json", data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write AllData to file: %w", err)
	}
	logrus.Infof("AllData saved to alldata.json")
	return nil
}

func (a *AllData) LoadFromFile() error {
	data, err := os.ReadFile("home/alldata.json")
	if err != nil {
		return fmt.Errorf("failed to read AllData from file: %w", err)
	}

	err = json.Unmarshal(data, a)
	if err != nil {
		return fmt.Errorf("failed to unmarshal AllData: %w", err)
	}

	if a.UUID == (uuid.UUID{}) {
		a.UUID = uuid.New()
		logrus.Infof("Generated new UUID for AllData: %s", a.UUID.String())
	}

	logrus.Infof("AllData loaded from file with UUID: %s", a.UUID.String())
	return nil
}

func (a *AllData) ResetData() {
	a.SaveToFile()
	a.UUID = uuid.New()
	a.LastSave = time.Time{}
	a.Settings.RaceCoeficient = 0
	a.CarMap = make(map[string]Car)
	a.Races = make(map[string]Race)
	a.Leaderboards = make(map[string][]LeaderboardEntry)
}

func (a *AllData) GetSettings() Settings {
	return a.Settings
}

func (a *AllData) UpdateSettings(settings Settings, srv *Service) {
	a.Settings = settings

	// Update PSU data for all registered cars based on new settings
	for carID, car := range a.CarMap {
		// Calculate max current
		if car.Params.SetVoltage <= 0 {
			logrus.Warnf("SetVoltage for car %s is zero or negative, skipping PSU update", carID)
			continue
		}
		maxCurrent := car.Params.Mass * a.Settings.RaceCoeficient / car.Params.SetVoltage
		car.Params.MaxCurrent = maxCurrent
		payload := dataOutPSU{
			U:      float32(car.Params.SetVoltage),
			I:      float32(maxCurrent),
			Status: 1,
		}
		srv.sendPSUData(carID, payload)
	}
}

func (a *AllData) CheckSpeed(carID string, speed float32, srv *Service) {
	if car, ok := a.CarMap[carID]; ok {
		var over bool
		if car.SpeedTestIterator > 6 && car.SpeedTestIterator <= 8 {
			car.SpeedTestIterator++
			return
		} else if car.SpeedTestIterator >= 8 {
			car.SpeedTestIterator = 0
			payload := dataOutPSU{
				U:      float32(car.Params.SetVoltage),
				I:      float32(car.Params.MaxCurrent),
				Status: 1,
			}
			srv.sendPSUData(carID, payload)
		}
		if over = (float64(speed) > a.Settings.MaxSpeed); over {
			car.SpeedTestIterator++
			a.CarMap[carID] = car
		}
		if car.SpeedTestIterator == 6 {
			payload := dataOutPSU{
				U:      float32(car.Params.SetVoltage),
				I:      float32(car.Params.MaxCurrent),
				Status: 0,
			}
			srv.sendPSUData(carID, payload)
			car.SpeedTestIterator++
			a.CarMap[carID] = car
		}
	}
}

func (a *AllData) AddCarToLiveData(carID string, username string, avatar string) {
	a.LiveDataMutex.Lock()
	defer a.LiveDataMutex.Unlock()

	dat := a.LiveData[carID]

	dat.Key = carID
	dat.CarID = carID
	dat.Username = username
	dat.Avatar = avatar
	dat.UpdatedAt = time.Now()

	a.LiveData[carID] = dat
}

func (a *AllData) UpdateLiveDataCarPositionCategory(carID string, Categories []string, Points []int, Position int) {
	a.LiveDataMutex.Lock()
	defer a.LiveDataMutex.Unlock()

	dat := a.LiveData[carID]

	dat.Categories = Categories
	dat.Points = Points
	dat.Position = Position
	dat.UpdatedAt = time.Now()

	a.LiveData[carID] = dat
}

func (a *AllData) UpdateLiveDataCarGPS(carID string, lat, lon, speed float64) {
	a.LiveDataMutex.Lock()
	defer a.LiveDataMutex.Unlock()

	dat := a.LiveData[carID]

	dat.Lat = lat
	dat.Lon = lon
	dat.Speed = speed
	dat.UpdatedAt = time.Now()

	a.LiveData[carID] = dat
}

func (a *AllData) UpdateLiveDataCarAccel(carID string, accel float64) {
	a.LiveDataMutex.Lock()
	defer a.LiveDataMutex.Unlock()

	dat := a.LiveData[carID]

	dat.Accel = accel
	dat.UpdatedAt = time.Now()

	a.LiveData[carID] = dat
}

func (a *AllData) UpdateLiveDataCarPSU(carID string, power, voltage float64) {
	a.LiveDataMutex.Lock()
	defer a.LiveDataMutex.Unlock()

	dat := a.LiveData[carID]

	dat.Power = power
	dat.Voltage = voltage
	dat.UpdatedAt = time.Now()

	a.LiveData[carID] = dat
}

func (a *AllData) LiveDataToJson() string {
	a.LiveDataMutex.Lock()
	defer a.LiveDataMutex.Unlock()

	data, err := json.Marshal(a.LiveData)
	if err != nil {
		return ""
	}
	return string(data)
}
