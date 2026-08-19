package jwtkeys

import "testing"

func TestJwtConfigWithDefaults(t *testing.T) {
	tests := []struct {
		name     string
		config   JwtKeysConfig
		expected JwtKeysConfig
	}{
		{
			name: "zero value",
			expected: JwtKeysConfig{
				PrivateKey: "./keys/private.pem",
				PublicKey:  "./keys/public.pem",
			},
		},
		{
			name: "preserves configured values",
			config: JwtKeysConfig{
				PrivateKey: "/etc/service/private.pem",
				PublicKey:  "/etc/service/public.pem",
			},
			expected: JwtKeysConfig{
				PrivateKey: "/etc/service/private.pem",
				PublicKey:  "/etc/service/public.pem",
			},
		},
		{
			name: "fills only missing values",
			config: JwtKeysConfig{
				PrivateKey: "/etc/service/private.pem",
			},
			expected: JwtKeysConfig{
				PrivateKey: "/etc/service/private.pem",
				PublicKey:  "./keys/public.pem",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.config.withDefaults(); got != test.expected {
				t.Errorf("JwtKeysConfig.withDefaults() = %+v, want %+v", got, test.expected)
			}
		})
	}
}
