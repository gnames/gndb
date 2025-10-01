package contracts

// ConfigLoader loads and validates configuration
type ConfigLoader interface {
	// Load reads configuration from file and flags
	Load(configPath string) (*Config, error)

	// Validate validates configuration completeness
	Validate(cfg *Config) error
}

// Config represents gndb configuration
type Config struct {
	Database   DatabaseConfig
	SFGA       SFGAConfig
	Migration  MigrationConfig
	Logging    LoggingConfig
}

// DatabaseConfig holds PostgreSQL connection settings
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	MaxConns int
}

// SFGAConfig holds SFGA data source settings
type SFGAConfig struct {
	DataDir string
	Version string
	Sources []string
}

// MigrationConfig holds Atlas migration settings
type MigrationConfig struct {
	Dir     string
	DevURL  string
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level  string
	Format string
}
