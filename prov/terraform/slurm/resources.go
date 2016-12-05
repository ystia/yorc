package slurm

type ComputeInstance struct {
	GpuType string `json:"gpu,omitempty"`
}

type Job struct {
	ScriptPath string `json:"scriptPath,omitempty"`
	ImgPath string `json:"imgPath,omitempty"`
	NbNode string `json:"nbNode,omitempty"`
	NodesName string `json:"nodesName,omitempty"`
}
