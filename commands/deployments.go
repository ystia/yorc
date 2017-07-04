package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
		err := cmd.Help()
		if err != nil {
			fmt.Print(err)
		}
	},
}

func setDeploymentsConfig() {

	deploymentsCmd.PersistentFlags().StringP("janus-api", "j", "localhost:8800", "specify the host and port used to join the Janus' REST API")
	deploymentsCmd.PersistentFlags().StringP("ca-file", "", "", "This provides a file path to a PEM-encoded certificate authority. This implies the use of HTTPS to connect to the Janus REST API.")
	deploymentsCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable coloring output")
	deploymentsCmd.PersistentFlags().BoolP("secured", "s", false, "Use HTTPS to connect to the Janus REST API")
	deploymentsCmd.PersistentFlags().BoolP("skip-tls-verify", "", false, "skip-tls-verify controls whether a client verifies the server's certificate chain and host name. If set to true, TLS accepts any certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks. This should be used only for testing. This implies the use of HTTPS to connect to the Janus REST API.")

	viper.BindPFlag("janus_api", deploymentsCmd.PersistentFlags().Lookup("janus-api"))
	viper.BindPFlag("secured", deploymentsCmd.PersistentFlags().Lookup("secured"))
	viper.BindPFlag("ca_file", deploymentsCmd.PersistentFlags().Lookup("ca-file"))
	viper.BindPFlag("skip_tls_verify", deploymentsCmd.PersistentFlags().Lookup("skip-tls-verify"))
	viper.SetEnvPrefix("janus")
	viper.BindEnv("janus_api", "JANUS_API")
	viper.BindEnv("secured")
	viper.BindEnv("ca_file")
	viper.BindEnv("skip_tls_verify")
	viper.SetDefault("janus_api", "localhost:8800")
	viper.SetDefault("secured", false)
	viper.SetDefault("skip_tls_verify", false)

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

func okExit(msg interface{}) {
	fmt.Println(msg)
	os.Exit(0)
}

func printErrors(body io.Reader) {
	printRestErrors(getRestErrors(body))
}

func getRestErrors(body io.Reader) rest.Errors {
	var errs rest.Errors
	bodyContent, _ := ioutil.ReadAll(body)
	json.Unmarshal(bodyContent, &errs)
	return errs
}

func getJSONEntityFromAtomGetRequest(client *janusClient, atomLink rest.AtomLink, entity interface{}) error {
	request, err := client.NewRequest("GET", atomLink.Href, nil)
	if err != nil {
		return errors.Wrap(err, janusAPIDefaultErrorMsg)
	}
	request.Header.Add("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return errors.Wrap(err, janusAPIDefaultErrorMsg)
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
