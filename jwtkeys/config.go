package jwtkeys

// JwtConfig defines the files and claims settings used by the key store.
type JwtKeysConfig struct {
	PrivateKey string `mapstructure:"private_key"`
	PublicKey  string `mapstructure:"public_key"`
}

func (config JwtKeysConfig) withDefaults() JwtKeysConfig {
	result := config

	if result.PrivateKey == "" {
		result.PrivateKey = "./keys/private.pem"
	}

	if result.PublicKey == "" {
		result.PublicKey = "./keys/public.pem"
	}

	return result
}
