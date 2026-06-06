package tenant

import (
	"errors"

	"github.com/TwiN/gatus/v5/config/ui"
)

var (
	// ErrTenantNameEmpty is returned when a tenant has no name
	ErrTenantNameEmpty = errors.New("tenant name cannot be empty")
	
	// ErrTenantDomainsEmpty is returned when a tenant has no domains configured
	ErrTenantDomainsEmpty = errors.New("tenant must have at least one domain")
)

// Tenant represents a single tenant configuration for multi-domain support
type Tenant struct {
	Name    string     `yaml:"name"`
	Domains []string   `yaml:"domains"`
	UI      *ui.Config `yaml:"ui,omitempty"`
}

// ValidateAndSetDefaults validates the tenant configuration
func (t *Tenant) ValidateAndSetDefaults() error {
	if len(t.Name) == 0 {
		return ErrTenantNameEmpty
	}
	if len(t.Domains) == 0 {
		return ErrTenantDomainsEmpty
	}
	if t.UI != nil {
		if err := t.UI.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	return nil
}
