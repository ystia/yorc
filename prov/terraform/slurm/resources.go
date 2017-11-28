package slurm

// A ComputeInstance represent an Slurm compute node
type ComputeInstance struct {
	CPU       string `json:"cpu,omitempty"`
	Memory    string `json:"memory,omitempty"`
	Gres      string `json:"gres,omitempty"`
	Partition string `json:"partition,omitempty"`
}

// A Cntk represent a CNTK singularity instance running on top of SLURM compute nodes
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
