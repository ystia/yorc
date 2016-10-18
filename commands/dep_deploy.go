package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"novaforge.bull.com/starlings-janus/janus/helper/ziputil"
	"novaforge.bull.com/starlings-janus/janus/rest"
	"os"
	"path"
	"path/filepath"
)

func init() {
	deploymentsCmd.AddCommand(deployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy <csar_path>",
	Short: "Deploy a CSAR",
	Long: `Deploy a file or directory pointed by <csar_path>
If <csar_path> point to a valid zip archive it is submited to Janus as it.
If <csar_path> point to a file or directory it is zipped before beeing submited to Janus.
If <csar_path> point to a single file it should be TOSCA YAML description.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("Expecting a path to a file or directory (got %d parameters)", len(args))
		}
		janusApi := viper.GetString("janus_api")
		absPath, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		fileInfo, err := os.Stat(absPath)
		var location string = ""
		if !fileInfo.IsDir() {
			file, err := os.Open(absPath)
			if err != nil {
				errExit(err)
			}

			defer file.Close()

			buff, err := ioutil.ReadAll(file)
			if err != nil {
				errExit(err)
			}
			fileType := http.DetectContentType(buff)
			if fileType == "application/zip" {
				location, err = postCSAR(buff, janusApi)
				if err != nil {
					errExit(err)
				}
			}
		}

		if location == "" {
			csarZip, err := ziputil.ZipPath(absPath)
			if err != nil {
				errExit(err)
			}
			location, err = postCSAR(csarZip, janusApi)

			if err != nil {
				errExit(err)
			}
		}

		fmt.Println("Deployment submited. Deployment Id:", path.Base(location))
		return nil
	},
}

func postCSAR(csarZip []byte, janusApi string) (string, error) {
	r, err := http.Post("http://"+janusApi+"/deployments", "application/zip", bytes.NewReader(csarZip))
	if err != nil {
		return "", err
	}
	if r.StatusCode != 201 {
		// Try to get the reason
		var errs rest.Errors
		body, _ := ioutil.ReadAll(r.Body)
		err = json.Unmarshal(body, &errs)
		if err == nil {
			printRestErrors(errs)
		}
		return "", fmt.Errorf("POST failed: Expecting HTTP Status code 201 got %d, reason %q", r.StatusCode, r.Status)
	}
	if location := r.Header.Get("Location"); location != "" {
		return location, nil
	}
	return "", errors.New("No \"Location\" header returned in Janus response")
}
