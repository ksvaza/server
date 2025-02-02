package master

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	httprouter "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

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

func (srv *Service) setMass(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	logrus.Debugf("got setMass request %+v, %+v", ps, r.URL.Query())

	errorHandler := func(err error, code int) {
		logrus.WithError(err).Error("Error")
		http.Error(w, err.Error(), code)
	}

	carID := ps.ByName("car")

	var mass float64
	if m, err := strconv.ParseFloat(r.URL.Query().Get("mass"), 64); err != nil {
		errorHandler(errors.Wrap(err, "Invalid mass"), http.StatusBadRequest)
		return
	} else if m < 0 {
		errorHandler(errors.New("Invalid mass"), http.StatusBadRequest)
		return
	} else {
		mass = m
	}

	var con float64
	if c, err := strconv.ParseFloat(r.URL.Query().Get("const"), 64); err != nil {
		errorHandler(errors.Wrap(err, "Invalid const"), http.StatusBadRequest)
		return
	} else if c < 0 {
		errorHandler(errors.New("Invalid const"), http.StatusBadRequest)
		return
	} else {
		con = c
	}

	// Retrieve the voltage of car from InfluxDB
	psu, err := srv.queryLatestPSU(r.Context(), carID)
	if err != nil {
		errorHandler(err, http.StatusInternalServerError)
		return
	}
	if psu == nil {
		errorHandler(errors.New("No PSU data"), http.StatusUnprocessableEntity)
		return
	}

	// Calculate the new voltage
	if psu.Uop == 0 {
		errorHandler(errors.New("Uop is 0"), http.StatusUnprocessableEntity)
		return
	}
	payload := dataOutPSU{U: float32(mass*con) / psu.Uop}

	// Publish the new voltage to the car
	err = srv.sendPSUData(carID, payload)
	if err != nil {
		errorHandler(err, http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(""))
}
