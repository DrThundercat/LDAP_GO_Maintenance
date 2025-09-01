package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config holds all configuration for the LDAP replication manager
// This structure makes it easy for non-programmers to understand what can be configured
// Each field has a clear purpose and sensible defaults
// The YAML format is human-readable and easy to modify without programming knowledge
// Comments in the YAML file will guide users on how to customize settings
type Config struct {
	// LDAP server connection settings
	LDAP LDAPConfig `yaml:"ldap"`

	// Password generation settings
	Password PasswordConfig `yaml:"password"`

	// GRPC monitoring settings for error 49 detection
	GRPC GRPCConfig `yaml:"grpc"`

	// Logging and operational settings
	Logging LoggingConfig `yaml:"logging"`
}

// LDAPConfig contains all LDAP connection and operation settings
// These settings control how the application connects to your 389DS servers
// Modify these values to match your LDAP infrastructure
type LDAPConfig struct {
	// Primary LDAP server (usually your supplier/hub)
	Host string `yaml:"host"`
	Port int    `yaml:"port"`

	// Authentication credentials for LDAP operations
	// Use a service account with replication management privileges
	BindDN   string `yaml:"bind_dn"`
	Password string `yaml:"password"`

	// Base DN where replication agreements are stored
	// Typically: cn=config for 389DS
	BaseDN string `yaml:"base_dn"`

	// TLS/SSL settings for secure connections
	UseTLS        bool `yaml:"use_tls"`
	SkipTLSVerify bool `yaml:"skip_tls_verify"`

	// Connection timeout in seconds
	Timeout int `yaml:"timeout"`
}

// PasswordConfig controls how new passwords are generated or specified
// These settings ensure passwords meet your security requirements
// You can either generate random passwords or specify predefined ones
// Adjust complexity and length based on your organization's policies
type PasswordConfig struct {
	// Length of generated passwords (minimum 12 recommended)
	// Only used when generating random passwords
	Length int `yaml:"length"`

	// Include uppercase letters (A-Z)
	IncludeUppercase bool `yaml:"include_uppercase"`

	// Include lowercase letters (a-z)
	IncludeLowercase bool `yaml:"include_lowercase"`

	// Include numbers (0-9)
	IncludeNumbers bool `yaml:"include_numbers"`

	// Include special characters (!@#$%^&*)
	IncludeSpecial bool `yaml:"include_special"`

	// Characters to exclude from passwords (to avoid confusion)
	ExcludeChars string `yaml:"exclude_chars"`

	// Predefined passwords for specific replication agreements
	// If specified, these passwords will be used instead of generating random ones
	// Format: agreement_name: password
	// Example:
	//   agreement-to-consumer1: "MySecurePassword123!"
	//   agreement-to-consumer2: "AnotherSecurePass456@"
	PredefinedPasswords map[string]string `yaml:"predefined_passwords"`

	// Default password to use for all agreements if no specific password is defined
	// If empty, random passwords will be generated
	// This is useful when you want all agreements to use the same password
	DefaultPassword string `yaml:"default_password"`

	// Whether to generate random passwords when no predefined password is available
	// If false and no predefined/default password exists, the operation will fail
	GenerateRandom bool `yaml:"generate_random"`
}

// GRPCConfig settings for real-time error monitoring
// This enables the application to detect error 49 events as they happen
// GRPC provides efficient, real-time communication for monitoring
type GRPCConfig struct {
	// Enable GRPC monitoring
	Enabled bool `yaml:"enabled"`

	// Port for GRPC server to listen on
	Port int `yaml:"port"`

	// Log file paths to monitor for error 49
	LogPaths []string `yaml:"log_paths"`

	// How often to check log files (in seconds)
	CheckInterval int `yaml:"check_interval"`
}

// LoggingConfig controls application logging behavior
// Proper logging helps with troubleshooting and audit trails
type LoggingConfig struct {
	// Log level: debug, info, warn, error
	Level string `yaml:"level"`

	// Log file path (empty means stdout only)
	File string `yaml:"file"`

	// Enable timestamps in log messages
	Timestamps bool `yaml:"timestamps"`
}

// Load reads configuration from a YAML file
// This function handles file reading and YAML parsing
// It provides clear error messages to help users fix configuration issues
// The function validates required settings and provides sensible defaults
// Non-programmers can easily modify the YAML file without touching code
func Load(filename string) (*Config, error) {
	// Read the configuration file from disk
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %v", filename, err)
	}

	// Parse YAML content into our configuration structure
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %v", err)
	}

	// Apply default values for any missing settings
	// This ensures the application works even with minimal configuration
	setDefaults(&config)

	// Validate that required settings are present
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	return &config, nil
}

// setDefaults applies sensible default values for missing configuration
// This makes the application easier to use by requiring minimal configuration
// Defaults are chosen based on common 389DS deployment patterns
func setDefaults(config *Config) {
	// LDAP defaults
	if config.LDAP.Port == 0 {
		config.LDAP.Port = 389 // Standard LDAP port
	}
	if config.LDAP.BaseDN == "" {
		config.LDAP.BaseDN = "cn=config" // Standard 389DS config location
	}
	if config.LDAP.Timeout == 0 {
		config.LDAP.Timeout = 30 // 30 second timeout
	}

	// Password generation defaults
	if config.Password.Length == 0 {
		config.Password.Length = 16 // Strong password length
	}
	// Enable all character types by default for strong passwords
	config.Password.IncludeUppercase = true
	config.Password.IncludeLowercase = true
	config.Password.IncludeNumbers = true
	config.Password.IncludeSpecial = true

	// Exclude confusing characters by default
	if config.Password.ExcludeChars == "" {
		config.Password.ExcludeChars = "0O1lI" // Avoid look-alike characters
	}

	// Initialize predefined passwords map if nil
	if config.Password.PredefinedPasswords == nil {
		config.Password.PredefinedPasswords = make(map[string]string)
	}

	// Enable random generation by default if no other password source is configured
	if config.Password.DefaultPassword == "" && len(config.Password.PredefinedPasswords) == 0 {
		config.Password.GenerateRandom = true
	}

	// GRPC defaults
	if config.GRPC.Port == 0 {
		config.GRPC.Port = 50051 // Standard GRPC port
	}
	if config.GRPC.CheckInterval == 0 {
		config.GRPC.CheckInterval = 5 // Check every 5 seconds
	}
	// Default log paths for RHEL 389DS
	if len(config.GRPC.LogPaths) == 0 {
		config.GRPC.LogPaths = []string{
			"/var/log/dirsrv/slapd-ldap/errors",
			"/var/log/dirsrv/slapd-ldap/access",
		}
	}

	// Logging defaults
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	config.Logging.Timestamps = true
}

// validate ensures required configuration values are present
// This prevents runtime errors by catching configuration problems early
// Clear error messages help users fix their configuration files
func validate(config *Config) error {
	// Validate required LDAP settings
	if config.LDAP.Host == "" {
		return fmt.Errorf("LDAP host is required")
	}
	if config.LDAP.BindDN == "" {
		return fmt.Errorf("LDAP bind DN is required")
	}
	if config.LDAP.Password == "" {
		return fmt.Errorf("LDAP password is required")
	}

	// Validate password settings
	if config.Password.Length < 8 {
		return fmt.Errorf("password length must be at least 8 characters")
	}

	// Validate GRPC settings if enabled
	if config.GRPC.Enabled {
		if config.GRPC.Port < 1 || config.GRPC.Port > 65535 {
			return fmt.Errorf("GRPC port must be between 1 and 65535")
		}
	}

	return nil
}
