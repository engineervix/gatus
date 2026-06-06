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
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/config/tenant"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/watchdog"
)

var (
	suiteTimestamp = time.Now()

	testSuiteEndpoint1 = endpoint.Endpoint{
		Name:                    "endpoint1",
		Group:                   "suite-group",
		URL:                     "https://example.org/endpoint1",
		Method:                  "GET",
		Interval:                30 * time.Second,
		Conditions:              []endpoint.Condition{endpoint.Condition("[STATUS] == 200"), endpoint.Condition("[RESPONSE_TIME] < 500")},
		NumberOfFailuresInARow:  0,
		NumberOfSuccessesInARow: 0,
	}
	testSuiteEndpoint2 = endpoint.Endpoint{
		Name:                    "endpoint2",
		Group:                   "suite-group",
		URL:                     "https://example.org/endpoint2",
		Method:                  "GET",
		Interval:                30 * time.Second,
		Conditions:              []endpoint.Condition{endpoint.Condition("[STATUS] == 200"), endpoint.Condition("[RESPONSE_TIME] < 300")},
		NumberOfFailuresInARow:  0,
		NumberOfSuccessesInARow: 0,
	}
	testSuite = suite.Suite{
		Name:     "test-suite",
		Group:    "suite-group",
		Interval: 60 * time.Second,
		Endpoints: []*endpoint.Endpoint{
			&testSuiteEndpoint1,
			&testSuiteEndpoint2,
		},
	}
	testSuccessfulSuiteResult = suite.Result{
		Name:      "test-suite",
		Group:     "suite-group",
		Success:   true,
		Timestamp: suiteTimestamp,
		Duration:  250 * time.Millisecond,
		EndpointResults: []*endpoint.Result{
			{
				Hostname:   "example.org",
				IP:         "127.0.0.1",
				HTTPStatus: 200,
				Success:    true,
				Timestamp:  suiteTimestamp,
				Duration:   100 * time.Millisecond,
				ConditionResults: []*endpoint.ConditionResult{
					{
						Condition: "[STATUS] == 200",
						Success:   true,
					},
					{
						Condition: "[RESPONSE_TIME] < 500",
						Success:   true,
					},
				},
			},
			{
				Hostname:   "example.org",
				IP:         "127.0.0.1",
				HTTPStatus: 200,
				Success:    true,
				Timestamp:  suiteTimestamp,
				Duration:   150 * time.Millisecond,
				ConditionResults: []*endpoint.ConditionResult{
					{
						Condition: "[STATUS] == 200",
						Success:   true,
					},
					{
						Condition: "[RESPONSE_TIME] < 300",
						Success:   true,
					},
				},
			},
		},
	}
	testUnsuccessfulSuiteResult = suite.Result{
		Name:      "test-suite",
		Group:     "suite-group",
		Success:   false,
		Timestamp: suiteTimestamp,
		Duration:  850 * time.Millisecond,
		Errors:    []string{"suite-error-1", "suite-error-2"},
		EndpointResults: []*endpoint.Result{
			{
				Hostname:   "example.org",
				IP:         "127.0.0.1",
				HTTPStatus: 200,
				Success:    true,
				Timestamp:  suiteTimestamp,
				Duration:   100 * time.Millisecond,
				ConditionResults: []*endpoint.ConditionResult{
					{
						Condition: "[STATUS] == 200",
						Success:   true,
					},
					{
						Condition: "[RESPONSE_TIME] < 500",
						Success:   true,
					},
				},
			},
			{
				Hostname:   "example.org",
				IP:         "127.0.0.1",
				HTTPStatus: 500,
				Errors:     []string{"endpoint-error-1"},
				Success:    false,
				Timestamp:  suiteTimestamp,
				Duration:   750 * time.Millisecond,
				ConditionResults: []*endpoint.ConditionResult{
					{
						Condition: "[STATUS] == 200",
						Success:   false,
					},
					{
						Condition: "[RESPONSE_TIME] < 300",
						Success:   false,
					},
				},
			},
		},
	}
)

