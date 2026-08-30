package setup

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/yuchen0x1/sub3api-relay/internal/config"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/logger"
	"github.com/yuchen0x1/sub3api-relay/internal/repository"
	"github.com/yuchen0x1/sub3api-relay/internal/service"

	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Config paths
const (
	ConfigFileName             = "config.yaml"
	InstallLockFile            = ".installed"
	AdminSetupPendingFile      = ".admin-setup-pending"
	defaultUserConcurrency     = 0
	simpleModeAdminConcurrency = 0
	defaultMigrationTimeout    = 60 * time.Second
	defaultAdminLogin          = "admin"
	defaultAdminPassword       = "admin"
)

// usableAdminWhereClause is shared by every bootstrap path. An administrator
// that is disabled, soft-deleted, or missing a password cannot complete the
// setup/login contract and must not suppress creation of the first usable
// site-owner account.
const usableAdminWhereClause = "role = $1 AND status = $2 AND deleted_at IS NULL AND password_hash <> ''"

func setupDefaultAdminConcurrency() int {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("RUN_MODE")), config.RunModeSimple) {
		return simpleModeAdminConcurrency
	}
	return defaultUserConcurrency
}

// GetDataDir returns the data directory for storing config and lock files.
// Priority: DATA_DIR env > /app/data (if exists and writable) > current directory
func GetDataDir() string {
	// Check DATA_DIR environment variable first
	if dir := os.Getenv("DATA_DIR"); dir != "" {
		return dir
	}

	// Check if /app/data exists and is writable (Docker environment)
	dockerDataDir := "/app/data"
	if info, err := os.Stat(dockerDataDir); err == nil && info.IsDir() {
		// Try to check if writable by creating a temp file
		testFile := dockerDataDir + "/.write_test"
		if f, err := os.Create(testFile); err == nil {
			_ = f.Close()
			_ = os.Remove(testFile)
			return dockerDataDir
		}
	}

	// Default to current directory
	return "."
}

// GetConfigFilePath returns the full path to config.yaml
func GetConfigFilePath() string {
	return GetDataDir() + "/" + ConfigFileName
}

// GetInstallLockPath returns the full path to .installed lock file
func GetInstallLockPath() string {
	return GetDataDir() + "/" + InstallLockFile
}

// GetAdminSetupPendingPath returns the marker used by automatic deployments
// that have initialized their infrastructure but still need the site owner to
// choose the first administrator credentials in the web setup page.
func GetAdminSetupPendingPath() string {
	return GetDataDir() + "/" + AdminSetupPendingFile
}

// AdminSetupPending reports whether automatic setup is waiting for the site
// owner to create the first administrator account.
func AdminSetupPending() bool {
	_, err := os.Stat(GetAdminSetupPendingPath())
	return err == nil
}

// AutoSetupOwnerSetupEnabled switches automatic deployments to the two-stage
// bootstrap flow.  Infrastructure (database, Redis, migrations and config)
// is prepared automatically, but the first administrator is created only
// after the site owner submits the web setup form.
func AutoSetupOwnerSetupEnabled() bool {
	for _, key := range []string{"AUTO_SETUP_OWNER_SETUP", "SETUP_MODE"} {
		value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		switch value {
		case "true", "1", "yes", "owner", "owner_setup", "docker-admin", "pending_admin":
			return true
		}
	}
	return false
}

func markAdminSetupPending() error {
	content := fmt.Sprintf("created_at=%s\n", time.Now().UTC().Format(time.RFC3339))
	file, err := os.OpenFile(GetAdminSetupPendingPath(), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			// Re-running automatic setup is expected after a restart. Preserve the
			// original marker timestamp and treat that case as success.
			return nil
		}
		return err
	}
	complete := false
	defer func() {
		_ = file.Close()
		if !complete {
			_ = os.Remove(GetAdminSetupPendingPath())
		}
	}()
	if _, err := file.WriteString(content); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	complete = true
	return nil
}

