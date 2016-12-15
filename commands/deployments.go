package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"novaforge.bull.com/starlings-janus/janus/rest"
)

func init() {
	RootCmd.AddCommand(deploymentsCmd)
	setDeploymentsConfig()
}

var noColor bool

var deploymentsCmd = &cobra.Command{
	Use:           "deployments",
	Aliases:       []string{"depls", "depl", "deps", "dep", "d"},
	Short:         "Perform commands on deployments",
	Long:          `Perform different commands on deployments`,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func setDeploymentsConfig() {

	deploymentsCmd.PersistentFlags().StringP("janus-api", "j", "localhost:8800", "specify the host and port used to join the Janus' REST API")
	deploymentsCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable coloring output")

	viper.BindPFlag("janus_api", deploymentsCmd.PersistentFlags().Lookup("janus-api"))
	viper.SetEnvPrefix("janus")
	viper.BindEnv("janus_api", "JANUS_API")
	viper.SetDefault("janus_api", "localhost:8800")

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

func printRestErrors(errs rest.Errors) {
	if len(errs.Errors) > 0 {
		fmt.Println("Got errors when interacting with Janus:")
	}
	for _, e := range errs.Errors {
		fmt.Printf("Error: %q: %q\n", e.Title, e.Detail)
	}
}

func errExit(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

func printErrors(body io.ReadCloser) {
	printRestErrors(getRestErrors(body))
}

func getRestErrors(body io.ReadCloser) rest.Errors {
	var errs rest.Errors
	bodyContent, _ := ioutil.ReadAll(body)
	json.Unmarshal(bodyContent, &errs)
	return errs
}

func getJSONEntityFromAtomGetRequest(janusAPI string, atomLink rest.AtomLink, entity interface{}) error {
	request, err := http.NewRequest("GET", "http://"+janusAPI+atomLink.Href, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to contact Janus API")
	}
	request.Header.Add("Accept", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return errors.Wrap(err, "Failed to contact Janus API")
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
