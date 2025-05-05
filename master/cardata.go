package master

import "time"

type CarDataStorage struct {
	accumulator time.Time
	timer       time.Time
	totalWh     float64
	lastPower   float64
	lastPORCode string
}
type CarMap map[string]CarDataStorage

// func set power
// func set PORCode
// func reset handle
// func normal handle

func (m *CarMap) GetCarDataInstance(carID string) *CarDataStorage {
	if instance, ok := (*m)[carID]; !ok {
		/*(*m)[carID] = CarDataStorage{
			accumulator: time.Now(),
			timer:       time.Now(),
			totalWh:     0,
			lastPower:   0,
			lastPORCode: "",
		}*/
		return nil
	} else {
		return &instance
	}
}

func (l *Log) GetCardataLogs() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	var result string
	for i := len(l.logs) - 1; i >= 0; i-- {
		result += l.logs[i] + "\n"
	}

	return result
}