func clearAdminSetupPending() error {
	if err := os.Remove(GetAdminSetupPendingPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// SetupConfig holds the setup configuration
type SetupConfig struct {
	Database                DatabaseConfig `json:"database" yaml:"database"`
	Redis                   RedisConfig    `json:"redis" yaml:"redis"`
	Admin                   AdminConfig    `json:"admin" yaml:"-"` // Not stored in config file
	Server                  ServerConfig   `json:"server" yaml:"server"`
	JWT                     JWTConfig      `json:"jwt" yaml:"jwt"`
	Timezone                string         `json:"timezone" yaml:"timezone"` // e.g. "Asia/Shanghai", "UTC"
	MigrationTimeoutSeconds int            `json:"migration_timeout_seconds" yaml:"migration_timeout_seconds,omitempty"`
}

type DatabaseConfig struct {
	Host              string `json:"host" yaml:"host"`
	Port              int    `json:"port" yaml:"port"`
	User              string `json:"user" yaml:"user"`
	Password          string `json:"password" yaml:"password"`
	DBName            string `json:"dbname" yaml:"dbname"`
	SSLMode           string `json:"sslmode" yaml:"sslmode"`
	resolvedAddresses []netip.Addr
}

type RedisConfig struct {
	Host              string `json:"host" yaml:"host"`
	Port              int    `json:"port" yaml:"port"`
	Username          string `json:"username" yaml:"username"`
	Password          string `json:"password" yaml:"password"`
	DB                int    `json:"db" yaml:"db"`
	EnableTLS         bool   `json:"enable_tls" yaml:"enable_tls"`
	resolvedAddresses []netip.Addr
}

type AdminConfig struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ServerConfig struct {
	Host string `json:"host" yaml:"host"`
	Port int    `json:"port" yaml:"port"`
	Mode string `json:"mode" yaml:"mode"`
}

type JWTConfig struct {
	Secret     string `json:"secret" yaml:"secret"`
	ExpireHour int    `json:"expire_hour" yaml:"expire_hour"`
}

const (
	adminBootstrapReasonEmptyDatabase          = "empty_database"
	adminBootstrapReasonAdminExists            = "admin_exists"
	adminBootstrapReasonUsersExistWithoutAdmin = "users_exist_without_admin"
	adminBootstrapReasonLoginExists            = "login_exists"
)

// ErrAdminLoginAlreadyExists lets the setup handler turn a database unique
// constraint into an actionable response. This commonly occurs when an older
// deployment has a disabled (or soft-deleted on a pre-016 schema) user that
// still owns the requested login identifier.
var ErrAdminLoginAlreadyExists = errors.New("administrator login already exists")

type adminBootstrapDecision struct {
	shouldCreate bool
	reason       string
}

func decideAdminBootstrap(totalUsers, adminUsers int64) adminBootstrapDecision {
	if adminUsers > 0 {
		return adminBootstrapDecision{
			shouldCreate: false,
			reason:       adminBootstrapReasonAdminExists,
		}
	}
	if totalUsers > 0 {
		// Existing non-admin users do not provide a usable way to administer the
		// installation.  Bootstrap the configured administrator instead of
		// leaving a completed installation with no administrator account.  This
		// is safe because the adminUsers == 0 branch guarantees that no existing
		// administrator password can be overwritten.
		return adminBootstrapDecision{
			shouldCreate: true,
			reason:       adminBootstrapReasonUsersExistWithoutAdmin,
		}
	}
	return adminBootstrapDecision{
		shouldCreate: true,
		reason:       adminBootstrapReasonEmptyDatabase,
	}
}

func skipSetupEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SKIP_SETUP"))) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

// NeedsSetup checks if the system needs initial setup
// Uses multiple checks to prevent attackers from forcing re-setup by deleting config
func NeedsSetup() bool {
	// Automatic deployments may persist the generated config before the owner
	// chooses credentials. Keep the setup server available until that step is
	// completed, even though config.yaml already exists.
	if AdminSetupPending() {
		return true
	}
	if skipSetupEnabled() {
		logger.L().Debug("setup.needs_setup_bypassed", zap.String("reason", "skip_setup_enabled"))
		return false
	}

	// Recover installations created by older releases that wrote the
	// completion lock before inserting an administrator. In owner-setup Docker
	// mode, a lock is considered complete only when the database also contains
	// an administrator. If the health check cannot reach PostgreSQL, leave the
	// normal startup path in charge so a transient database outage is not
	// mistaken for a credential setup request.
	if AutoSetupEnabled() && AutoSetupOwnerSetupEnabled() {
		if _, err := os.Stat(GetInstallLockPath()); err == nil {
			if missing, err := ownerAdminMissingFromDatabase(); err == nil && missing {
				return true
			}
		}
	}

	// The installation lock is the authoritative completion marker. A config
	// file without the lock means setup was interrupted between writing config
	// and completing installation; keep the setup page available so the
	// operator can finish instead of silently starting an unbootstrapped app.
	if _, err := os.Stat(GetInstallLockPath()); !os.IsNotExist(err) {
		return false // Lock file exists, already installed
	}
	return true
}

func ownerAdminMissingFromDatabase() (bool, error) {
	cfg, _, err := loadAutoSetupConfig()
	if err != nil {
		logger.LegacyPrintf("setup", "unable to load generated config during startup: %v", err)
		return false, err
	}
	exists, err := adminUserExists(cfg)
	if err != nil {
		logger.LegacyPrintf("setup", "unable to verify existing administrator during startup: %v", err)
		return false, err
	}
	return !exists, nil
}

func buildPostgresDSN(cfg *DatabaseConfig, dbName string) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, dbName, cfg.SSLMode,
	)
}

