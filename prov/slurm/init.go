package slurm

import "novaforge.bull.com/starlings-janus/janus/registry"

func init() {
	reg := registry.GetRegistry()
	reg.RegisterDelegates([]string{`janus\.nodes\.slurm\..*`}, newExecutor(&slurmGenerator{}), registry.BuiltinOrigin)
	reg.RegisterResourcesProvider("slurm", newResourcesProvider(), registry.BuiltinOrigin)
}
