package deployments

import (
	"fmt"
	"net/http"
	"path"

	"novaforge.bull.com/starlings-janus/janus/log"

	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"novaforge.bull.com/starlings-janus/janus/commands/httputil"
)

func init() {
	var shouldStreamLogs bool
	var shouldStreamEvents bool
	var nodeName string
	var instancesDelta int32
	var scaleCmd = &cobra.Command{
		Use:   "scale <id>",
		Short: "Scale a node",
		Long:  `Scale a given node of a deployment <id> by adding or removing the specified number of instances.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.Errorf("Expecting a deployment id (got %d parameters)", len(args))
			}

			if nodeName == "" {
				return errors.New("Missing mandatory \"node\" flag")
			}

			if instancesDelta == 0 {
				return errors.New("Missing non-zero \"delta\" flag")
			}

			client, err := httputil.GetClient()
			if err != nil {
				httputil.ErrExit(err)
			}
			deploymentID := args[0]

			location, err := postScalingRequest(client, deploymentID, nodeName, instancesDelta)
			if err != nil {
				return err
			}

			fmt.Println("Scaling request submitted. Task Id:", path.Base(location))
			if shouldStreamLogs && !shouldStreamEvents {
				StreamsLogs(client, deploymentID, !NoColor, false, false)
			} else if !shouldStreamLogs && shouldStreamEvents {
				StreamsEvents(client, deploymentID, !NoColor, false, false)
			} else if shouldStreamLogs && shouldStreamEvents {
				return errors.Errorf("You can't provide stream-events and stream-logs flags at same time")
			}
			return nil
		},
	}
	scaleCmd.PersistentFlags().StringVarP(&nodeName, "node", "n", "", "The name of the node that should be scaled.")
	scaleCmd.PersistentFlags().Int32VarP(&instancesDelta, "delta", "d", 0, "The non-zero number of instance to add (if > 0) or remove (if < 0).")
	scaleCmd.PersistentFlags().BoolVarP(&shouldStreamLogs, "stream-logs", "l", false, "Stream logs after issuing the scaling request. In this mode logs can't be filtered, to use this feature see the \"log\" command.")
	scaleCmd.PersistentFlags().BoolVarP(&shouldStreamEvents, "stream-events", "e", false, "Stream events after  issuing the scaling request.")
	DeploymentsCmd.AddCommand(scaleCmd)
}

func postScalingRequest(client *httputil.JanusClient, deploymentID, nodeName string, instancesDelta int32) (string, error) {
	request, err := client.NewRequest("POST", path.Join("/deployments", deploymentID, "scale", nodeName), nil)
	if err != nil {
		httputil.ErrExit(errors.Wrap(err, httputil.JanusAPIDefaultErrorMsg))
	}

	query := request.URL.Query()
	query.Set("delta", strconv.Itoa(int(instancesDelta)))

	request.URL.RawQuery = query.Encode()

	log.Debugf("POST: %s", request.URL.String())

	response, err := client.Do(request)
	defer response.Body.Close()
	if err != nil {
		httputil.ErrExit(errors.Wrap(err, httputil.JanusAPIDefaultErrorMsg))
	}

	ids := deploymentID + "/" + nodeName
	httputil.HandleHTTPStatusCode(response, ids, "deployment/node", http.StatusAccepted)
	location := response.Header.Get("Location")
	if location == "" {
		return "", errors.New("No \"Location\" header returned in Janus response")
	}
	return location, nil
}
