package master

import (
	"encoding/json"
	"fmt"
	"net/http"

	httprouter "github.com/julienschmidt/httprouter"
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

	// get carID from URL query
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
