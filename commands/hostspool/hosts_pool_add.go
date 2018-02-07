package hostspool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/commands/httputil"
	"novaforge.bull.com/starlings-janus/janus/prov/hostspool"
	"novaforge.bull.com/starlings-janus/janus/rest"
)

func init() {
	var jsonParam string
	var privateKey string
	var password string

	var addCmd = &cobra.Command{
		Use:   "add <hostname>",
		Short: "Add host pool",
		Long:  `Adds a host to the hosts pool managed by this Janus cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a hostname (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient()
			if err != nil {
				httputil.ErrExit(err)
			}
			if len(jsonParam) == 0 && len(privateKey) == 0 && len(password) == 0 {
				return errors.Errorf("You need to provide either JSON with connection information or private key or password for the host pool")
			}
			if len(jsonParam) == 0 {
				var hostRequest rest.HostRequest
				hostRequest.Connection = &hostspool.Connection{
					Password:   password,
					PrivateKey: privateKey,
				}

				tmp, err := json.Marshal(hostRequest)
				if err != nil {
					log.Panic(err)
				}

				jsonParam = string(tmp)
			}

			request, err := client.NewRequest("PUT", "/hosts_pool/"+args[0], bytes.NewBuffer([]byte(jsonParam)))
			if err != nil {
				httputil.ErrExit(err)
			}
			request.Header.Add("Content-Type", "application/json")

			response, err := client.Do(request)
			defer response.Body.Close()
			if err != nil {
				httputil.ErrExit(err)
			}

			httputil.HandleHTTPStatusCode(response, args[0], "host pool", http.StatusCreated)
			fmt.Println("Command submitted. path :", response.Header.Get("Location"))
			return nil
		},
	}
	addCmd.Flags().StringVarP(&jsonParam, "data", "d", "", "Need to provide the JSON format of the host pool")
	addCmd.Flags().StringVarP(&privateKey, "key", "k", "", "Need to provide a private key or a password for the host pool")
	addCmd.Flags().StringVarP(&password, "pwd", "p", "", "Need to provide a private key or a password for the host pool")

	hostsPoolCmd.AddCommand(addCmd)
}
