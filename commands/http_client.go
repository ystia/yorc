package commands

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/url"

	"strings"

	"crypto/x509"
	"io/ioutil"

	"github.com/goware/urlx"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type janusClient struct {
	*http.Client
	baseUrl string
}

func (c *janusClient) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(method, c.baseUrl+path, body)
}

func (c *janusClient) Get(path string) (*http.Response, error) {
	return c.Client.Get(c.baseUrl + path)
}

func (c *janusClient) Head(path string) (*http.Response, error) {
	return c.Client.Head(c.baseUrl + path)
}
func (c *janusClient) Post(path string, contentType string, body io.Reader) (*http.Response, error) {
	return c.Client.Post(c.baseUrl+path, contentType, body)
}

func (c *janusClient) PostForm(path string, data url.Values) (*http.Response, error) {
	return c.Client.PostForm(c.baseUrl+path, data)
}

func getClient() (*janusClient, error) {
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
		return &janusClient{
			baseUrl: "https://" + janusAPI,
			Client:  &http.Client{Transport: tr},
		}, nil
	}

	return &janusClient{
		baseUrl: "http://" + janusAPI,
		Client:  &http.Client{},
	}, nil

}