func buildDatabaseConnectionDSNs(cfg *DatabaseConfig) (bootstrapDSN, targetDSN string) {
	return buildPostgresDSN(cfg, "postgres"), buildPostgresDSN(cfg, cfg.DBName)
}

type setupDialer struct {
	addresses []netip.Addr
	timeout   time.Duration
}

func (d setupDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialTimeout(network, address, d.timeout)
}

func (d setupDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return d.DialContext(ctx, network, address)
}

func (d setupDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{Timeout: d.timeout}
	var lastErr error
	for _, target := range d.addresses {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(target.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no approved target addresses")
	}
	return nil, lastErr
}

func openSetupPostgres(dsn string, addresses []netip.Addr) (*sql.DB, error) {
	if len(addresses) == 0 {
		return sql.Open("postgres", dsn)
	}
	connector, err := pq.NewConnector(dsn)
	if err != nil {
		return nil, err
	}
	connector.Dialer(setupDialer{addresses: addresses, timeout: 5 * time.Second})
	return sql.OpenDB(connector), nil
}

// TestDatabaseConnection tests the database connection and creates database if not exists
func TestDatabaseConnection(cfg *DatabaseConfig) error {
	// First, connect to the default 'postgres' database to check/create target database.
	// Connecting to cfg.DBName here fails when the target database has not been
	// created yet, so the bootstrap connection must use PostgreSQL's maintenance DB.
	defaultDSN, targetDSN := buildDatabaseConnectionDSNs(cfg)

	db, err := openSetupPostgres(defaultDSN, cfg.resolvedAddresses)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	defer func() {
		if db == nil {
			return
		}
		if err := db.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close postgres connection: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Check if target database exists
	var exists bool
	row := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", cfg.DBName)
	if err := row.Scan(&exists); err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	// Create database if not exists
	if !exists {
		// 注意：数据库名不能参数化，依赖前置输入校验保障安全。
		// Note: Database names cannot be parameterized, but we've already validated cfg.DBName
		// in the handler using validateDBName() which only allows [a-zA-Z][a-zA-Z0-9_]*
		_, err := db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", cfg.DBName))
		if err != nil {
			return fmt.Errorf("failed to create database '%s': %w", cfg.DBName, err)
		}
		logger.LegacyPrintf("setup", "Database '%s' created successfully", cfg.DBName)
	}

	// Now connect to the target database to verify
	if err := db.Close(); err != nil {
		logger.LegacyPrintf("setup", "failed to close postgres connection: %v", err)
	}
	db = nil

	targetDB, err := openSetupPostgres(targetDSN, cfg.resolvedAddresses)
	if err != nil {
		return fmt.Errorf("failed to connect to database '%s': %w", cfg.DBName, err)
	}

	defer func() {
		if err := targetDB.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close postgres connection: %v", err)
		}
	}()

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	if err := targetDB.PingContext(ctx2); err != nil {
		return fmt.Errorf("ping target database failed: %w", err)
	}

	return nil
}

// TestRedisConnection tests the Redis connection
func TestRedisConnection(cfg *RedisConfig) error {
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
	if len(cfg.resolvedAddresses) > 0 {
		dialer := setupDialer{addresses: cfg.resolvedAddresses, timeout: 5 * time.Second}
		opts.Dialer = func(ctx context.Context, network, address string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, network, address)
			if err != nil || !cfg.EnableTLS {
				return conn, err
			}
			tlsConn := tls.Client(conn, opts.TLSConfig)
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				_ = conn.Close()
				return nil, err
			}
			return tlsConn, nil
		}
	}

	if cfg.EnableTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: cfg.Host,
		}
	}

	rdb := redis.NewClient(opts)
	defer func() {
		if err := rdb.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close redis client: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

// Install performs the installation with the given configuration
func Install(cfg *SetupConfig) error {
	if cfg == nil {
		return fmt.Errorf("setup config is nil")
	}
	// Security check: prevent re-installation if already installed
	if !NeedsSetup() {
		return fmt.Errorf("system is already installed, re-installation is not allowed")
	}
	if err := validateOwnerAdmin(cfg.Admin); err != nil {
		return fmt.Errorf("invalid initial administrator: %w", err)
	}

	// Generate JWT secret if not provided
	if cfg.JWT.Secret == "" {
		secret, err := generateSecret(32)
		if err != nil {
			return fmt.Errorf("failed to generate jwt secret: %w", err)
		}
		cfg.JWT.Secret = secret
		logger.LegacyPrintf("setup", "%s", "Warning: JWT secret auto-generated. Consider setting a fixed secret for production.")
	}

	// Test connections
	if err := TestDatabaseConnection(&cfg.Database); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	if err := TestRedisConnection(&cfg.Redis); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}

	// Initialize database
	if err := initializeDatabase(cfg); err != nil {
		return fmt.Errorf("database initialization failed: %w", err)
	}

	// Create the configured admin when no administrator exists. Existing
	// non-admin users are preserved; the bootstrap insert only adds access.
	if _, _, err := createAdminUser(cfg); err != nil {
		return fmt.Errorf("admin user creation failed: %w", err)
	}

	// Write config file
	if err := writeConfigFile(cfg); err != nil {
		return fmt.Errorf("config file creation failed: %w", err)
	}

	// Create installation lock file to prevent re-setup attacks
	if err := createInstallLock(); err != nil {
		return fmt.Errorf("failed to create install lock: %w", err)
	}

	return nil
}

