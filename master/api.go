package master

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	httprouter "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (srv *Service) getParameters(w http.ResponseWriter, r *http.Request, ps httprouter.Params) { // GET /api/parameters
	logrus.Debugf("got get request %+v", ps)

	params := srv.CarTable.GetCarParameters()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(params)
}

func (srv *Service) postParameters(w http.ResponseWriter, r *http.Request, ps httprouter.Params) { // POST /api/parameters
	logrus.Debugf("got post request %+v", ps)

	errorHandler := func(err error, code int) {
		logrus.WithError(err).Error("Error")
		http.Error(w, err.Error(), code)
	}

	var params []CarParameters
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorHandler(errors.Wrap(err, "ReadAll"), http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &params); err != nil {
		errorHandler(errors.Wrap(err, "Unmarshal"), http.StatusBadRequest)
		return
	}

	srv.CarTable.UpdateCarParameters(params)

	w.WriteHeader(http.StatusOK)
}

func (srv *Service) getRaceConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) { // GET /api/raceconfiguration
	logrus.Debugf("got get request %+v", ps)

	configs := srv.RaceTable.GetRaceConfig()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(configs)
}

func (srv *Service) postRaceConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) { // POST /api/raceconfiguration
	logrus.Debugf("got post request %+v", ps)

	errorHandler := func(err error, code int) {
		logrus.WithError(err).Error("Error")
		http.Error(w, err.Error(), code)
	}

	var configs []RaceConfig
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorHandler(errors.Wrap(err, "ReadAll"), http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &configs); err != nil {
		errorHandler(errors.Wrap(err, "Unmarshal"), http.StatusBadRequest)
		return
	}

	srv.RaceTable.UpdateRaceConfig(configs)

	w.WriteHeader(http.StatusOK)
}

