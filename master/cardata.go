package master

import (
	"time"

	"github.com/sirupsen/logrus"
)

type CarDataStorage struct {
	accumulator         time.Time
	timer               time.Time
	lastLatency         time.Duration
	TotalWh             float64
	lastPower           float64
	LastPORCode         string
	lastMessageWasPower bool
}
type CarMap map[string]CarDataStorage

var CarDataStorageMap CarMap

// func normal message
func (c *CarDataStorage) normalMessage() {
	if c == nil {
		return
	}
	if c.accumulator.IsZero() {
		c.accumulator = time.Now()
		c.timer = time.Now()
	}
	c.lastLatency = time.Since(c.timer)
	c.timer = time.Now()
	c.accumulator = c.timer

	if c.lastMessageWasPower {
		c.TotalWh += c.lastPower * c.lastLatency.Hours()
		c.lastPower = 0
		c.lastMessageWasPower = false
		logrus.Debugf("CarDataStorageMap: %s, TotalWh: %f, lastPower: %f, lastLatency: %s", c.LastPORCode, c.TotalWh, c.lastPower, c.lastLatency)
	}
}

func (m *CarMap) NormalMessage(carID string) {
	if instance, ok := (*m)[carID]; ok {
		instance.normalMessage()
		(*m)[carID] = instance
	} else {
		(*m)[carID] = CarDataStorage{
			accumulator: time.Now(),
			timer:       time.Now(),
			TotalWh:     0,
			lastPower:   0,
			LastPORCode: "",
		}
	}
}

// func error message
func (c *CarDataStorage) errorMessage() {
	if c == nil {
		return
	}
	if c.accumulator.IsZero() {
		c.accumulator = time.Now()
		c.timer = time.Now()
	}
	c.lastLatency = time.Since(c.timer)
	c.timer = c.accumulator
}

func (m *CarMap) ErrorMessage(carID string) {
	if instance, ok := (*m)[carID]; ok {
		instance.errorMessage()
		(*m)[carID] = instance
	} else {
		(*m)[carID] = CarDataStorage{
			accumulator: time.Now(),
			timer:       time.Now(),
			TotalWh:     0,
			lastPower:   0,
			LastPORCode: "",
		}
	}
}

// func set power
func (m *CarMap) SetPower(carID string, power float64) {
	if instance, ok := (*m)[carID]; ok {
		instance.lastPower = power
		instance.lastMessageWasPower = true
		(*m)[carID] = instance
	} else {
		(*m)[carID] = CarDataStorage{
			accumulator: time.Now(),
			timer:       time.Now(),
			TotalWh:     0,
			lastPower:   power,
			LastPORCode: "",
		}
	}
}

// func set PORCode
func (m *CarMap) SetPOR(carID string, porCode string) {
	if instance, ok := (*m)[carID]; ok {
		instance.LastPORCode = porCode
		instance.lastMessageWasPower = false
		(*m)[carID] = instance
	} else {
		(*m)[carID] = CarDataStorage{
			accumulator: time.Now(),
			timer:       time.Now(),
			TotalWh:     0,
			lastPower:   0,
			LastPORCode: porCode,
		}
	}
}
