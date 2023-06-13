package hconfig

type HConfig struct {
	Feishu *Feishu `yaml:"feishu"`
	Dump   *Dump   `yaml:"dump"`
}

type Feishu struct {
	Url string `yaml:"url"`
	Msg string `yaml:"msg"`
}

type Dump struct {
	OssRs   string `yaml:"ossRs"`
	OssPath string `yaml:"ossPath"`
	OssPod  string `yaml:"ossPod"`
}
