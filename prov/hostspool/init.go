package hostspool

import "github.com/ystia/yorc/registry"

func init() {
	reg := registry.GetRegistry()
	reg.RegisterDelegates([]string{`yorc\.nodes\.hostspool\..*`}, &defaultExecutor{}, registry.BuiltinOrigin)
}
