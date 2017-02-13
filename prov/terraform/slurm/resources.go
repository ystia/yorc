package slurm

// A ComputeInstance represent an Slurm compute node
type ComputeInstance struct {
	GpuType string `json:"gpu,omitempty"`
}
