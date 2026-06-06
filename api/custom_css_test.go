package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/tenant"
	"github.com/TwiN/gatus/v5/config/ui"
)

func TestGetCustomCSS(t *testing.T) {
	cfg := &config.Config{
		UI: &ui.Config{
			CustomCSS: "body { background: red; }",
		},
		Tenants: []*tenant.Tenant{
			{
				Name:    "css-tenant",
				Domains: []string{"status.css-tenant.com"},
				UI: &ui.Config{
					CustomCSS: "body { background: blue; }",
				},
			},
			{
				Name:    "empty-ui-tenant",
				Domains: []string{"status.empty-ui.com"},
				UI:      &ui.Config{},
			},
			{
				Name:    "no-ui-tenant",
				Domains: []string{"status.no-ui.com"},
				// No UI block — fix #5 ensures t.UI is initialised to &ui.Config{} by ValidateAndSetDefaults
			},
		},
	}
	// Initialise tenant defaults (mirrors what config loading does)
	for _, tn := range cfg.Tenants {
		if err := tn.ValidateAndSetDefaults(); err != nil {
			t.Fatalf("tenant.ValidateAndSetDefaults: %v", err)
		}
	}
	router := New(cfg).Router()

	scenarios := []struct {
		name         string
		host         string
		wantBody     string
		wantNotBody  string
	}{
		{
			name:     "non-tenant-host-gets-operator-css",
			host:     "status.operator.com",
			wantBody: "body { background: red; }",
		},
		{
			name:     "tenant-with-custom-css-gets-its-own-css",
			host:     "status.css-tenant.com",
			wantBody: "body { background: blue; }",
		},
		{
			// A tenant with an explicit ui: {} block but no custom-css must receive
			// an empty stylesheet, not the operator's custom CSS.
			name:        "tenant-with-empty-ui-gets-empty-stylesheet",
			host:        "status.empty-ui.com",
			wantBody:    "",
			wantNotBody: "body { background: red; }",
		},
		{
			// A tenant with no ui: block at all (fix #5 initialises t.UI) must also
			// receive an empty stylesheet.
			name:        "tenant-with-no-ui-block-gets-empty-stylesheet",
			host:        "status.no-ui.com",
			wantBody:    "",
			wantNotBody: "body { background: red; }",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/css/custom.css", http.NoBody)
			req.Host = sc.host
			resp, err := router.Test(req)
			if err != nil {
				t.Fatalf("router.Test: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200, got %d", resp.StatusCode)
			}
			body, _ := io.ReadAll(resp.Body)
			got := string(body)
			if got != sc.wantBody {
				t.Errorf("body = %q, want %q", got, sc.wantBody)
			}
			if sc.wantNotBody != "" && got == sc.wantNotBody {
				t.Errorf("body must not equal %q (operator CSS leaked to tenant domain)", sc.wantNotBody)
			}
		})
	}
}
