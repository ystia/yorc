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

package maas

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/juju/gomaasapi"
	"github.com/pkg/errors"
	"github.com/ystia/yorc/v4/config"
)

var apiVersion string = "2.0"

type deployResults struct {
	ips       []string
	system_id string
}

func getMaasClient(locationProps config.DynamicMap) (*gomaasapi.MAASObject, error) {
	// Check manadatory maas configuration
	if err := checkLocationConfig(locationProps); err != nil {
		log.Printf("Unable to provide maas client due to:%+v", err)
		return nil, err
	}

	apiURL := locationProps.GetString("api_url")
	apiKey := locationProps.GetString("api_key")

	authClient, err := gomaasapi.NewAuthenticatedClient(gomaasapi.AddAPIVersionToURL(apiURL, apiVersion), apiKey)
	if err != nil {
		return nil, err
	}

	maas := gomaasapi.NewMAAS(*authClient)

	return maas, nil
}

func checkLocationConfig(locationProps config.DynamicMap) error {
	if strings.Trim(locationProps.GetString("api_url"), "") == "" {
		return errors.New("maas location ulr is not set")
	}

	if strings.Trim(locationProps.GetString("api_key"), "") == "" {
		return errors.New("maas location api key is not set")
	}

	// if strings.Trim(locationProps.GetString("user_token"), "") == "" {
	// 	return errors.New("maas location user token is not set")
	// }
	return nil
}

// func getMachines(maas *gomaasapi.MAASObject) []byte {
// 	machineListing := maas.GetSubObject("machines")
// 	machines, err := machineListing.CallGet("", url.Values{})
// 	json, err := machines.MarshalJSON()
// 	return json
// }

func allocateMachine(maas *gomaasapi.MAASObject) (string, error) {
	machineListing := maas.GetSubObject("machines")
	machineJsonObj, err := machineListing.CallPost("allocate", url.Values{})
	if err != nil {
		return "", err
	}

	machineMaasObj, err := machineJsonObj.GetMAASObject()
	if err != nil {
		return "", err
	}

	system_id, err := machineMaasObj.GetField("system_id")
	if err != nil {
		return "", err
	}

	return system_id, nil
}

func deploy(maas *gomaasapi.MAASObject, system_id string) (*gomaasapi.JSONObject, error) {
	machineListing := maas.GetSubObject("machines/" + system_id)
	machineJsonObj, err := machineListing.CallPost("deploy", url.Values{})

	if err != nil {
		return nil, err
	}

	return &machineJsonObj, nil
}

func release(maas *gomaasapi.MAASObject, systemId string) error {
	machineListing := maas.GetSubObject("machines/" + systemId)
	_, err := machineListing.CallPost("release", url.Values{"quick_erase": {"true"}})
	if err != nil {
		return err
	}

	return nil
}

func getMachineInfo(maas *gomaasapi.MAASObject, systemId string) (*gomaasapi.JSONObject, error) {
	machineListing := maas.GetSubObject("machines/" + systemId)
	jsonObj, err := machineListing.CallGet("", url.Values{})
	if err != nil {
		return nil, err
	}
	return &jsonObj, nil
}

func allocateAndDeploy(maas *gomaasapi.MAASObject) (*deployResults, error) {
	// log.Println("Allocating machine")
	system_id, err := allocateMachine(maas)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate machine: %w", err)
	}

	// log.Println("Deploying machine with system_id : " + system_id)
	_, err = deploy(maas, system_id)
	if err != nil {
		return nil, err
	}

	// log.Println("Allocating and deployement of " + system_id + " successfull")
	// Wait for the node to finish deploying
	var deployRes gomaasapi.MAASObject
	for {
		time.Sleep(20 * time.Second)
		// log.Println("Checking if " + system_id + " has finished deploying...")

		json, err := getMachineInfo(maas, system_id)
		if err != nil {
			return nil, err
		}
		deployRes, err = json.GetMAASObject()
		if err != nil {
			return nil, err
		}
		status, err := deployRes.GetField("status_name")
		if err != nil {
			return nil, err
		}
		if status != "Deploying" {
			break
		}
	}

	// Get machines ips
	test := deployRes.GetMap()
	ipaddrObj := test["ip_addresses"]

	ipsAddr, err := ipaddrObj.GetArray()
	if err != nil {
		return nil, err
	}

	var ips []string
	for _, ipObj := range ipsAddr {
		ipString, err := ipObj.GetString()
		if err != nil {
			return nil, errors.Wrapf(err, "Failed getting ips from machines with system_id : %s", system_id)
		}
		ips = append(ips, ipString)
	}

	return &deployResults{
		ips:       ips,
		system_id: system_id,
	}, nil
}