func (srv *Service) startRace(w http.ResponseWriter, r *http.Request, ps httprouter.Params) { // POST /api/start-race
	logrus.Debugf("got post startRace request %+v", ps)

	errorHandler := func(err error, code int) {
		logrus.WithError(err).Error("Error")
		http.Error(w, err.Error(), code)
	}

	var race RaceConfig
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorHandler(errors.Wrap(err, "ReadAll"), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &race); err != nil {
		errorHandler(errors.Wrap(err, "Unmarshal"), http.StatusBadRequest)
		return
	}

	logrus.Debugf("Starting race %+v", race)

	if r, ok := srv.RaceTable[race.Name]; ok {
		CurrentRace = &r
	} else {
		errorHandler(fmt.Errorf("race '%s' not found", race.Name), http.StatusInternalServerError)
	}

	err = srv.CarTable.StartRace(srv)
	if err != nil {
		errorHandler(err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (srv *Service) endRace(w http.ResponseWriter, r *http.Request, ps httprouter.Params) { // POST /api/end-race
	logrus.Debugf("got post endRace request %+v", ps)

	errorHandler := func(err error, code int) {
		logrus.WithError(err).Error("Error")
		http.Error(w, err.Error(), code)
	}

	var race RaceConfig
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorHandler(errors.Wrap(err, "ReadAll"), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &race); err != nil {
		errorHandler(errors.Wrap(err, "Unmarshal"), http.StatusBadRequest)
		return
	}

	logrus.Debugf("Ending race %+v", race)

	if race.Name == CurrentRace.Name {
		err = srv.CarTable.EndRace(srv)
		if err != nil {
			errorHandler(err, http.StatusInternalServerError)
			return
		}
	} else {
		errorHandler(fmt.Errorf("cannot end race '%s' when current race is '%s'", race.Name, CurrentRace.Name), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (srv *Service) whoFinished(w http.ResponseWriter, r *http.Request, ps httprouter.Params) { // POST /api/whofinished
	logrus.Debugf("got post whoFinished request %+v", ps)

	errorHandler := func(err error, code int) {
		logrus.WithError(err).Error("Error")
		http.Error(w, err.Error(), code)
	}

	var param CarParameters
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorHandler(errors.Wrap(err, "ReadAll"), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &param); err != nil {
		errorHandler(errors.Wrap(err, "Unmarshal"), http.StatusBadRequest)
		return
	}

	logrus.Debugf("Getting who finished for car %+v", param)

	if _, ok := srv.CarTable[param.CarID]; ok {
		err = srv.CarTable.FinishRace(srv, param.CarID)
		if err != nil {
			errorHandler(err, http.StatusInternalServerError)
			return
		}
	} else {
		errorHandler(fmt.Errorf("car '%s' not found", param.CarID), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

// ----------------------------------------------------------------

func (srv *Service) getLatestData(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logrus.Debugf("got get request %+v", ps)

	errorHandler := func(err error) {
		logrus.WithError(err).Error("Error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	/*
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode("Hello, world")
	*/

	// get carID from URL
	carID := ps.ByName("car")

	// get latest data from InfluxDB for given carID
	data, err := srv.queryLatestData(r.Context(), carID)
	if err != nil {
		errorHandler(err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(data)
	}
}

func (srv *Service) getOutdoors(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("got /outdoors request\n")

	errorHandler := func(err error) {
		fmt.Printf("Error: '%s'\n", err.Error())
		http.Error(w, "Error: ", http.StatusInternalServerError)
	}

	data, err := srv.queryMeasurements(r.Context())
	if err != nil {
		errorHandler(err)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(data)
	}
}

// func (srv *Service) setMass(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
// 	logrus.Debugf("got setMass request %+v, %+v", ps, r.URL.Query())

// 	errorHandler := func(err error, code int) {
// 		logrus.WithError(err).Error("Error")
// 		http.Error(w, err.Error(), code)
// 	}

// 	carID := ps.ByName("car")

// 	var mass float64
// 	if m, err := strconv.ParseFloat(r.URL.Query().Get("mass"), 64); err != nil {
// 		errorHandler(errors.Wrap(err, "Invalid mass"), http.StatusBadRequest)
// 		return
// 	} else if m < 0 {
// 		errorHandler(errors.New("Invalid mass"), http.StatusBadRequest)
// 		return
// 	} else {
// 		mass = m
// 	}

// 	var voltage float64
// 	if v, err := strconv.ParseFloat(r.URL.Query().Get("voltage"), 64); err != nil {
// 		errorHandler(errors.Wrap(err, "Invalid voltage"), http.StatusBadRequest)
// 		return
// 	} else if v < 0 {
// 		errorHandler(errors.New("Invalid voltage"), http.StatusBadRequest)
// 		return
// 	} else {
// 		voltage = v
// 	}

// 	var con float64
// 	if c, err := strconv.ParseFloat(r.URL.Query().Get("const"), 64); err != nil {
// 		errorHandler(errors.Wrap(err, "Invalid const"), http.StatusBadRequest)
// 		return
// 	} else if c < 0 {
// 		errorHandler(errors.New("Invalid const"), http.StatusBadRequest)
// 		return
// 	} else {
// 		con = c
// 	}

// 	/*// Retrieve the voltage of car from InfluxDB
// 	psu, err := srv.queryLatestPSU(r.Context(), carID)
// 	if err != nil {
// 		errorHandler(err, http.StatusInternalServerError)
// 		return
// 	}
// 	if psu == nil {
// 		errorHandler(errors.New("No PSU data"), http.StatusUnprocessableEntity)
// 		return
// 	}

// 	// Calculate the new voltage
// 	if psu.Uop == 0 {
// 		errorHandler(errors.New("Uop is 0"), http.StatusUnprocessableEntity)
// 		return
// 	}
// 	payload := dataOutPSU{U: float32(mass*con) / psu.Uop}*/

// 	// Calculate the new max current
// 	var maxI float64
// 	if voltage == 0 {
// 		errorHandler(errors.New("Voltage is 0"), http.StatusUnprocessableEntity)
// 		return
// 	} else if voltage < 0 {
// 		errorHandler(errors.New("Voltage is negative"), http.StatusUnprocessableEntity)
// 		return
// 	} else if con == 0 {
// 		errorHandler(errors.New("Const is 0"), http.StatusUnprocessableEntity)
// 		return
// 	} else if con < 0 {
// 		errorHandler(errors.New("Const is negative"), http.StatusUnprocessableEntity)
// 		return
// 	} else if mass == 0 {
// 		errorHandler(errors.New("Mass is 0"), http.StatusUnprocessableEntity)
// 		return
// 	} else if mass < 0 {
// 		errorHandler(errors.New("Mass is negative"), http.StatusUnprocessableEntity)
// 		return
// 	} else {
// 		maxI = mass * con / voltage
// 	}
// 	srv.dataCarStorage.SetMassAndVoltageAndCurrent(carID, mass, voltage, maxI)
// 	srv.dataCarStorage.Const = con

// 	/*// Publish the new voltage to the car
// 	err := srv.sendPSUData(carID, payload)
// 	if err != nil {
// 		errorHandler(err, http.StatusInternalServerError)
// 	}*/

// 	w.WriteHeader(http.StatusOK)
// 	w.Header().Set("Content-Type", "text/plain")
// 	w.Write([]byte(fmt.Sprintf(`Set mass to '%f', target voltage to '%f' with const to '%f' for car '%s'`, mass, voltage, con, carID)))
// }

func (srv *Service) sendToMqtt(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logrus.Debugf("got send request %+v, %+v", ps, r.URL.Query())

	errorHandler := func(err error, code int) {
		logrus.WithError(err).Error("Error")
		http.Error(w, err.Error(), code)
	}

	topic := ps.ByName("topic")
	if len(topic) > 0 && topic[0] == '/' {
		topic = topic[1:]
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorHandler(errors.Wrap(err, "ReadAll"), http.StatusBadRequest)
		return
	}

	err = srv.sendAnyTopic(topic, body)
	if err != nil {
		errorHandler(err, http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf(`
Sent to '%s'
Payload: '%s'
`, topic, string(body))))
}

func (srv *Service) getMqttLog(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logrus.Debugf("got getMqttLog request")

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(srv.log.GetLogs()))
}

func (srv *Service) deleteMqttLogs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logrus.Debugf("got deleteMqttLogs request")

	srv.log.ClearLogs()

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Logs cleared"))
}

func (srv *Service) triggerRaceStart(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logrus.Debugf("got triggerRaceStart request")

	errorHandler := func(err error, code int) {
		logrus.WithError(err).Error("Error")
		http.Error(w, err.Error(), code)
	}

	err := srv.CarTable.RaceStart(srv)
	if err != nil {
		errorHandler(err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Triggered Race Start"))
}

func (srv *Service) triggerRaceFinish(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logrus.Debugf("got triggerRaceFinish request")

	errorHandler := func(err error, code int) {
		logrus.WithError(err).Error("Error")
		http.Error(w, err.Error(), code)
	}

	carID := ps.ByName("car")
	err := srv.CarTable.CarRaceFinish(carID)
	if err != nil {
		errorHandler(err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("Triggered Car %s Race Finish", carID)))
}
