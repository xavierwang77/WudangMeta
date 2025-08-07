package llm

type DeepSeekConfig struct {
	Model   string `json:"model"`
	ApiKey  string `json:"api_key"`
	BaseUrl string `json:"base_url"`
}
