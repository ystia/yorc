package deployments

import (
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/helper/consulutil"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path"
	"time"
)

/**
This function allow you to store log in consul
*/
func LogInConsul(kv *api.KV, depId, message string) {
	if kv == nil || depId == "" {
		log.Panic("Can't use LogInConsul function without KV or deployment ID")
	}
	err := consulutil.StoreConsulKeyAsString(path.Join(DeploymentKVPrefix, depId, "logs", log.ENGINE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano)), message)
	if err != nil {
		log.Printf("Failed to publish log in consul for deployment %q: %+v", depId, err)
	}
}
