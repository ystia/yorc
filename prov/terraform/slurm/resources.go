package slurm

// A ComputeInstance represent an Slurm compute node
type ComputeInstance struct {
	GpuType string `json:"gpu,omitempty"`
}

type Cntk struct {
	Partition    string `json:"partition,omitempty"`
	RunAsUser    string `json:"runAsUser,omitempty"`
	ModelPath    string `json:"modelPath,omitempty"`
	ModelFile    string `json:"modelFile,omitempty"`
	ImgPath      string `json:"imgPath,omitempty"`
	NbNode       string `json:"nbNode,omitempty"`
	NodesName    string `json:"nodesName,omitempty"`
	NbMpiProcess string `json:"nbMpiProcess,omitempty"`
}