func TestSuiteStatus(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	cfg := &config.Config{
		Metrics: true,
		Suites: []*suite.Suite{
			{
				Name:  "frontend-suite",
				Group: "core",
			},
			{
				Name:  "backend-suite",
				Group: "core",
			},
		},
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
	}
	watchdog.UpdateSuiteStatus(cfg.Suites[0], &suite.Result{Success: true, Duration: time.Millisecond, Timestamp: time.Now(), Name: cfg.Suites[0].Name, Group: cfg.Suites[0].Group})
	watchdog.UpdateSuiteStatus(cfg.Suites[1], &suite.Result{Success: false, Duration: time.Second, Timestamp: time.Now(), Name: cfg.Suites[1].Name, Group: cfg.Suites[1].Group})
	api := New(cfg)
	router := api.Router()
	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
		Gzip         bool
	}
	scenarios := []Scenario{
		{
			Name:         "suite-status",
			Path:         "/api/v1/suites/core_frontend-suite/statuses",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "suite-status-gzip",
			Path:         "/api/v1/suites/core_frontend-suite/statuses",
			ExpectedCode: http.StatusOK,
			Gzip:         true,
		},
		{
			Name:         "suite-status-pagination",
			Path:         "/api/v1/suites/core_frontend-suite/statuses?page=1&pageSize=20",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "suite-status-for-invalid-key",
			Path:         "/api/v1/suites/invalid_key/statuses",
			ExpectedCode: http.StatusNotFound,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
			if scenario.Gzip {
				request.Header.Set("Accept-Encoding", "gzip")
			}
			response, err := router.Test(request)
			if err != nil {
				return
			}
			if response.StatusCode != scenario.ExpectedCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.ExpectedCode, response.StatusCode)
			}
		})
	}
}

func TestSuiteStatus_SuiteNotInStoreButInConfig(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	tests := []struct {
		name         string
		suiteKey     string
		cfg          *config.Config
		expectedCode int
		expectJSON   bool
		expectError  string
	}{
		{
			name:     "suite-not-in-store-but-exists-in-config-enabled",
			suiteKey: "test-group_test-suite",
			cfg: &config.Config{
				Metrics: true,
				Suites: []*suite.Suite{
					{
						Name:    "test-suite",
						Group:   "test-group",
						Enabled: boolPtr(true),
						Endpoints: []*endpoint.Endpoint{
							{
								Name:  "endpoint-1",
								Group: "test-group",
								URL:   "https://example.com",
							},
						},
					},
				},
				Storage: &storage.Config{
					MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
					MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
				},
			},
			expectedCode: http.StatusOK,
			expectJSON:   true,
		},
		{
			name:     "suite-not-in-store-but-exists-in-config-disabled",
			suiteKey: "test-group_disabled-suite",
			cfg: &config.Config{
				Metrics: true,
				Suites: []*suite.Suite{
					{
						Name:    "disabled-suite",
						Group:   "test-group",
						Enabled: boolPtr(false),
					},
				},
				Storage: &storage.Config{
					MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
					MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
				},
			},
			expectedCode: http.StatusOK,
			expectJSON:   true,
		},
		{
			name:     "suite-not-in-store-and-not-in-config",
			suiteKey: "nonexistent_suite",
			cfg: &config.Config{
				Metrics: true,
				Suites: []*suite.Suite{
					{
						Name:  "different-suite",
						Group: "different-group",
					},
				},
				Storage: &storage.Config{
					MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
					MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
				},
			},
			expectedCode: http.StatusNotFound,
			expectError:  "not found",
		},
		{
			name:     "suite-with-empty-group-in-config",
			suiteKey: "_empty-group-suite",
			cfg: &config.Config{
				Metrics: true,
				Suites: []*suite.Suite{
					{
						Name:  "empty-group-suite",
						Group: "",
					},
				},
				Storage: &storage.Config{
					MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
					MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
				},
			},
			expectedCode: http.StatusOK,
			expectJSON:   true,
		},
		{
			name:     "suite-nil-enabled-defaults-to-true",
			suiteKey: "default_enabled-suite",
			cfg: &config.Config{
				Metrics: true,
				Suites: []*suite.Suite{
					{
						Name:    "enabled-suite",
						Group:   "default",
						Enabled: nil,
					},
				},
				Storage: &storage.Config{
					MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
					MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
				},
			},
			expectedCode: http.StatusOK,
			expectJSON:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := New(tt.cfg)
			router := api.Router()
			request := httptest.NewRequest("GET", "/api/v1/suites/"+tt.suiteKey+"/statuses", http.NoBody)
			response, err := router.Test(request)
			if err != nil {
				t.Fatalf("Router test failed: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, response.StatusCode)
			}
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}
			bodyStr := string(body)
			if tt.expectJSON {
				if response.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected JSON content type, got %s", response.Header.Get("Content-Type"))
				}
				if len(bodyStr) == 0 || bodyStr[0] != '{' {
					t.Errorf("Expected JSON response, got: %s", bodyStr)
				}
			}
			if tt.expectError != "" {
				if !contains(bodyStr, tt.expectError) {
					t.Errorf("Expected error message '%s' in response, got: %s", tt.expectError, bodyStr)
				}
			}
		})
	}
}

