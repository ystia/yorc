package commands

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"net/http"
)

func init() {
	var jsonParam string

	var updCmd = &cobra.Command{
		Use:   "update <hostname>",
		Short: "Update host pool",
		Long:  `Update tags list or connection of an host of the hosts pool managed by this Janus cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a hostname (got %d parameters)", len(args))
			}
			client, err := getClient()
			if err != nil {
				errExit(err)
			}
			if len(jsonParam) == 0 {
				return errors.Errorf("You need to provide a JSON with updated information")
			}

			request, err := client.NewRequest("PATCH", "/hosts_pool/"+args[0], bytes.NewBuffer([]byte(jsonParam)))
			if err != nil {
				errExit(err)
			}
			request.Header.Add("Content-Type", "application/json")

			response, err := client.Do(request)
			defer response.Body.Close()
			if err != nil {
				errExit(err)
			}

			handleHTTPStatusCode(response, args[0], "host pool", http.StatusOK)
			return nil
		},
	}
	updCmd.Flags().StringVarP(&jsonParam, "data", "d", "", "Need to provide the JSON format of the updated host pool")

	hostsPoolCmd.AddCommand(updCmd)
}
