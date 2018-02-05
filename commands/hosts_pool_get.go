package commands

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/helper/tabutil"
	"novaforge.bull.com/starlings-janus/janus/rest"
)

func init() {
	var getCmd = &cobra.Command{
		Use:   "get <hostname>",
		Short: "Get host pool",
		Long:  `Gets the description of an host of the hosts pool managed by this Janus cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			colorize := !noColor
			if len(args) != 1 {
				return errors.Errorf("Expecting a hostname (got %d parameters)", len(args))
			}
			client, err := getClient()
			if err != nil {
				errExit(err)
			}

			request, err := client.NewRequest("GET", "/hosts_pool/"+args[0], nil)
			request.Header.Add("Accept", "application/json")
			if err != nil {
				errExit(err)
			}

			response, err := client.Do(request)
			defer response.Body.Close()
			if err != nil {
				errExit(err)
			}

			handleHTTPStatusCode(response, args[0], "host pool", http.StatusOK)
			var host rest.Host
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				errExit(err)
			}
			err = json.Unmarshal(body, &host)
			if err != nil {
				errExit(err)
			}

			hostsTable := tabutil.NewTable()
			hostsTable.AddHeaders("Name", "Connection", "Status", "Tags")
			var tagsList string
			for k, v := range host.Tags {
				if tagsList != "" {
					tagsList += ", "
				}
				tagsList += fmt.Sprintf("%s:%s", k, v)
			}

			hostsTable.AddRow(host.Name, host.Connection.String(), getColoredHostStatus(colorize, host.Status.String()), tagsList)

			if colorize {
				defer color.Unset()
			}
			fmt.Println("Host pool:")
			fmt.Println(hostsTable.Render())
			return nil
		},
	}
	hostsPoolCmd.AddCommand(getCmd)
}
