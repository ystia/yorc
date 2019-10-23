// Copyright 2019 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rest

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/log"
)

func (s *Server) listLocations(w http.ResponseWriter, r *http.Request) {
	locs := make([]LocationResult, 0)
	locationConfigurations, err := s.locationMgr.GetLocations()
	if err != nil {
		log.Panic(err)
	}
	for _, locationConfiguration := range locationConfigurations {
		locationName := locationConfiguration.Name
		locationType := locationConfiguration.Type
		locationProps := locationConfiguration.Properties
		locs = append(locs, LocationResult{
			Name:       locationName,
			Type:       locationType,
			Properties: locationProps,
		})
	}
	if len(locs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	encodeJSONResponse(w, r, LocationsCollection{Locations: locs})
}

func (s *Server) createLocation(w http.ResponseWriter, r *http.Request) {
	lConfig, err := getRequestedLocationConfig(r)
	if err != nil {
		writeError(w, r, newBadRequestError(err))
		return
	}

	err = s.locationMgr.CreateLocation(lConfig)
	if err != nil {
		if locations.IsLocationAlreadyExistError(err) {
			writeError(w, r, newBadRequestError(errors.Errorf("Cannot create location. Already exist a location with name %s", lConfig.Name)))
			return
		}
		log.Panic(err)
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) updateLocation(w http.ResponseWriter, r *http.Request) {
	lConfig, err := getRequestedLocationConfig(r)
	if err != nil {
		writeError(w, r, newBadRequestError(err))
		return
	}

	err = s.locationMgr.UpdateLocation(lConfig)
	if err != nil {
		if locations.IsLocationNotFoundError(err) {
			writeError(w, r, newBadRequestError(errors.Errorf("Cannot update location %s as it does not exist", lConfig.Name)))
			return
		}
		log.Panic(err)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) deleteLocation(w http.ResponseWriter, r *http.Request) {

	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	locationName := params.ByName("locationName")
	err := s.locationMgr.RemoveLocation(locationName)
	if err != nil {
		if locations.IsLocationNotFoundError(err) {
			writeError(w, r, newBadRequestError(errors.Errorf("Cannot remove location %s as it does not exist", locationName)))
			return
		}
		log.Panic(err)
	}

	w.WriteHeader(http.StatusOK)
}

func getRequestedLocationConfig(r *http.Request) (locations.LocationConfiguration, error) {
	var lConfig locations.LocationConfiguration

	var params httprouter.Params
	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	locationName := params.ByName("locationName")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Panic(err)
		return lConfig, err
	}

	var locationRequest *LocationRequest
	err = json.Unmarshal(body, &locationRequest)
	if err != nil {
		return lConfig, err
	}

	lConfig.Name = locationName
	lConfig.Type = locationRequest.Type
	lConfig.Properties = locationRequest.Properties

	return lConfig, nil
}