// createInstallLock creates a lock file to prevent re-installation attacks
func createInstallLock() error {
	content := fmt.Sprintf("installed_at=%s\n", time.Now().UTC().Format(time.RFC3339))
	path := GetInstallLockPath()
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0400)
	if err != nil {
		if os.IsExist(err) {
			// A previous process may have completed the final step already. The
			// lock is intentionally immutable, so do not truncate or rewrite it.
			existing, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if validInstallLock(existing) {
				return nil
			}
			return fmt.Errorf("installation lock already exists but is invalid")
		}
		return err
	}
	complete := false
	defer func() {
		_ = file.Close()
		if !complete {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.WriteString(content); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	complete = true
	return nil
}

func validInstallLock(content []byte) bool {
	line := strings.TrimSpace(string(content))
	const prefix = "installed_at="
	if !strings.HasPrefix(line, prefix) {
		return false
	}
	value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	if value == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}

func initializeDatabase(cfg *SetupConfig) error {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode,
	)

	db, err := openSetupPostgres(dsn, cfg.Database.resolvedAddresses)
	if err != nil {
		return err
	}

	defer func() {
		if err := db.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close postgres connection: %v", err)
		}
	}()

	migrationCtx, cancel := context.WithTimeout(context.Background(), cfg.migrationTimeout())
	defer cancel()
	return repository.ApplyMigrations(migrationCtx, db)
}

func (cfg *SetupConfig) migrationTimeout() time.Duration {
	if cfg != nil && cfg.MigrationTimeoutSeconds > 0 {
		return time.Duration(cfg.MigrationTimeoutSeconds) * time.Second
	}
	return defaultMigrationTimeout
}

func createAdminUser(cfg *SetupConfig) (bool, string, error) {
	if cfg == nil {
		return false, "", fmt.Errorf("setup config is nil")
	}
	applyAdminDefaults(&cfg.Admin)
	return createAdminUserWithConfiguredCredentials(cfg)
}

