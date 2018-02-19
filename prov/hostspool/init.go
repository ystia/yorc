package hostspool

import "novaforge.bull.com/starlings-janus/janus/registry"

func init() {
	reg := registry.GetRegistry()
	reg.RegisterDelegates([]string{`janus\.nodes\.hostspool\..*`}, &defaultExecutor{}, registry.BuiltinOrigin)
}
