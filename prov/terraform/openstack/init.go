package openstack

import "novaforge.bull.com/starlings-janus/janus/registry"
import "novaforge.bull.com/starlings-janus/janus/prov/terraform"

func init() {
	reg := registry.GetRegistry()
	reg.RegisterDelegates([]string{`janus\.nodes\.openstack\..*`}, terraform.NewExecutor(&osGenerator{}), registry.BuiltinOrigin)
}
