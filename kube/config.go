package kube

type Config struct {
	Kube struct {
		OutCluster bool   `yaml:"outCluster" json:"outCluster"`
		ConfigPath string `yaml:"configPath" json:"configPath" default:"kubeconfig.yaml"`
	} `yaml:"kube" json:"kube"`
}
