package deployments

import (
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path/filepath"
	"time"
)

/**
This function allow you to store log in consul
*/
func LogInConsul(kv *api.KV, depId, message string) {
	if kv == nil || depId == "" {
		log.Panic("Can't use LogInConsul function without KV or deployment ID")
	}

	storeConsulKey(kv, filepath.Join(DeploymentKVPrefix, depId, "logs", log.ENGINE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano)), []byte(message))
}

func storeConsulKey(kv *api.KV, key string, value []byte) {
	// PUT a new KV pair
	p := &api.KVPair{Key: key, Value: value}
	if _, err := kv.Put(p, nil); err != nil {
		log.Panic(err)
	}
}