func TestSuiteStatuses(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	firstResult := &testSuccessfulSuiteResult
	secondResult := &testUnsuccessfulSuiteResult
	store.Get().InsertSuiteResult(&testSuite, firstResult)
	store.Get().InsertSuiteResult(&testSuite, secondResult)
	// Can't be bothered dealing with timezone issues on the worker that runs the automated tests
	firstResult.Timestamp = time.Time{}
	secondResult.Timestamp = time.Time{}
	for i := range firstResult.EndpointResults {
		firstResult.EndpointResults[i].Timestamp = time.Time{}
	}
	for i := range secondResult.EndpointResults {
		secondResult.EndpointResults[i].Timestamp = time.Time{}
	}
	api := New(&config.Config{
		Metrics: true,
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
	})
	router := api.Router()
	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
		ExpectedBody string
	}
	scenarios := []Scenario{
		{
			Name:         "no-pagination",
			Path:         "/api/v1/suites/statuses",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[{"name":"test-suite","group":"suite-group","key":"suite-group_test-suite","results":[{"name":"test-suite","group":"suite-group","success":true,"timestamp":"0001-01-01T00:00:00Z","duration":250000000,"endpointResults":[{"status":200,"hostname":"example.org","duration":100000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 300","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"}]},{"name":"test-suite","group":"suite-group","success":false,"timestamp":"0001-01-01T00:00:00Z","duration":850000000,"endpointResults":[{"status":200,"hostname":"example.org","duration":100000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":500,"hostname":"example.org","duration":750000000,"errors":["endpoint-error-1"],"conditionResults":[{"condition":"[STATUS] == 200","success":false},{"condition":"[RESPONSE_TIME] \u003c 300","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}],"errors":["suite-error-1","suite-error-2"]}]}]`,
		},
		{
			Name:         "pagination-first-result",
			Path:         "/api/v1/suites/statuses?page=1&pageSize=1",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[{"name":"test-suite","group":"suite-group","key":"suite-group_test-suite","results":[{"name":"test-suite","group":"suite-group","success":false,"timestamp":"0001-01-01T00:00:00Z","duration":850000000,"endpointResults":[{"status":200,"hostname":"example.org","duration":100000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":500,"hostname":"example.org","duration":750000000,"errors":["endpoint-error-1"],"conditionResults":[{"condition":"[STATUS] == 200","success":false},{"condition":"[RESPONSE_TIME] \u003c 300","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}],"errors":["suite-error-1","suite-error-2"]}]}]`,
		},
		{
			Name:         "pagination-second-result",
			Path:         "/api/v1/suites/statuses?page=2&pageSize=1",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[{"name":"test-suite","group":"suite-group","key":"suite-group_test-suite","results":[{"name":"test-suite","group":"suite-group","success":true,"timestamp":"0001-01-01T00:00:00Z","duration":250000000,"endpointResults":[{"status":200,"hostname":"example.org","duration":100000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 300","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"}]}]}]`,
		},
		{
			Name:         "pagination-no-results",
			Path:         "/api/v1/suites/statuses?page=5&pageSize=20",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[{"name":"test-suite","group":"suite-group","key":"suite-group_test-suite","results":[]}]`,
		},
		{
			Name:         "invalid-pagination-should-fall-back-to-default",
			Path:         "/api/v1/suites/statuses?page=INVALID&pageSize=INVALID",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[{"name":"test-suite","group":"suite-group","key":"suite-group_test-suite","results":[{"name":"test-suite","group":"suite-group","success":true,"timestamp":"0001-01-01T00:00:00Z","duration":250000000,"endpointResults":[{"status":200,"hostname":"example.org","duration":100000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 300","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"}]},{"name":"test-suite","group":"suite-group","success":false,"timestamp":"0001-01-01T00:00:00Z","duration":850000000,"endpointResults":[{"status":200,"hostname":"example.org","duration":100000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":500,"hostname":"example.org","duration":750000000,"errors":["endpoint-error-1"],"conditionResults":[{"condition":"[STATUS] == 200","success":false},{"condition":"[RESPONSE_TIME] \u003c 300","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}],"errors":["suite-error-1","suite-error-2"]}]}]`,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
			response, err := router.Test(request)
			if err != nil {
				return
			}
			defer response.Body.Close()
			if response.StatusCode != scenario.ExpectedCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.ExpectedCode, response.StatusCode)
			}
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Error("expected err to be nil, but was", err)
			}
			if string(body) != scenario.ExpectedBody {
				t.Errorf("expected:\n %s\n\ngot:\n %s", scenario.ExpectedBody, string(body))
			}
		})
	}
}

func TestSuiteStatuses_NoSuitesInStoreButExistInConfig(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	cfg := &config.Config{
		Metrics: true,
		Suites: []*suite.Suite{
			{
				Name:    "config-only-suite-1",
				Group:   "test-group",
				Enabled: boolPtr(true),
			},
			{
				Name:    "config-only-suite-2",
				Group:   "test-group",
				Enabled: boolPtr(true),
			},
			{
				Name:    "disabled-suite",
				Group:   "test-group",
				Enabled: boolPtr(false),
			},
		},
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
	}
	api := New(cfg)
	router := api.Router()
	request := httptest.NewRequest("GET", "/api/v1/suites/statuses", http.NoBody)
	response, err := router.Test(request)
	if err != nil {
		t.Fatalf("Router test failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	bodyStr := string(body)
	if !contains(bodyStr, "config-only-suite-1") {
		t.Error("Expected config-only-suite-1 in response")
	}
	if !contains(bodyStr, "config-only-suite-2") {
		t.Error("Expected config-only-suite-2 in response")
	}
	if contains(bodyStr, "disabled-suite") {
		t.Error("Should not include disabled-suite in response")
	}
}

func TestSuiteStatuses_TenantFiltering(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	clientASuite := &suite.Suite{
		Name:    "client-a-suite",
		Group:   "tenanted",
		Tenants: []string{"client-a"},
	}
	defaultSuite := &suite.Suite{
		Name:  "default-suite",
		Group: "tenanted",
	}
	watchdog.UpdateSuiteStatus(clientASuite, &suite.Result{Success: true, Duration: time.Millisecond, Timestamp: time.Now(), Name: clientASuite.Name, Group: clientASuite.Group})
	watchdog.UpdateSuiteStatus(defaultSuite, &suite.Result{Success: true, Duration: time.Millisecond, Timestamp: time.Now(), Name: defaultSuite.Name, Group: defaultSuite.Group})
	cfg := &config.Config{
		Metrics: true,
		Suites:  []*suite.Suite{clientASuite, defaultSuite},
		Tenants: []*tenant.Tenant{
			{Name: "client-a", Domains: []string{"status.clienta.com"}},
		},
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
	}
	api := New(cfg)
	router := api.Router()
	type Scenario struct {
		Name          string
		Host          string
		Path          string
		ExpectedCode  int
		ShouldContain string
		ShouldExclude string
	}
	scenarios := []Scenario{
		{
			Name:          "default-domain-sees-only-default-suite",
			Host:          "",
			Path:          "/api/v1/suites/statuses",
			ExpectedCode:  http.StatusOK,
			ShouldContain: "default-suite",
			ShouldExclude: "client-a-suite",
		},
		{
			Name:          "tenant-domain-sees-only-its-suite",
			Host:          "status.clienta.com",
			Path:          "/api/v1/suites/statuses",
			ExpectedCode:  http.StatusOK,
			ShouldContain: "client-a-suite",
			ShouldExclude: "default-suite",
		},
		{
			Name:         "tenant-domain-cannot-access-default-suite",
			Host:         "status.clienta.com",
			Path:         "/api/v1/suites/tenanted_default-suite/statuses",
			ExpectedCode: http.StatusNotFound,
		},
		{
			Name:         "default-domain-cannot-access-tenant-suite",
			Host:         "",
			Path:         "/api/v1/suites/tenanted_client-a-suite/statuses",
			ExpectedCode: http.StatusNotFound,
		},
		{
			Name:         "tenant-domain-can-access-its-suite",
			Host:         "status.clienta.com",
			Path:         "/api/v1/suites/tenanted_client-a-suite/statuses",
			ExpectedCode: http.StatusOK,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
			if scenario.Host != "" {
				request.Host = scenario.Host
			}
			response, err := router.Test(request)
			if err != nil {
				t.Fatalf("Router test failed: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != scenario.ExpectedCode {
				t.Errorf("%s %s (Host: %s) should have returned %d, but returned %d instead", request.Method, request.URL, scenario.Host, scenario.ExpectedCode, response.StatusCode)
			}
			if scenario.ShouldContain != "" || scenario.ShouldExclude != "" {
				body, err := io.ReadAll(response.Body)
				if err != nil {
					t.Fatalf("Failed to read response body: %v", err)
				}
				bodyStr := string(body)
				if scenario.ShouldContain != "" && !contains(bodyStr, scenario.ShouldContain) {
					t.Errorf("expected body to contain %q, got: %s", scenario.ShouldContain, bodyStr)
				}
				if scenario.ShouldExclude != "" && contains(bodyStr, scenario.ShouldExclude) {
					t.Errorf("expected body NOT to contain %q, got: %s", scenario.ShouldExclude, bodyStr)
				}
			}
		})
	}
}

func TestSuiteStatus_404IsPlainText(t *testing.T) {
	// All SuiteStatus 404s must return plain text to match EndpointStatus.
	// Uses a mixed-case key to also exercise the normalisation path: the guard
	// finds the suite via GetSuiteByKey (which lowercases), but without the fix
	// the fallback loop uses the raw key and misses, returning a JSON 404.
	defer store.Get().Clear()
	defer cache.Clear()
	cfg := &config.Config{
		Suites: []*suite.Suite{
			{Name: "cold-suite", Group: "test"}, // no store records
		},
	}
	router := New(cfg).Router()
	scenarios := []struct {
		name         string
		path         string
		expectedCode int
	}{
		// key not in config at all → 404 plain text
		{"not-in-config", "/api/v1/suites/test_ghost/statuses", http.StatusNotFound},
		// mixed-case key for a suite with no store records:
		// guard passes (GetSuiteByKey lowercases), fallback must serve stub (not JSON 404)
		{"mixed-case-no-store-returns-stub", "/api/v1/suites/Test_Cold-Suite/statuses", http.StatusOK},
	}
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, sc.path, http.NoBody)
			resp, err := router.Test(req)
			if err != nil {
				t.Fatalf("router.Test: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != sc.expectedCode {
				t.Errorf("expected %d, got %d", sc.expectedCode, resp.StatusCode)
				return
			}
			ct := resp.Header.Get("Content-Type")
			if sc.expectedCode == http.StatusNotFound && strings.Contains(ct, "application/json") {
				t.Errorf("404 must not be JSON (Content-Type: %s)", ct)
			}
		})
	}
}

func TestSuiteStatuses_DisabledSuiteWithStoreRecordsIsHidden(t *testing.T) {
	disabledSuite := &suite.Suite{Name: "disabled-suite", Group: "test", Enabled: boolPtr(false)}
	enabledSuite := &suite.Suite{Name: "enabled-suite", Group: "test"}
	// Both have store records
	watchdog.UpdateSuiteStatus(disabledSuite, &suite.Result{Success: true, Duration: time.Millisecond, Timestamp: time.Now(), Name: disabledSuite.Name, Group: disabledSuite.Group})
	watchdog.UpdateSuiteStatus(enabledSuite, &suite.Result{Success: true, Duration: time.Millisecond, Timestamp: time.Now(), Name: enabledSuite.Name, Group: enabledSuite.Group})
	defer store.Get().Clear()
	defer cache.Clear()

	cfg := &config.Config{
		Suites: []*suite.Suite{disabledSuite, enabledSuite},
	}
	router := New(cfg).Router()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/statuses", http.NoBody)
	resp, err := router.Test(req)
	if err != nil {
		t.Fatalf("router.Test: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	if contains(bodyStr, "disabled-suite") {
		t.Errorf("disabled suite must not appear in SuiteStatuses response, got: %s", bodyStr)
	}
	if !contains(bodyStr, "enabled-suite") {
		t.Errorf("enabled suite must appear in SuiteStatuses response, got: %s", bodyStr)
	}
}

func TestSuiteStatus_GhostKeyAndFallbackTenantChecks(t *testing.T) {
	ghostSuite := &suite.Suite{Name: "removed-suite", Group: "tenanted"}
	tenantSuite := &suite.Suite{Name: "tenant-suite", Group: "tenanted", Tenants: []string{"client-a"}}
	defaultSuite := &suite.Suite{Name: "default-suite", Group: "tenanted"}
	// Seed the store for the ghost and tenant suites
	watchdog.UpdateSuiteStatus(ghostSuite, &suite.Result{Success: true, Duration: time.Millisecond, Timestamp: time.Now(), Name: ghostSuite.Name, Group: ghostSuite.Group})
	watchdog.UpdateSuiteStatus(tenantSuite, &suite.Result{Success: true, Duration: time.Millisecond, Timestamp: time.Now(), Name: tenantSuite.Name, Group: tenantSuite.Group})
	defer store.Get().Clear()
	defer cache.Clear()

	// Config does NOT include ghostSuite — simulates a removed suite still in the store
	cfg := &config.Config{
		Suites: []*suite.Suite{tenantSuite, defaultSuite},
		Tenants: []*tenant.Tenant{
			{Name: "client-a", Domains: []string{"status.clienta.com"}},
		},
	}
	router := New(cfg).Router()

	scenarios := []struct {
		name         string
		host         string
		path         string
		expectedCode int
	}{
		{
			// Ghost key: in store but absent from config — any domain must get 404
			name:         "ghost-key-on-default-domain-returns-404",
			host:         "",
			path:         "/api/v1/suites/tenanted_removed-suite/statuses",
			expectedCode: http.StatusNotFound,
		},
		{
			// Ghost key: tenant domain must also get 404
			name:         "ghost-key-on-tenant-domain-returns-404",
			host:         "status.clienta.com",
			path:         "/api/v1/suites/tenanted_removed-suite/statuses",
			expectedCode: http.StatusNotFound,
		},
		{
			// Fallback path: default-only suite requested from a tenant domain with empty store
			// Should return 404, not serve a stub built from config without tenant check
			name:         "config-fallback-respects-tenant-on-tenant-domain",
			host:         "status.clienta.com",
			path:         "/api/v1/suites/tenanted_default-suite/statuses",
			expectedCode: http.StatusNotFound,
		},
	}
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, sc.path, http.NoBody)
			if sc.host != "" {
				req.Host = sc.host
			}
			resp, err := router.Test(req)
			if err != nil {
				t.Fatalf("router.Test: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != sc.expectedCode {
				t.Errorf("path=%s host=%s: expected %d, got %d", sc.path, sc.host, sc.expectedCode, resp.StatusCode)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
