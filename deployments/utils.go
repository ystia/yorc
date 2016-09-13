package deployments

import (
	"github.com/hashicorp/consul/api"
	"novaforge.bull.com/starlings-janus/janus/log"
	"path/filepath"
	"time"
)

type DeploymentLogSender struct {
	kv    *api.KV
	depId string
}

func NewDeploymentLogSender(api *api.KV, depId string) *DeploymentLogSender {
	return &DeploymentLogSender{
		kv:    api,
		depId: depId,
	}
}

func (d *DeploymentLogSender) SetDeploymentId(depId string) {
	d.depId = depId
}

/**
This function allow you to store log in consul
*/
func (d *DeploymentLogSender) LogInConsul(message string) {
	storeConsulKey(d.kv, filepath.Join(DeploymentKVPrefix, d.depId, "logs", log.ENGINE_LOG_PREFIX+"__"+time.Now().Format(time.RFC3339Nano)), []byte(message))
}

func storeConsulKey(kv *api.KV, key string, value []byte) {
	// PUT a new KV pair
	p := &api.KVPair{Key: key, Value: value}
	if _, err := kv.Put(p, nil); err != nil {
		log.Panic(err)
	}
}
