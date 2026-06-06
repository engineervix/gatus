package tenant

import (
	"testing"

	"github.com/TwiN/gatus/v5/config/ui"
)

func TestTenant_ValidateAndSetDefaults(t *testing.T) {
	t.Run("valid-tenant-no-ui", func(t *testing.T) {
		tenant := &Tenant{
			Name:    "client-a",
			Domains: []string{"status.clienta.com"},
		}
		if err := tenant.ValidateAndSetDefaults(); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if tenant.UI == nil {
			t.Error("expected tenant.UI to be initialised with defaults, got nil")
		}
		if tenant.UI != nil && tenant.UI.Header == "" {
			t.Error("expected tenant.UI.Header to have a default value after ValidateAndSetDefaults")
		}
	})

	t.Run("valid-tenant-with-ui", func(t *testing.T) {
		tenant := &Tenant{
			Name:    "client-b",
			Domains: []string{"status.clientb.com"},
			UI: &ui.Config{
				Title: "Client B",
			},
		}
		if err := tenant.ValidateAndSetDefaults(); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if tenant.UI.Title != "Client B" {
			t.Errorf("expected title to be 'Client B', got %s", tenant.UI.Title)
		}
		// Check that defaults were set by the UI validate method
		if tenant.UI.Description == "" {
			t.Error("expected description to be set by defaults")
		}
	})

	t.Run("invalid-tenant-no-name", func(t *testing.T) {
		tenant := &Tenant{
			Domains: []string{"status.clienta.com"},
		}
		if err := tenant.ValidateAndSetDefaults(); err != ErrTenantNameEmpty {
			t.Errorf("expected %v, got %v", ErrTenantNameEmpty, err)
		}
	})

	t.Run("invalid-tenant-no-domains", func(t *testing.T) {
		tenant := &Tenant{
			Name: "client-a",
		}
		if err := tenant.ValidateAndSetDefaults(); err != ErrTenantDomainsEmpty {
			t.Errorf("expected %v, got %v", ErrTenantDomainsEmpty, err)
		}
	})
	
	t.Run("invalid-tenant-invalid-ui", func(t *testing.T) {
		tenant := &Tenant{
			Name: "client-a",
			Domains: []string{"status.clienta.com"},
			UI: &ui.Config{
				DefaultSortBy: "invalid_sort",
			},
		}
		if err := tenant.ValidateAndSetDefaults(); err == nil {
			t.Error("expected error due to invalid UI config, got nil")
		}
	})
}
