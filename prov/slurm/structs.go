package slurm

type infrastructure struct {
	nodes []*nodeAllocation
}

type nodeAllocation struct {
	cpu          string
	memory       string
	gres         string
	partition    string
	jobName      string
	instanceName string
}
