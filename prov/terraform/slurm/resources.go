package slurm

type ComputeInstance struct {
	GpuType string `json:"gpu,omitempty"`
}

type Job struct {
	ModelPath   string `json:"modelPath,omitempty"`
	ModelFile   string `json:"modelFile,omitempty"`
	ImgPath      string `json:"imgPath,omitempty"`
	NbNode       string `json:"nbNode,omitempty"`
	NodesName    string `json:"nodesName,omitempty"`
	NbMpiProcess string `json:"nbMpiProcess,omitempty"`
}
