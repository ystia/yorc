package deployments

import (
	"fmt"
	"path"
	"time"

	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
)

// LogInConsul stores a log message in consul
func LogInConsul(kv *api.KV, deploymentID, message string) {
	if kv == nil || deploymentID == "" {
		log.Panic("Can't use LogInConsul function without KV or deployment ID")
	}
	err := consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, deploymentID, "logs", ENGINE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano)), message)
	if err != nil {
		log.Printf("Failed to publish log in consul for deployment %q: %+v", deploymentID, err)
	}
}

// LogErrorInConsul stores an error message in consul.
//
// Basically it's a shortcut for:
//	LogInConsul(kv, deploymentID, fmt.Sprintf("%v", err))
func LogErrorInConsul(kv *api.KV, deploymentID string, err error) {
	LogInConsul(kv, deploymentID, fmt.Sprintf("%v", err))
}
