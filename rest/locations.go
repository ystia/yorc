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
	"path"

	"github.com/julienschmidt/httprouter"

	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/locations"
	"github.com/ystia/yorc/v4/locations/adapter"
	"github.com/ystia/yorc/v4/log"
)

func (s *Server) listLocationsHandler(w http.ResponseWriter, r *http.Request) {
	locationConfigurations, err := s.locationMgr.GetLocations()
	if err != nil {
		log.Panic(err)
	}

	locs := make([]AtomLink, len(locationConfigurations))
	for i, locationConfiguration := range locationConfigurations {
		locs[i] = newAtomLink(LinkRelLocation, path.Join("/locations", locationConfiguration.Name))
	}
	if len(locs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	encodeJSONResponse(w, r, LocationCollection{Locations: locs})
}

func (s *Server) getLocationHandler(w http.ResponseWriter, r *http.Request) {
	var params httprouter.Params
	var locConfig *LocationConfiguration

	ctx := r.Context()
	params = ctx.Value(paramsLookupKey).(httprouter.Params)
	locationName := params.ByName("locationName")

	locationConfigurations, err := s.locationMgr.GetLocations()
	if err != nil {
		log.Panic(err)
	}

	for _, locationConfiguration := range locationConfigurations {
		if locationConfiguration.Name == locationName {
			locConfig = &LocationConfiguration{
				Name:       locationConfiguration.Name,
				Type:       locationConfiguration.Type,
				Properties: locationConfiguration.Properties,
			}
			break
		}
	}
	if locConfig == nil {
		writeError(w, r, newBadRequestError(errors.Errorf("Cannot get location with name %s as it does not exist", locationName)))
		return
	}
	encodeJSONResponse(w, r, *locConfig)

}

func (s *Server) createLocationHandler(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) updateLocationHandler(w http.ResponseWriter, r *http.Request) {
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
		if adapter.IsBadRequestError(err) {
			writeError(w, r, newBadRequestError(errors.Errorf("Cannot update location %s as its not correctly defined", lConfig.Name)))
			return

		}
		log.Panic(err)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) deleteLocationHandler(w http.ResponseWriter, r *http.Request) {

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
