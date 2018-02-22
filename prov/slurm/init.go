package slurm

import "github.com/ystia/yorc/registry"

func init() {
	reg := registry.GetRegistry()
	reg.RegisterDelegates([]string{`yorc\.nodes\.slurm\..*`}, newExecutor(&slurmGenerator{}), registry.BuiltinOrigin)
}
