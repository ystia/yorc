// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package httputil

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/goware/urlx"
	"github.com/hashicorp/go-rootcerts"
	"github.com/pkg/errors"

	"github.com/ystia/yorc/v4/config"
	"github.com/ystia/yorc/v4/rest"
)

// YorcAPIDefaultErrorMsg is the default communication error message
const YorcAPIDefaultErrorMsg = "Failed to contact Yorc API"

// YorcClient is the Yorc HTTP client structure
type YorcClient struct {
	*http.Client
	baseURL string
}

// NewRequest returns a new HTTP request
func (c *YorcClient) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, c.baseURL+path, body)
}

// Get returns a new HTTP request with GET method
func (c *YorcClient) Get(path string) (*http.Response, error) {
	return c.Client.Get(c.baseURL + path)
}

// Head returns a new HTTP request with HEAD method
func (c *YorcClient) Head(path string) (*http.Response, error) {
	return c.Client.Head(c.baseURL + path)
}

// Post returns a new HTTP request with Post method
func (c *YorcClient) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return c.Client.Post(c.baseURL+path, contentType, body)
}

// PostForm returns a new HTTP request with Post method and form content
func (c *YorcClient) PostForm(path string, data url.Values) (*http.Response, error) {
	return c.Client.PostForm(c.baseURL+path, data)
}

// GetClient returns a yorc HTTP Client
func GetClient(cc config.Client) (*YorcClient, error) {
	yorcAPI := cc.YorcAPI
	yorcAPI = strings.TrimRight(yorcAPI, "/")
	caFile := cc.CAFile
	caPath := cc.CAPath
	certFile := cc.CertFile
	keyFile := cc.KeyFile
	if cc.SSLEnabled || cc.CAFile != "" || cc.CAPath != "" || (certFile != "" && keyFile != "") {
		url, err := urlx.Parse(yorcAPI)
		if err != nil {
			return nil, errors.Wrap(err, "Malformed Yorc URL")
		}
		yorcHost, _, err := urlx.SplitHostPort(url)
		if err != nil {
			return nil, errors.Wrap(err, "Malformed Yorc URL")
		}

		tlsConfig := &tls.Config{ServerName: yorcHost}
		if certFile != "" && keyFile != "" {
			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to load TLS certificates")
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		if caFile != "" || caPath != "" {
			cfg := &rootcerts.Config{
				CAFile: caFile,
				CAPath: caPath,
			}
			rootcerts.ConfigureTLS(tlsConfig, cfg)
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
			tlsConfig.BuildNameToCertificate()
		}
		if cc.SkipTLSVerify {
			tlsConfig.InsecureSkipVerify = true
			fmt.Println("Warning : usage of skip_tls_verify is not recommended for production and may expose to MITM attack")
		}

		tr := &http.Transport{
			TLSClientConfig: tlsConfig,
		}
		return &YorcClient{
			baseURL: "https://" + yorcAPI,
			Client:  &http.Client{Transport: tr},
		}, nil
	}

	return &YorcClient{
		baseURL: "http://" + yorcAPI,
		Client:  &http.Client{},
	}, nil

}

// HandleHTTPStatusCode handles Yorc HTTP status code and displays error if needed
func HandleHTTPStatusCode(response *http.Response, resourceID string, resourceType string, expectedStatusCodes ...int) {
	HandleHTTPStatusCodeWithCustomizedErrorMessage(
		response,
		resourceID,
		resourceType,
		nil, // no customized error message
		expectedStatusCodes...)
}

// HandleHTTPStatusCodeWithCustomizedErrorMessage handles Yorc HTTP status
// code and in case of error, can display a customized error message instead of
// the source error if this source error contains a given string provided as
// key in the map argument customizedErrorMessages
func HandleHTTPStatusCodeWithCustomizedErrorMessage(
	response *http.Response,
	resourceID string,
	resourceType string,
	customizedErrorMessages map[string]string,
	expectedStatusCodes ...int) {
	if len(expectedStatusCodes) == 0 {
		panic("expected status code parameter is required")
	}
	if !isExpected(response.StatusCode, expectedStatusCodes) {
		switch response.StatusCode {
		// This case is not an error so the exit code is OK
		case http.StatusNotFound:
			okExit(fmt.Sprintf("The %s with the following id %q doesn't exist", resourceType, resourceID))
		case http.StatusNoContent:
			// same point as above
			okExit(fmt.Sprintf("No %s", resourceType))
		default:
			errs := getRestErrors(response.Body)
			errMsg := getCustomizedErrorMessage(errs, customizedErrorMessages)
			if errMsg == "" {
				// No error message to customize
				printRestErrors(errs)
				errMsg = fmt.Sprintf("Expecting HTTP Status code in %d but got %d, reason %q",
					expectedStatusCodes, response.StatusCode, response.Status)
			}
			ErrExit(errors.Errorf(errMsg))
		}
	}
}

func getCustomizedErrorMessage(
	errs rest.Errors,
	customizedErrorMessages map[string]string) string {

	if customizedErrorMessages != nil {
		for _, e := range errs.Errors {
			for key, value := range customizedErrorMessages {
				if strings.Contains(e.Detail, key) {
					return value
				}
			}
		}
	}
	return ""
}

type cmdRestError struct {
	errs rest.Errors
}

func (cre cmdRestError) Error() string {
	var buf bytes.Buffer
	if len(cre.errs.Errors) > 0 {
		buf.WriteString("Got errors when interacting with Yorc:\n")
		for _, e := range cre.errs.Errors {
			buf.WriteString(fmt.Sprintf("Error: %q: %q\n", e.Title, e.Detail))
		}
	}
	return buf.String()
}

// ErrExit allows to exit on error with exit code 1 after printing error message
func ErrExit(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

// GetJSONEntityFromAtomGetRequest returns JSON entity from AtomLink request
func GetJSONEntityFromAtomGetRequest(client *YorcClient, atomLink rest.AtomLink, entity interface{}) error {
	request, err := client.NewRequest("GET", atomLink.Href, nil)
	if err != nil {
		return errors.Wrap(err, YorcAPIDefaultErrorMsg)
	}
	request.Header.Add("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return errors.Wrap(err, YorcAPIDefaultErrorMsg)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// Try to get the reason
		errs := getRestErrors(response.Body)
		err = cmdRestError{errs: errs}
		return errors.Wrapf(err, "Expecting HTTP Status code 2xx got %d, reason %q: ", response.StatusCode, response.Status)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errors.Wrap(err, "Failed to read response from Yorc")
	}
	return errors.Wrap(json.Unmarshal(body, entity), "Fail to parse JSON response from Yorc")
}

// okExit allows to exit successfully after printing a message
func okExit(msg interface{}) {
	fmt.Println(msg)
	os.Exit(0)
}

// PrintErrors allows to print REST errors
func PrintErrors(body io.Reader) {
	printRestErrors(getRestErrors(body))
}

func getRestErrors(body io.Reader) rest.Errors {
	var errs rest.Errors
	bodyContent, _ := ioutil.ReadAll(body)
	json.Unmarshal(bodyContent, &errs)
	return errs
}

func printRestErrors(errs rest.Errors) {
	if len(errs.Errors) > 0 {
		fmt.Println("Got errors when interacting with Yorc:")
	}
	for _, e := range errs.Errors {
		fmt.Printf("Error: %q: %q\n", e.Title, e.Detail)
	}
}

func isExpected(got int, expected []int) bool {
	for _, code := range expected {
		if got == code {
			return true
		}
	}
	return false
}
