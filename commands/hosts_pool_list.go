package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"net/http"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"novaforge.bull.com/starlings-janus/janus/helper/tabutil"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"strings"
)

func init() {
	hostsPoolCmd.AddCommand(hpListCmd)
}

var hpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List hosts pools",
	Long:  `Lists hosts of the hosts pool managed by this Janus cluster.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		colorize := !noColor
		client, err := getClient()
		if err != nil {
			errExit(err)
		}
		request, err := client.NewRequest("GET", "/hosts_pool", nil)
		if err != nil {
			errExit(err)
		}
		request.Header.Add("Accept", "application/json")
		response, err := client.Do(request)
		defer response.Body.Close()
		if err != nil {
			errExit(err)
		}
		handleHTTPStatusCode(response, "", "host pool", http.StatusOK)
		var hostsColl rest.HostsCollection
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			errExit(err)
		}
		err = json.Unmarshal(body, &hostsColl)
		if err != nil {
			errExit(err)
		}

		hostsTable := tabutil.NewTable()
		hostsTable.AddHeaders("Name", "Connection", "Status", "Tags")
		for _, hostLink := range hostsColl.Hosts {
			if hostLink.Rel == rest.LinkRelHost {
				var host rest.Host
				var tagsList string

				err = getJSONEntityFromAtomGetRequest(client, hostLink, &host)
				if err != nil {
					errExit(err)
				}

				for k, v := range host.Tags {
					if tagsList != "" {
						tagsList += ", "
					}
					tagsList += fmt.Sprintf("%s:%s", k, v)
				}

				hostsTable.AddRow(host.Name, host.Connection.String(), getColoredHostStatus(colorize, host.Status.String()), tagsList)
			}
		}
		if colorize {
			defer color.Unset()
		}
		fmt.Println("Host pools:")
		fmt.Println(hostsTable.Render())
		return nil
	},
}

func getColoredHostStatus(colorize bool, status string) string {
	if !colorize {
		return status
	}
	switch {
	case strings.ToLower(status) == "free":
		return color.New(color.FgHiGreen, color.Bold).SprintFunc()(status)
	case strings.ToLower(status) == "allocated":
		return color.New(color.FgHiRed, color.Bold).SprintFunc()(status)
	default:
		return color.New(color.FgHiYellow, color.Bold).SprintFunc()(status)
	}
}
