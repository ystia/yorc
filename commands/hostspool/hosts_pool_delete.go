package hostspool

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/commands/httputil"
)

func init() {
	var delCmd = &cobra.Command{
		Use:   "delete <hostname>",
		Short: "Delete host pool",
		Long:  `Delete a host of the hosts pool managed by this Janus cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a hostname (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient()
			if err != nil {
				httputil.ErrExit(err)
			}

			request, err := client.NewRequest("DELETE", "/hosts_pool/"+args[0], nil)
			if err != nil {
				httputil.ErrExit(err)
			}

			response, err := client.Do(request)
			defer response.Body.Close()
			if err != nil {
				httputil.ErrExit(err)
			}

			httputil.HandleHTTPStatusCode(response, args[0], "host pool", http.StatusOK)
			return nil
		},
	}
	hostsPoolCmd.AddCommand(delCmd)
}
