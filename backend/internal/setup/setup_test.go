package setup

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSetupGuardRejectsRemoteRequestsByDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("SKIP_SETUP", "")
	t.Setenv("SETUP_ALLOW_REMOTE", "")

	router := gin.New()
	router.POST("/setup/install", setupGuard(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/setup/install", nil)
	request.RemoteAddr = "203.0.113.10:12345"
	request.Host = "127.0.0.1:8080"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestSetupGuardAllowsLocalRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("SKIP_SETUP", "")
	t.Setenv("SETUP_ALLOW_REMOTE", "")

	router := gin.New()
	router.POST("/setup/install", setupGuard(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/setup/install", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	request.Host = "localhost:8080"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestSetupGuardRequiresRemoteToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("SKIP_SETUP", "")
	t.Setenv("SETUP_ALLOW_REMOTE", "true")
	t.Setenv("SETUP_REMOTE_TOKEN", "0123456789abcdef0123456789abcdef")

	router := gin.New()
	router.POST("/setup/install", setupGuard(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for _, tc := range []struct {
		name  string
		token string
		want  int
	}{
		{name: "missing", want: http.StatusForbidden},
		{name: "wrong", token: "fedcba9876543210fedcba9876543210", want: http.StatusForbidden},
		{name: "valid", token: "0123456789abcdef0123456789abcdef", want: http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/setup/install", nil)
			request.RemoteAddr = "203.0.113.10:12345"
			request.Host = "setup.example.com"
			if tc.token != "" {
				request.Header.Set("X-Setup-Token", tc.token)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			require.Equal(t, tc.want, recorder.Code)
		})
	}
}

func TestIsLocalSetupRequest(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		host       string
		headers    http.Header
		want       bool
	}{
		{name: "IPv4 loopback", remoteAddr: "127.0.0.1:12345", host: "127.0.0.1:8080", want: true},
		{name: "IPv6 loopback", remoteAddr: "[::1]:12345", host: "[::1]:8080", want: true},
		{name: "localhost", remoteAddr: "127.0.0.1:12345", host: "localhost:8080", want: true},
		{name: "public address", remoteAddr: "203.0.113.10:12345", host: "127.0.0.1:8080", want: false},
		{name: "DNS rebinding host", remoteAddr: "127.0.0.1:12345", host: "attacker.example", want: false},
		{name: "forwarded loopback", remoteAddr: "127.0.0.1:12345", host: "127.0.0.1:8080", headers: http.Header{"X-Forwarded-For": []string{"203.0.113.10"}}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := &http.Request{RemoteAddr: test.remoteAddr, Host: test.host, Header: test.headers}
			if req.Header == nil {
				req.Header = make(http.Header)
			}
			if got := isLocalSetupRequest(req); got != test.want {
				t.Fatalf("isLocalSetupRequest() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestValidateSetupTargetAddress(t *testing.T) {
	t.Setenv("SETUP_ALLOW_PRIVATE_TARGETS", "")

	allowed := []string{"127.0.0.1", "::1", "8.8.8.8", "2606:4700:4700::1111"}
	for _, raw := range allowed {
		if err := validateSetupTargetAddress(netip.MustParseAddr(raw), true); err != nil {
			t.Fatalf("validateSetupTargetAddress(%s) error = %v", raw, err)
		}
	}

	blocked := []string{"0.0.0.0", "10.0.0.1", "100.64.0.1", "169.254.169.254", "172.16.0.1", "192.168.1.1", "224.0.0.1", "fc00::1", "fe80::1"}
	for _, raw := range blocked {
		if err := validateSetupTargetAddress(netip.MustParseAddr(raw), false); err == nil {
			t.Fatalf("validateSetupTargetAddress(%s) error = nil, want rejection", raw)
		}
	}
}

func TestValidateSetupTargetAddressAllowsPrivateOptIn(t *testing.T) {
	t.Setenv("SETUP_ALLOW_PRIVATE_TARGETS", "true")
	if err := validateSetupTargetAddress(netip.MustParseAddr("192.168.1.10"), privateSetupTargetsEnabled()); err != nil {
		t.Fatalf("private target opt-in error = %v", err)
	}
	if err := validateSetupTargetAddress(netip.MustParseAddr("169.254.169.254"), privateSetupTargetsEnabled()); err == nil {
		t.Fatal("link-local metadata target must remain blocked")
	}
}

func TestRemoteSetupEnabledRequiresExplicitOptIn(t *testing.T) {
	t.Setenv("SETUP_ALLOW_REMOTE", "")
	if remoteSetupEnabled() {
		t.Fatal("remote setup enabled by default")
	}
	t.Setenv("SETUP_ALLOW_REMOTE", "yes")
	if !remoteSetupEnabled() {
		t.Fatal("remote setup opt-in was ignored")
	}
}

func TestDecideAdminBootstrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalUsers int64
		adminUsers int64
		should     bool
		reason     string
	}{
		{
			name:       "empty database should create admin",
			totalUsers: 0,
			adminUsers: 0,
			should:     true,
			reason:     adminBootstrapReasonEmptyDatabase,
		},
		{
			name:       "admin exists should skip",
			totalUsers: 10,
			adminUsers: 1,
			should:     false,
			reason:     adminBootstrapReasonAdminExists,
		},
		{
			name:       "users exist without admin should skip",
			totalUsers: 5,
			adminUsers: 0,
			should:     false,
			reason:     adminBootstrapReasonUsersExistWithoutAdmin,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := decideAdminBootstrap(tc.totalUsers, tc.adminUsers)
			if got.shouldCreate != tc.should {
				t.Fatalf("shouldCreate=%v, want %v", got.shouldCreate, tc.should)
			}
			if got.reason != tc.reason {
				t.Fatalf("reason=%q, want %q", got.reason, tc.reason)
			}
		})
	}
}

func TestSetupDefaultAdminConcurrency(t *testing.T) {
	t.Run("simple mode admin uses higher concurrency", func(t *testing.T) {
		t.Setenv("RUN_MODE", "simple")
		if got := setupDefaultAdminConcurrency(); got != simpleModeAdminConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, simpleModeAdminConcurrency)
		}
	})

	t.Run("standard mode keeps existing default", func(t *testing.T) {
		t.Setenv("RUN_MODE", "standard")
		if got := setupDefaultAdminConcurrency(); got != defaultUserConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, defaultUserConcurrency)
		}
	})
}

func TestApplyAdminDefaults(t *testing.T) {
	admin := AdminConfig{}
	applyAdminDefaults(&admin)

	if admin.Email != defaultAdminLogin {
		t.Fatalf("Email=%q, want %q", admin.Email, defaultAdminLogin)
	}
	if admin.Password != "" {
		t.Fatalf("Password=%q, want empty password so first install must set one explicitly", admin.Password)
	}
}

func TestValidateInitialAdminRejectsMissingOrWeakPassword(t *testing.T) {
	tests := []AdminConfig{
		{Email: defaultAdminLogin},
		{Email: defaultAdminLogin, Password: "admin"},
		{Email: "invalid login", Password: "strong-password"},
	}

	for _, admin := range tests {
		if err := validateInitialAdmin(admin); err == nil {
			t.Fatalf("validateInitialAdmin(%+v) error = nil, want validation failure", admin)
		}
	}
}

func TestValidateInitialAdminAcceptsStrongPassword(t *testing.T) {
	admin := AdminConfig{Email: defaultAdminLogin, Password: "strong-password"}
	if err := validateInitialAdmin(admin); err != nil {
		t.Fatalf("validateInitialAdmin() error = %v", err)
	}
}

func TestNeedsSetupSkipsWhenSkipSetupIsEnabled(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "true", value: "true"},
		{name: "one", value: "1"},
		{name: "yes", value: "yes"},
		{name: "trimmed mixed case true", value: "  TrUe  "},
		{name: "trimmed mixed case yes", value: "  YeS  "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATA_DIR", t.TempDir())
			t.Setenv("SKIP_SETUP", tc.value)

			if NeedsSetup() {
				t.Fatalf("NeedsSetup() = true, want false when SKIP_SETUP is enabled")
			}
		})
	}
}