func createAdminUserWithConfiguredCredentials(cfg *SetupConfig) (bool, string, error) {
	if cfg == nil {
		return false, "", fmt.Errorf("setup config is nil")
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode,
	)

	db, err := openSetupPostgres(dsn, cfg.Database.resolvedAddresses)
	if err != nil {
		return false, "", err
	}

	defer func() {
		if err := db.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close postgres connection: %v", err)
		}
	}()

	// 使用超时上下文避免安装流程因数据库异常而长时间阻塞。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// The setup HTTP handler is process-serialized, but automatic deployments
	// can be restarted or accidentally launched more than once.  Serialize the
	// bootstrap decision in PostgreSQL as well so two instances cannot both see
	// an empty users table and create different administrators.
	const initialAdminAdvisoryLockKey int64 = 8675309
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, "", err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", initialAdminAdvisoryLockKey); err != nil {
		return false, "", err
	}

	var totalUsers int64
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM users WHERE deleted_at IS NULL").Scan(&totalUsers); err != nil {
		return false, "", err
	}
	var adminUsers int64
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM users WHERE "+usableAdminWhereClause, service.RoleAdmin, service.StatusActive).Scan(&adminUsers); err != nil {
		return false, "", err
	}
	decision := decideAdminBootstrap(totalUsers, adminUsers)
	if !decision.shouldCreate {
		return false, decision.reason, nil
	}
	if err := validateInitialAdmin(cfg.Admin); err != nil {
		return false, "", err
	}
	// Do not silently overwrite or collide with an existing user when the
	// deployment has no usable administrator. Older schemas may keep a unique
	// constraint even for soft-deleted rows, so inspect all rows here rather
	// than relying only on the current partial unique index.
	var loginUsers int64
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM users WHERE email = $1", cfg.Admin.Email).Scan(&loginUsers); err != nil {
		return false, "", err
	}
	if loginUsers > 0 {
		return false, adminBootstrapReasonLoginExists, fmt.Errorf("%w: %s", ErrAdminLoginAlreadyExists, cfg.Admin.Email)
	}

	admin := &service.User{
		Email:       cfg.Admin.Email,
		Role:        service.RoleAdmin,
		Status:      service.StatusActive,
		Balance:     0,
		Concurrency: setupDefaultAdminConcurrency(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := admin.SetPassword(cfg.Admin.Password); err != nil {
		return false, "", err
	}

	if _, err = tx.ExecContext(
		ctx,
		`INSERT INTO users (email, password_hash, role, balance, concurrency, status, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		admin.Email,
		admin.PasswordHash,
		admin.Role,
		admin.Balance,
		admin.Concurrency,
		admin.Status,
		admin.CreatedAt,
		admin.UpdatedAt,
	); err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return false, adminBootstrapReasonLoginExists, fmt.Errorf("%w: %s", ErrAdminLoginAlreadyExists, cfg.Admin.Email)
		}
		return false, "", err
	}
	if err := tx.Commit(); err != nil {
		return false, "", err
	}
	return true, decision.reason, nil
}

func applyAdminDefaults(admin *AdminConfig) {
	if admin == nil {
		return
	}
	admin.Email = strings.TrimSpace(admin.Email)
	if admin.Email == "" {
		admin.Email = defaultAdminLogin
	}
	if admin.Password == "" {
		admin.Password = defaultAdminPassword
	}
}

func validateInitialAdmin(admin AdminConfig) error {
	if !validateAdminIdentifier(admin.Email) {
		return fmt.Errorf("invalid admin login format")
	}
	if err := validatePassword(admin.Password); err != nil {
		return fmt.Errorf("invalid initial admin password: %w", err)
	}
	return nil
}

// validateOwnerAdmin is stricter than the legacy bootstrap validator: a new
// web deployment must not silently create the publicly known admin/admin
// credential pair.
func validateOwnerAdmin(admin AdminConfig) error {
	if err := validateInitialAdmin(admin); err != nil {
		return err
	}
	if admin.Password == defaultAdminPassword {
		return fmt.Errorf("choose a password other than the default admin password")
	}
	return nil
}

func marshalConfigFile(cfg *SetupConfig) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("setup config is nil")
	}
	// Ensure timezone has a default value
	tz := cfg.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}

	// Prepare config for YAML (exclude sensitive data and admin config)
	yamlConfig := struct {
		Server   ServerConfig   `yaml:"server"`
		Database DatabaseConfig `yaml:"database"`
		Redis    RedisConfig    `yaml:"redis"`
		JWT      struct {
			Secret     string `yaml:"secret"`
			ExpireHour int    `yaml:"expire_hour"`
		} `yaml:"jwt"`
		Default struct {
			UserConcurrency int     `yaml:"user_concurrency"`
			UserBalance     float64 `yaml:"user_balance"`
			APIKeyPrefix    string  `yaml:"api_key_prefix"`
			RateMultiplier  float64 `yaml:"rate_multiplier"`
		} `yaml:"default"`
		RateLimit struct {
			RequestsPerMinute int `yaml:"requests_per_minute"`
			BurstSize         int `yaml:"burst_size"`
		} `yaml:"rate_limit"`
		Timezone string `yaml:"timezone"`
	}{
		Server:   cfg.Server,
		Database: cfg.Database,
		Redis:    cfg.Redis,
		JWT: struct {
			Secret     string `yaml:"secret"`
			ExpireHour int    `yaml:"expire_hour"`
		}{
			Secret:     cfg.JWT.Secret,
			ExpireHour: cfg.JWT.ExpireHour,
		},
		Default: struct {
			UserConcurrency int     `yaml:"user_concurrency"`
			UserBalance     float64 `yaml:"user_balance"`
			APIKeyPrefix    string  `yaml:"api_key_prefix"`
			RateMultiplier  float64 `yaml:"rate_multiplier"`
		}{
			UserConcurrency: defaultUserConcurrency,
			UserBalance:     0,
			APIKeyPrefix:    "sk-",
			RateMultiplier:  1.0,
		},
		RateLimit: struct {
			RequestsPerMinute int `yaml:"requests_per_minute"`
			BurstSize         int `yaml:"burst_size"`
		}{
			RequestsPerMinute: 60,
			BurstSize:         10,
		},
		Timezone: tz,
	}

	data, err := yaml.Marshal(&yamlConfig)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func writeConfigFile(cfg *SetupConfig) error {
	data, err := marshalConfigFile(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(GetConfigFilePath(), data, 0600)
}

// createConfigFileIfMissing publishes a generated config atomically. It is
// used by automatic setup so two app processes racing on first start cannot
// overwrite one another's JWT secret or database coordinates.
func createConfigFileIfMissing(cfg *SetupConfig) (bool, error) {
	data, err := marshalConfigFile(cfg)
	if err != nil {
		return false, err
	}
	path := GetConfigFilePath()
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, err
	}
	complete := false
	defer func() {
		_ = file.Close()
		if !complete {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return false, err
	}
	if err := file.Sync(); err != nil {
		return false, err
	}
	complete = true
	return true, nil
}

func generateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// =============================================================================
// Auto Setup for Docker Deployment
// =============================================================================

// AutoSetupEnabled checks if auto setup is enabled via environment variable
func AutoSetupEnabled() bool {
	val := os.Getenv("AUTO_SETUP")
	return val == "true" || val == "1" || val == "yes"
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// getEnvIntOrDefault gets environment variable as int or returns default value
func getEnvIntOrDefault(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}

// autoSetupConfigFromEnvironment builds the infrastructure configuration used
// by a fresh automatic deployment. Administrator credentials are intentionally
// not defaulted here: owner-setup deployments collect them from the web wizard.
func autoSetupConfigFromEnvironment() *SetupConfig {
	tz := getEnvOrDefault("TZ", "")
	if tz == "" {
		tz = getEnvOrDefault("TIMEZONE", "Asia/Shanghai")
	}
	return &SetupConfig{
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("DATABASE_HOST", "localhost"),
			Port:     getEnvIntOrDefault("DATABASE_PORT", 5432),
			User:     getEnvOrDefault("DATABASE_USER", "postgres"),
			Password: getEnvOrDefault("DATABASE_PASSWORD", ""),
			DBName:   getEnvOrDefault("DATABASE_DBNAME", "sub3api"),
			SSLMode:  getEnvOrDefault("DATABASE_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:      getEnvOrDefault("REDIS_HOST", "localhost"),
			Port:      getEnvIntOrDefault("REDIS_PORT", 6379),
			Username:  getEnvOrDefault("REDIS_USERNAME", ""),
			Password:  getEnvOrDefault("REDIS_PASSWORD", ""),
			DB:        getEnvIntOrDefault("REDIS_DB", 0),
			EnableTLS: strings.EqualFold(getEnvOrDefault("REDIS_ENABLE_TLS", "false"), "true"),
		},
		Admin: AdminConfig{
			Email:    strings.TrimSpace(os.Getenv("ADMIN_EMAIL")),
			Password: os.Getenv("ADMIN_PASSWORD"),
		},
		Server: ServerConfig{
			Host: getEnvOrDefault("SERVER_HOST", "0.0.0.0"),
			Port: getEnvIntOrDefault("SERVER_PORT", 8080),
			Mode: getEnvOrDefault("SERVER_MODE", "release"),
		},
		JWT: JWTConfig{
			Secret:     getEnvOrDefault("JWT_SECRET", ""),
			ExpireHour: getEnvIntOrDefault("JWT_EXPIRE_HOUR", 24),
		},
		Timezone:                tz,
		MigrationTimeoutSeconds: getEnvIntOrDefault("SETUP_MIGRATION_TIMEOUT_SECONDS", 0),
	}
}

// autoSetupConfigFromEnv is retained for callers/tests that need a best-effort
// view of the effective setup configuration. Startup paths use
// loadAutoSetupConfig instead so malformed persisted state is reported rather
// than silently replaced with environment values.
func autoSetupConfigFromEnv() *SetupConfig {
	if cfg, _, err := loadAutoSetupConfig(); err == nil {
		return cfg
	}
	return autoSetupConfigFromEnvironment()
}

// loadAutoSetupConfig returns the persisted infrastructure configuration when
// one exists. Once automatic setup has written config.yaml, that file is the
// source of truth for owner completion; environment changes must not redirect
// the request to a different database or regenerate the JWT secret.
func loadAutoSetupConfig() (*SetupConfig, bool, error) {
	data, err := os.ReadFile(GetConfigFilePath())
	if os.IsNotExist(err) {
		return autoSetupConfigFromEnvironment(), false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read generated config: %w", err)
	}
	var cfg SetupConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, true, fmt.Errorf("parse generated config: %w", err)
	}
	if strings.TrimSpace(cfg.Database.Host) == "" || cfg.Database.Port <= 0 ||
		strings.TrimSpace(cfg.Database.User) == "" || strings.TrimSpace(cfg.Database.DBName) == "" ||
		strings.TrimSpace(cfg.Redis.Host) == "" || cfg.Redis.Port <= 0 {
		return nil, true, fmt.Errorf("generated config is incomplete")
	}
	return &cfg, true, nil
}

// adminUserExists reports whether the configured database already contains a
// usable administrator. It deliberately does not inspect or return passwords.
func adminUserExists(cfg *SetupConfig) (bool, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode,
	)
	db, err := openSetupPostgres(dsn, cfg.Database.resolvedAddresses)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = db.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var count int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(1) FROM users WHERE "+usableAdminWhereClause, service.RoleAdmin, service.StatusActive).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// CompleteAdminSetup creates the one initial site-owner account for an
// automatic deployment. The caller must hold installMutex and the pending
// marker must still exist. It is intentionally separate from Install so the
// browser never receives database or Redis credentials.
func CompleteAdminSetup(admin AdminConfig) error {
	if !AdminSetupPending() {
		return fmt.Errorf("owner setup is not pending")
	}

	admin.Email = strings.TrimSpace(admin.Email)
	if err := validateOwnerAdmin(admin); err != nil {
		return err
	}

	cfg, configPersisted, err := loadAutoSetupConfig()
	if err != nil {
		return err
	}
	if !configPersisted {
		// A crash may have happened after the pending marker was created but
		// before config.yaml was written. Recreate it from the same env config.
		if cfg.JWT.Secret == "" {
			secret, generateErr := generateSecret(32)
			if generateErr != nil {
				return fmt.Errorf("generate jwt secret: %w", generateErr)
			}
			cfg.JWT.Secret = secret
		}
		created, writeErr := createConfigFileIfMissing(cfg)
		if writeErr != nil {
			return fmt.Errorf("write generated config: %w", writeErr)
		}
		if !created {
			cfg, _, err = loadAutoSetupConfig()
			if err != nil {
				return err
			}
		}
	}
	cfg.Admin = admin

	created, reason, err := createAdminUserWithConfiguredCredentials(cfg)
	if err != nil {
		return fmt.Errorf("admin user creation failed: %w", err)
	}
	if !created && reason != adminBootstrapReasonAdminExists {
		return fmt.Errorf("admin setup was not completed: %s", reason)
	}
	if err := createInstallLock(); err != nil {
		return fmt.Errorf("create install lock: %w", err)
	}
	if err := clearAdminSetupPending(); err != nil {
		return fmt.Errorf("clear owner setup marker: %w", err)
	}
	return nil
}

// AutoSetupFromEnv performs automatic setup using environment variables.
// This is designed for Docker deployment where all infrastructure config is
// passed via env vars. In owner-setup mode it stops before creating an admin
// and leaves a durable pending marker for the web wizard.
func AutoSetupFromEnv() error {
	logger.LegacyPrintf("setup", "%s", "Auto setup enabled, configuring from environment variables...")
	logger.LegacyPrintf("setup", "Data directory: %s", GetDataDir())
	cfg, configPersisted, err := loadAutoSetupConfig()
	if err != nil {
		return err
	}

	// Generate JWT secret if not provided
	if cfg.JWT.Secret == "" {
		secret, generateErr := generateSecret(32)
		if generateErr != nil {
			return fmt.Errorf("failed to generate jwt secret: %w", generateErr)
		}
		cfg.JWT.Secret = secret
		logger.LegacyPrintf("setup", "%s", "Warning: JWT secret auto-generated. Consider setting a fixed secret for production.")
	}

	// Test database connection
	logger.LegacyPrintf("setup", "%s", "Testing database connection...")
	if err := TestDatabaseConnection(&cfg.Database); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Database connection successful")

	// Test Redis connection
	logger.LegacyPrintf("setup", "%s", "Testing Redis connection...")
	if err := TestRedisConnection(&cfg.Redis); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Redis connection successful")

	// Initialize database
	logger.LegacyPrintf("setup", "%s", "Initializing database...")
	if err := initializeDatabase(cfg); err != nil {
		return fmt.Errorf("database initialization failed: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Database initialized successfully")

	ownerSetup := AutoSetupOwnerSetupEnabled()
	var adminExists bool
	if ownerSetup {
		// Do not consume ADMIN_EMAIL/ADMIN_PASSWORD in owner mode. Existing
		// users are preserved and the site owner chooses the sole bootstrap
		// administrator in the browser.
		var err error
		adminExists, err = adminUserExists(cfg)
		if err != nil {
			return fmt.Errorf("check existing administrator: %w", err)
		}
	}

	// Persist generated infrastructure configuration once. A config file that
	// already exists is authoritative and must not be overwritten during a
	// retry, because it contains the original database coordinates and JWT key.
	if !configPersisted {
		logger.LegacyPrintf("setup", "%s", "Writing configuration file...")
		created, err := createConfigFileIfMissing(cfg)
		if err != nil {
			return fmt.Errorf("config file creation failed: %w", err)
		}
		if created {
			logger.LegacyPrintf("setup", "%s", "Configuration file created")
		} else {
			logger.LegacyPrintf("setup", "%s", "Configuration file was created by another setup process; preserving it")
		}
	} else {
		logger.LegacyPrintf("setup", "%s", "Existing configuration file preserved")
	}

	if ownerSetup {
		if adminExists {
			logger.LegacyPrintf("setup", "%s", "Administrator already exists; completing automatic setup")
			if err := createInstallLock(); err != nil {
				return fmt.Errorf("failed to create install lock: %w", err)
			}
			_ = clearAdminSetupPending()
			return nil
		}
		if err := markAdminSetupPending(); err != nil {
			return fmt.Errorf("failed to create owner setup marker: %w", err)
		}
		logger.LegacyPrintf("setup", "%s", "Infrastructure is ready; waiting for site owner credentials at /setup")
		return nil
	}

	// Legacy automatic mode keeps its backwards-compatible env bootstrap.
	// It is retained only for explicitly configured legacy deployments. Never
	// synthesize the publicly known admin/admin pair when an operator omitted
	// credentials; new Compose deployments use owner-setup mode instead.
	if strings.TrimSpace(cfg.Admin.Email) == "" || cfg.Admin.Password == "" {
		return fmt.Errorf("ADMIN_EMAIL and ADMIN_PASSWORD are required when AUTO_SETUP_OWNER_SETUP is disabled")
	}
	if err := validateOwnerAdmin(cfg.Admin); err != nil {
		return fmt.Errorf("invalid configured administrator: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Creating admin user...")
	created, reason, err := createAdminUser(cfg)
	if err != nil {
		return fmt.Errorf("admin user creation failed: %w", err)
	}
	if created {
		logger.LegacyPrintf("setup", "Admin user created: %s", cfg.Admin.Email)
	} else {
		switch reason {
		case adminBootstrapReasonAdminExists:
			logger.LegacyPrintf("setup", "%s", "Admin user already exists, skipping admin bootstrap")
		case adminBootstrapReasonUsersExistWithoutAdmin:
			logger.LegacyPrintf("setup", "%s", "Existing users found without an administrator; configured admin bootstrap completed")
		default:
			logger.LegacyPrintf("setup", "%s", "Admin bootstrap skipped")
		}
	}

	// Create installation lock file
	if err := createInstallLock(); err != nil {
		return fmt.Errorf("failed to create install lock: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Installation lock created")

	logger.LegacyPrintf("setup", "%s", "Auto setup completed successfully!")
	return nil
}
