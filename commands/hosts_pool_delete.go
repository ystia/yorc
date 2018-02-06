package commands

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"net/http"
)

func init() {
	var delCmd = &cobra.Command{
		Use:   "delete <hostname>",
		Short: "Delete host pool",
		Long:  `Delete tags list or connection of a host of the hosts pool managed by this Janus cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a hostname (got %d parameters)", len(args))
			}
			client, err := getClient()
			if err != nil {
				errExit(err)
			}

			request, err := client.NewRequest("DELETE", "/hosts_pool/"+args[0], nil)
			if err != nil {
				errExit(err)
			}

			response, err := client.Do(request)
			defer response.Body.Close()
			if err != nil {
				errExit(err)
			}

			handleHTTPStatusCode(response, args[0], "host pool", http.StatusOK)
			return nil
		},
	}
	hostsPoolCmd.AddCommand(delCmd)
}
