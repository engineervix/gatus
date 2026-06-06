package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/tenant"
	"github.com/TwiN/gatus/v5/config/ui"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/watchdog"
)

func TestSinglePageApplication(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	cfg := &config.Config{
		Metrics: true,
		Endpoints: []*endpoint.Endpoint{
			{
				Name:  "frontend",
				Group: "core",
			},
			{
				Name:  "backend",
				Group: "core",
			},
		},
		UI: &ui.Config{
			Title: "example-title",
		},
		Tenants: []*tenant.Tenant{
			{
				Name: "client",
				Domains: []string{"status.client.com"},
				UI: &ui.Config{
					Title: "Client Status",
				},
			},
		},
	}
	watchdog.UpdateEndpointStatus(cfg.Endpoints[0], &endpoint.Result{Success: true, Duration: time.Millisecond, Timestamp: time.Now()})
	watchdog.UpdateEndpointStatus(cfg.Endpoints[1], &endpoint.Result{Success: false, Duration: time.Second, Timestamp: time.Now()})
	api := New(cfg)
	router := api.Router()
	type Scenario struct {
		Name              string
		Host              string
		Path              string
		Gzip              bool
		CookieDarkMode    bool
		UIDarkMode        bool
		ExpectedCode      int
		ExpectedDarkTheme bool
	}
	scenarios := []Scenario{
		{
			Name:              "frontend-home",
			Host:              "status.gatus.io", // Doesn't match any tenant
			Path:              "/",
			CookieDarkMode:    true,
			UIDarkMode:        false,
			ExpectedDarkTheme: true,
			ExpectedCode:      200,
		},
		{
			Name:              "frontend-tenant-home",
			Host:              "status.client.com", // Matches tenant
			Path:              "/",
			CookieDarkMode:    true,
			UIDarkMode:        false,
			ExpectedDarkTheme: true,
			ExpectedCode:      200,
		},
		{
			Name:              "frontend-endpoint-light",
			Path:              "/endpoints/core_frontend",
			CookieDarkMode:    false,
			UIDarkMode:        false,
			ExpectedDarkTheme: false,
			ExpectedCode:      200,
		},
		{
			Name:              "frontend-endpoint-dark",
			Path:              "/endpoints/core_frontend",
			CookieDarkMode:    false,
			UIDarkMode:        true,
			ExpectedDarkTheme: true,
			ExpectedCode:      200,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			cfg.UI.DarkMode = &scenario.UIDarkMode
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
			if scenario.Host != "" {
				request.Host = scenario.Host
			}
			if scenario.Gzip {
				request.Header.Set("Accept-Encoding", "gzip")
			}
			if scenario.CookieDarkMode {
				request.Header.Set("Cookie", "theme=dark")
			}
			response, err := router.Test(request)
			if err != nil {
				return
			}
			defer response.Body.Close()
			if response.StatusCode != scenario.ExpectedCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.ExpectedCode, response.StatusCode)
			}
			body, _ := io.ReadAll(response.Body)
			strBody := string(body)
			
			expectedTitle := cfg.UI.Title
			if scenario.Host == "status.client.com" {
				expectedTitle = "Client Status"
			}
			if !strings.Contains(strBody, expectedTitle) {
				t.Errorf("%s %s (Host: %s) should have contained the title %s", request.Method, request.URL, scenario.Host, expectedTitle)
			}
			if scenario.ExpectedDarkTheme && !strings.Contains(strBody, "class=\"dark\"") {
				t.Errorf("%s %s should have responded with dark mode headers", request.Method, request.URL)
			}
			if !scenario.ExpectedDarkTheme && strings.Contains(strBody, "class=\"dark\"") {
				t.Errorf("%s %s should not have responded with dark mode headers", request.Method, request.URL)
			}
		})
	}
}
