package api

import (
	"github.com/TwiN/gatus/v5/config"
)

// resolveTenantName returns the tenant name that matches the given hostname,
// or an empty string when no tenant matches (default domain).
func resolveTenantName(cfg *config.Config, hostname string) string {
	if t := cfg.GetTenantByDomain(hostname); t != nil {
		return t.Name
	}
	return ""
}

// hasEndpointAccess reports whether the given hostname has permission to view the
// endpoint identified by key. Keys absent from config are denied; the store will
// return ErrEndpointNotFound for keys that genuinely do not exist.
func hasEndpointAccess(cfg *config.Config, hostname, key string) bool {
	tenantName := resolveTenantName(cfg, hostname)
	if ep := cfg.GetEndpointByKey(key); ep != nil {
		return ep.BelongsToTenant(tenantName)
	}
	if ee := cfg.GetExternalEndpointByKey(key); ee != nil {
		return ee.BelongsToTenant(tenantName)
	}
	return false
}
