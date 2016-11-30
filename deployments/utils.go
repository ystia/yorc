package deployments

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path"
	"time"
)

// LogInConsul stores a log message in consul
func LogInConsul(kv *api.KV, depId, message string) {
	if kv == nil || depId == "" {
		log.Panic("Can't use LogInConsul function without KV or deployment ID")
	}
	err := consulutil.StoreConsulKeyAsString(path.Join(consulutil.DeploymentKVPrefix, depId, "logs", ENGINE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano)), message)
	if err != nil {
		log.Printf("Failed to publish log in consul for deployment %q: %+v", depId, err)
	}
}

// LogErrorInConsul stores an error message in consul.
//
// Basically it's a shortcut for:
//	LogInConsul(kv, depId, fmt.Sprintf("%v", err))
func LogErrorInConsul(kv *api.KV, depId string, err error) {
	LogInConsul(kv, depId, fmt.Sprintf("%v", err))
}
