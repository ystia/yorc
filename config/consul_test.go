package config

import (
	"testing"
)

func TestConfiguration_GetConsulClient(t *testing.T) {
	type fields struct {
		ConsulToken      string
		ConsulDatacenter string
		ConsulAddress    string
		ConsulKey        string
		ConsulCert       string
		ConsulCA         string
		ConsulCAPath     string
		ConsulSSL        bool
		ConsulSSLVerify  bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"TestDefaultConfig", fields{}, false},
		{"TestCustomConfig", fields{"token", "dc", "http://127.0.0.1:8500", "testdata/comp.key", "testdata/comp.pem", "testdata/ca.pem", "/path/ca/dir", true, true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Configuration{
				ConsulToken:      tt.fields.ConsulToken,
				ConsulDatacenter: tt.fields.ConsulDatacenter,
				ConsulAddress:    tt.fields.ConsulAddress,
				ConsulKey:        tt.fields.ConsulKey,
				ConsulCert:       tt.fields.ConsulCert,
				ConsulCA:         tt.fields.ConsulCA,
				ConsulCAPath:     tt.fields.ConsulCAPath,
				ConsulSSL:        tt.fields.ConsulSSL,
				ConsulSSLVerify:  tt.fields.ConsulSSLVerify,
			}
			got, err := cfg.GetConsulClient()
			if (err != nil) != tt.wantErr {
				t.Errorf("Configuration.GetConsulClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				t.Error("Configuration.GetConsulClient() client is nil")
				return
			}
		})
	}
}