func TestNeedsSetupFallsBackToFileDetectionWhenSkipSetupIsDisabled(t *testing.T) {
	tests := []struct {
		name         string
		skipSetupSet bool
		skipSetup    string
		markerFile   string
		want         bool
	}{
		{
			name: "unset without installation files",
			want: true,
		},
		{
			name:         "false without installation files",
			skipSetupSet: true,
			skipSetup:    " false ",
			want:         true,
		},
		{
			name:         "invalid value without installation files",
			skipSetupSet: true,
			skipSetup:    "enabled",
			want:         true,
		},
		{
			name:         "config file exists",
			skipSetupSet: true,
			skipSetup:    "false",
			markerFile:   ConfigFileName,
			want:         false,
		},
		{
			name:         "install lock file exists",
			skipSetupSet: true,
			skipSetup:    "invalid",
			markerFile:   InstallLockFile,
			want:         false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dataDir := t.TempDir()
			t.Setenv("DATA_DIR", dataDir)
			if tc.skipSetupSet {
				t.Setenv("SKIP_SETUP", tc.skipSetup)
			} else {
				originalValue, wasSet := os.LookupEnv("SKIP_SETUP")
				if err := os.Unsetenv("SKIP_SETUP"); err != nil {
					t.Fatalf("Unsetenv(SKIP_SETUP) error = %v", err)
				}
				t.Cleanup(func() {
					if wasSet {
						_ = os.Setenv("SKIP_SETUP", originalValue)
						return
					}
					_ = os.Unsetenv("SKIP_SETUP")
				})
			}

			if tc.markerFile != "" {
				if err := os.WriteFile(filepath.Join(dataDir, tc.markerFile), nil, 0o600); err != nil {
					t.Fatalf("WriteFile(%s) error = %v", tc.markerFile, err)
				}
			}

			if got := NeedsSetup(); got != tc.want {
				t.Fatalf("NeedsSetup() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSetupMigrationTimeout(t *testing.T) {
	t.Run("uses default timeout when unset", func(t *testing.T) {
		cfg := &SetupConfig{}
		if got := cfg.migrationTimeout(); got != 60*time.Second {
			t.Fatalf("migrationTimeout()=%s, want 60s", got)
		}
	})

	t.Run("uses configured timeout", func(t *testing.T) {
		cfg := &SetupConfig{MigrationTimeoutSeconds: 300}
		if got := cfg.migrationTimeout(); got != 300*time.Second {
			t.Fatalf("migrationTimeout()=%s, want 300s", got)
		}
	})
}

func TestWriteConfigFileKeepsDefaultUserConcurrency(t *testing.T) {
	t.Setenv("RUN_MODE", "simple")
	t.Setenv("DATA_DIR", t.TempDir())

	if err := writeConfigFile(&SetupConfig{}); err != nil {
		t.Fatalf("writeConfigFile() error = %v", err)
	}

	data, err := os.ReadFile(GetConfigFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(data), "user_concurrency: 0") {
		t.Fatalf("config missing default user concurrency, got:\n%s", string(data))
	}
}

func TestWriteConfigFileIncludesRedisUsername(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())

	if err := writeConfigFile(&SetupConfig{
		Redis: RedisConfig{
			Host:     "redis",
			Port:     6379,
			Username: "app-user",
		},
	}); err != nil {
		t.Fatalf("writeConfigFile() error = %v", err)
	}

	data, err := os.ReadFile(GetConfigFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(data), "username: app-user") {
		t.Fatalf("config missing Redis username, got:\n%s", string(data))
	}
}

func TestBuildDatabaseConnectionDSNsUsesPostgresForBootstrap(t *testing.T) {
	cfg := &DatabaseConfig{
		Host:     "db",
		Port:     5432,
		User:     "sub3api",
		Password: "secret",
		DBName:   "sub3api",
		SSLMode:  "disable",
	}

	bootstrapDSN, targetDSN := buildDatabaseConnectionDSNs(cfg)

	if !strings.Contains(bootstrapDSN, "dbname=postgres") {
		t.Fatalf("bootstrap DSN = %q, want default postgres database", bootstrapDSN)
	}
	if strings.Contains(bootstrapDSN, "dbname=sub3api") {
		t.Fatalf("bootstrap DSN = %q, should not connect to target database before checking/creating it", bootstrapDSN)
	}
	if !strings.Contains(targetDSN, "dbname=sub3api") {
		t.Fatalf("target DSN = %q, want configured database", targetDSN)
	}
}
