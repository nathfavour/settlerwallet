package orchestration

type Config struct {
	Orchestration struct {
		Compliance ComplianceConfig `yaml:"compliance"`
		Privacy    PrivacyConfig    `yaml:"privacy"`
		Intent     IntentConfig     `yaml:"intent"`
	} `yaml:"orchestration"`
}

type ComplianceConfig struct {
	Enabled         bool    `yaml:"enabled"`
	RiskAPIURL      string  `yaml:"risk_api_url"`
	RiskThreshold   float64 `yaml:"risk_threshold"`
	APIKeySecretID  string  `yaml:"api_key_secret_id"`
}

type PrivacyConfig struct {
	Enabled   bool   `yaml:"enabled"`
	KYCAPIURL string `yaml:"kyc_api_url"`
	SecretID  string `yaml:"secret_id"`
}

type IntentConfig struct {
	Enabled        bool   `yaml:"enabled"`
	LLMAPIURL      string `yaml:"llm_api_url"`
	Model          string `yaml:"model"`
	APIKeySecretID string `yaml:"api_key_secret_id"`
}

type SettlementObject struct {
	Asset       string  `json:"asset"`
	Amount      float64 `json:"amount"`
	Destination string  `json:"destination"`
}
