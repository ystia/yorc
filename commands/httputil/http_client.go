package httputil

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"

	"strings"

	"crypto/x509"
	"io/ioutil"

	"fmt"

	"bytes"
	"encoding/json"
	"github.com/goware/urlx"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"os"
)

// JanusAPIDefaultErrorMsg is the default communication error message
const JanusAPIDefaultErrorMsg = "Failed to contact Janus API"

// JanusClient is the Janus HTTP client structure
type JanusClient struct {
	*http.Client
	baseURL string
}

// NewRequest returns a new HTTP request
func (c *JanusClient) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, c.baseURL+path, body)
}

// Get returns a new HTTP request with GET method
func (c *JanusClient) Get(path string) (*http.Response, error) {
	return c.Client.Get(c.baseURL + path)
}

// Head returns a new HTTP request with HEAD method
func (c *JanusClient) Head(path string) (*http.Response, error) {
	return c.Client.Head(c.baseURL + path)
}

// Post returns a new HTTP request with Post method
func (c *JanusClient) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return c.Client.Post(c.baseURL+path, contentType, body)
}

// PostForm returns a new HTTP request with Post method and form content
func (c *JanusClient) PostForm(path string, data url.Values) (*http.Response, error) {
	return c.Client.PostForm(c.baseURL+path, data)
}

// GetClient returns a janus HTTP Client
func GetClient() (*JanusClient, error) {
	tlsEnable := viper.GetBool("secured")
	janusAPI := viper.GetString("janus_api")
	janusAPI = strings.TrimRight(janusAPI, "/")
	caFile := viper.GetString("ca_file")
	skipTLSVerify := viper.GetBool("skip_tls_verify")
	if tlsEnable || skipTLSVerify || caFile != "" {
		url, err := urlx.Parse(janusAPI)
		if err != nil {
			return nil, errors.Wrap(err, "Malformed Janus URL")
		}
		janusHost, _, err := urlx.SplitHostPort(url)
		if err != nil {
			return nil, errors.Wrap(err, "Malformed Janus URL")
		}
		tlsConfig := &tls.Config{ServerName: janusHost}
		if caFile != "" {
			certPool := x509.NewCertPool()
			caCert, err := ioutil.ReadFile(caFile)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to read certificate authority file")
			}
			if !certPool.AppendCertsFromPEM(caCert) {
				return nil, errors.Errorf("%q is not a valid certificate authority.", caFile)
			}
			tlsConfig.RootCAs = certPool
		}
		tlsConfig.InsecureSkipVerify = skipTLSVerify
		tr := &http.Transport{
			TLSClientConfig: tlsConfig,
		}
		return &JanusClient{
			baseURL: "https://" + janusAPI,
			Client:  &http.Client{Transport: tr},
		}, nil
	}

	return &JanusClient{
		baseURL: "http://" + janusAPI,
		Client:  &http.Client{},
	}, nil

}

// HandleHTTPStatusCode handles Janus HTTP status code and displays error if needed
func HandleHTTPStatusCode(response *http.Response, resourceID string, resourceType string, expectedStatusCodes ...int) {
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
			PrintErrors(response.Body)
			ErrExit(errors.Errorf("Expecting HTTP Status code in %d but got %d, reason %q", expectedStatusCodes, response.StatusCode, response.Status))
		}
	}
}

type cmdRestError struct {
	errs rest.Errors
}

func (cre cmdRestError) Error() string {
	var buf bytes.Buffer
	if len(cre.errs.Errors) > 0 {
		buf.WriteString("Got errors when interacting with Janus:\n")
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
func GetJSONEntityFromAtomGetRequest(client *JanusClient, atomLink rest.AtomLink, entity interface{}) error {
	request, err := client.NewRequest("GET", atomLink.Href, nil)
	if err != nil {
		return errors.Wrap(err, JanusAPIDefaultErrorMsg)
	}
	request.Header.Add("Accept", "application/json")

	response, err := client.Do(request)
	defer response.Body.Close()
	if err != nil {
		return errors.Wrap(err, JanusAPIDefaultErrorMsg)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		// Try to get the reason
		errs := getRestErrors(response.Body)
		err = cmdRestError{errs: errs}
		return errors.Wrapf(err, "Expecting HTTP Status code 2xx got %d, reason %q: ", response.StatusCode, response.Status)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errors.Wrap(err, "Failed to read response from Janus")
	}
	return errors.Wrap(json.Unmarshal(body, entity), "Fail to parse JSON response from Janus")
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
		fmt.Println("Got errors when interacting with Janus:")
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
