package hostspool

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"novaforge.bull.com/starlings-janus/janus/commands/httputil"
	"novaforge.bull.com/starlings-janus/janus/prov/hostspool"
	"novaforge.bull.com/starlings-janus/janus/rest"
)

func init() {
	var jsonParam string
	var privateKey string
	var password string
	var user string
	var host string
	var port uint64
	var labelsAdd []string
	var labelsRemove []string

	var updCmd = &cobra.Command{
		Use:   "update <hostname>",
		Short: "Update host pool",
		Long:  `Update labels list or connection of a host of the hosts pool managed by this Janus cluster.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a hostname (got %d parameters)", len(args))
			}
			client, err := httputil.GetClient()
			if err != nil {
				httputil.ErrExit(err)
			}
			if len(jsonParam) == 0 {
				var hostRequest rest.HostRequest
				hostRequest.Connection = &hostspool.Connection{
					User:       user,
					Host:       host,
					Port:       port,
					Password:   password,
					PrivateKey: privateKey,
				}
				for _, l := range labelsAdd {
					parts := strings.SplitN(l, "=", 2)
					me := rest.MapEntry{Op: rest.MapEntryOperationAdd, Name: parts[0]}
					if len(parts) == 2 {
						me.Value = parts[1]
					}
					hostRequest.Labels = append(hostRequest.Labels, me)
				}
				for _, l := range labelsRemove {
					hostRequest.Labels = append(hostRequest.Labels, rest.MapEntry{Op: rest.MapEntryOperationRemove, Name: l})
				}
				tmp, err := json.Marshal(hostRequest)
				if err != nil {
					log.Panic(err)
				}

				jsonParam = string(tmp)
			}

			request, err := client.NewRequest("PATCH", "/hosts_pool/"+args[0], bytes.NewBuffer([]byte(jsonParam)))
			if err != nil {
				httputil.ErrExit(err)
			}
			request.Header.Add("Content-Type", "application/json")

			response, err := client.Do(request)
			defer response.Body.Close()
			if err != nil {
				httputil.ErrExit(err)
			}

			httputil.HandleHTTPStatusCode(response, args[0], "host pool", http.StatusOK)
			return nil
		},
	}
	updCmd.Flags().StringVarP(&jsonParam, "data", "d", "", "Need to provide the JSON format of the updated host pool")
	updCmd.Flags().StringVarP(&user, "user", "", "root", "User used to connect to the host")
	updCmd.Flags().StringVarP(&host, "host", "", "", "Hostname or ip address used to connect to the host. (defaults to the hostname in the hosts pool)")
	updCmd.Flags().Uint64VarP(&port, "port", "", 22, "Port used to connect to the host. (defaults to the hostname in the hosts pool)")
	updCmd.Flags().StringVarP(&privateKey, "key", "k", "", "Need to provide a private key or a password for the host pool")
	updCmd.Flags().StringVarP(&password, "password", "p", "", "Need to provide a private key or a password for the host pool")
	updCmd.Flags().StringSliceVarP(&labelsAdd, "add-label", "", nil, "Add a label in form 'key=value' to the host. May be specified several time.")
	updCmd.Flags().StringSliceVarP(&labelsRemove, "remove-label", "", nil, "Remove a label from the host. May be specified several time.")

	hostsPoolCmd.AddCommand(updCmd)
}
