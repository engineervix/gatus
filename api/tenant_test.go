package api

import (
	"testing"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/tenant"
)

func TestHasEndpointAccess(t *testing.T) {
	cfg := &config.Config{
		Endpoints: []*endpoint.Endpoint{
			{Name: "frontend", Group: "core"},
			{Name: "dashboard", Group: "core", Tenants: []string{"client-a"}},
		},
		ExternalEndpoints: []*endpoint.ExternalEndpoint{
			{Name: "ext-api", Group: "core", Tenants: []string{"client-a"}},
		},
		Tenants: []*tenant.Tenant{
			{Name: "client-a", Domains: []string{"status.clienta.com"}},
		},
	}

	tests := []struct {
		name     string
		hostname string
		key      string
		want     bool
	}{
		// Regular endpoints
		{"default domain sees shared endpoint", "", "core_frontend", true},
		{"default domain cannot see tenant-only endpoint", "", "core_dashboard", false},
		{"tenant sees its own endpoint", "status.clienta.com", "core_dashboard", true},
		{"tenant cannot see shared endpoint", "status.clienta.com", "core_frontend", false},
		// External endpoints
		{"tenant sees its external endpoint", "status.clienta.com", "core_ext-api", true},
		{"default domain cannot see tenant external endpoint", "", "core_ext-api", false},
		// Ghost keys — in store but absent from config; must always be denied
		{"ghost key denied on tenant domain", "status.clienta.com", "core_ghost", false},
		{"ghost key denied on default domain", "", "core_ghost", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasEndpointAccess(cfg, tt.hostname, tt.key)
			if got != tt.want {
				t.Errorf("hasEndpointAccess(%q, %q) = %v, want %v", tt.hostname, tt.key, got, tt.want)
			}
		})
	}
}
