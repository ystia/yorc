package hostspool

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/ystia/yorc/commands/httputil"
)

func init() {
	var delCmd = &cobra.Command{
		Use:   "delete <hostname> [hostname...]",
		Short: "Delete hosts from hosts pool",
		Long:  `Delete hosts of the hosts pool managed by this Yorc cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.Errorf("Expecting a hostname (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient()
			if err != nil {
				httputil.ErrExit(err)
			}
			for i := range args {
				request, err := client.NewRequest("DELETE", "/hosts_pool/"+args[i], nil)
				if err != nil {
					httputil.ErrExit(err)
				}

				response, err := client.Do(request)
				defer response.Body.Close()
				if err != nil {
					httputil.ErrExit(err)
				}

				httputil.HandleHTTPStatusCode(response, args[0], "host pool", http.StatusOK)
			}
			return nil
		},
	}
	hostsPoolCmd.AddCommand(delCmd)
}
