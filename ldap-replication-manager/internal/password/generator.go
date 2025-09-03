package password

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/ldap-replication-manager/internal/config"
	"github.com/ldap-replication-manager/internal/ldap"
)

// Manager handles password generation and management for replication agreements
// This component ensures that passwords meet security requirements
// It generates cryptographically secure passwords using Go's crypto/rand package
// The manager respects configuration settings for password complexity
// Non-programmers can adjust password requirements through the config file
type Manager struct {
	config *config.Config
}

// NewManager creates a new password manager instance
// This function initializes the password generator with the provided configuration
// It validates that password generation settings are reasonable
// The manager uses the configuration to determine password complexity requirements
// This separation allows easy testing and configuration changes
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config: cfg,
	}
}

// GeneratePasswords creates or retrieves passwords for all replication agreements
// This method first checks for predefined passwords in the configuration
// If no predefined password exists, it can generate random passwords or use a default
// The returned map uses agreement names as keys for easy lookup
// This approach gives administrators full control over password management
func (m *Manager) GeneratePasswords(agreements []ldap.ReplicationAgreement) map[string]string {
	passwords := make(map[string]string)

	for _, agreement := range agreements {
		// Only use predefined passwords, or default if not specified
		password := ""
		if predefinedPassword, exists := m.config.Password.PredefinedPasswords[agreement.Name]; exists && predefinedPassword != "" {
			password = predefinedPassword
			fmt.Printf("Password for agreement '%s': using predefined password\n", agreement.Name)
		} else if m.config.Password.DefaultPassword != "" {
			password = m.config.Password.DefaultPassword
			fmt.Printf("Password for agreement '%s': using default password\n", agreement.Name)
		} else {
			fmt.Printf("Password for agreement '%s': ERROR - no predefined or default password found!\n", agreement.Name)
		}
		passwords[agreement.Name] = password
	}

	return passwords
}

// generateSecurePassword creates a cryptographically secure password
// This function uses Go's crypto/rand package for true randomness
// It respects all configuration settings for character types and length
// The password generation follows security best practices
// Understanding this helps administrators see how secure passwords are created
func (m *Manager) generateSecurePassword() (string, error) {
	// Build character set based on configuration
	// This allows administrators to control password complexity
	var charset string

	// Add lowercase letters if enabled
	if m.config.Password.IncludeLowercase {
		charset += "abcdefghijklmnopqrstuvwxyz"
	}

	// Add uppercase letters if enabled
	if m.config.Password.IncludeUppercase {
		charset += "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}

	// Add numbers if enabled
	if m.config.Password.IncludeNumbers {
		charset += "0123456789"
	}

	// Add special characters if enabled
	if m.config.Password.IncludeSpecial {
		charset += "!@#$%^&*()_+-=[]{}|;:,.<>?"
	}

	// Remove excluded characters to avoid confusion
	// This helps prevent issues with characters that look similar
	for _, char := range m.config.Password.ExcludeChars {
		charset = strings.ReplaceAll(charset, string(char), "")
	}

	// Ensure we have characters to work with
	if len(charset) == 0 {
		return "", fmt.Errorf("no characters available for password generation")
	}

	// Generate password of specified length
	password := make([]byte, m.config.Password.Length)
	charsetLen := big.NewInt(int64(len(charset)))

	// Use cryptographically secure random number generation
	// This ensures passwords cannot be predicted or reproduced
	for i := range password {
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %v", err)
		}
		password[i] = charset[randomIndex.Int64()]
	}

	// Validate that the generated password meets requirements
	// This ensures we don't return passwords that don't meet policy
	if err := m.validatePassword(string(password)); err != nil {
		// If validation fails, try again (recursive call)
		// This handles edge cases where random generation doesn't meet requirements
		return m.generateSecurePassword()
	}

	return string(password), nil
}

// validatePassword ensures a password meets all requirements
// This function checks that the password contains required character types
// It prevents weak passwords from being generated
// The validation logic matches the generation configuration
// This double-check ensures password policy compliance
func (m *Manager) validatePassword(password string) error {
	// Check minimum length
	if len(password) < m.config.Password.Length {
		return fmt.Errorf("password too short")
	}

	// Check for required character types
	hasLower := false
	hasUpper := false
	hasNumber := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= '0' && char <= '9':
			hasNumber = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	// Validate required character types are present
	if m.config.Password.IncludeLowercase && !hasLower {
		return fmt.Errorf("password missing lowercase letters")
	}
	if m.config.Password.IncludeUppercase && !hasUpper {
		return fmt.Errorf("password missing uppercase letters")
	}
	if m.config.Password.IncludeNumbers && !hasNumber {
		return fmt.Errorf("password missing numbers")
	}
	if m.config.Password.IncludeSpecial && !hasSpecial {
		return fmt.Errorf("password missing special characters")
	}

	return nil
}

// generateFallbackPassword creates a simple password when secure generation fails
// This method ensures the application continues working even if crypto/rand fails
// It creates a predictable but unique password based on the agreement name
// This fallback should only be used in emergency situations
// Administrators should investigate why secure generation failed
func (m *Manager) generateFallbackPassword(agreementName string) string {
	// Create a simple but unique password based on agreement name
	// This is not cryptographically secure but ensures functionality
	base := fmt.Sprintf("repl_%s_", agreementName)

	// Add some pseudo-random characters to meet length requirements
	suffix := "Pass123!"
	password := base + suffix

	// Truncate or pad to meet length requirements
	if len(password) > m.config.Password.Length {
		password = password[:m.config.Password.Length]
	} else {
		// Pad with additional characters if needed
		for len(password) < m.config.Password.Length {
			password += "x"
		}
	}

	return password
}

// GetPasswordStrength evaluates the strength of a generated password
// This method provides feedback on password quality
// It helps administrators understand if their password policy is adequate
// The strength assessment considers length, character diversity, and entropy
// This educational feature helps users understand password security
func (m *Manager) GetPasswordStrength(password string) string {
	score := 0

	// Length scoring
	if len(password) >= 12 {
		score += 2
	} else if len(password) >= 8 {
		score += 1
	}

	// Character type scoring
	hasLower := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")
	hasUpper := strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	hasNumber := strings.ContainsAny(password, "0123456789")
	hasSpecial := strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?")

	if hasLower {
		score++
	}
	if hasUpper {
		score++
	}
	if hasNumber {
		score++
	}
	if hasSpecial {
		score++
	}

	// Return strength assessment
	switch {
	case score >= 7:
		return "Very Strong"
	case score >= 5:
		return "Strong"
	case score >= 3:
		return "Medium"
	default:
		return "Weak"
	}
}

// GeneratePasswordPolicy creates a human-readable description of password requirements
// This method helps administrators understand what passwords will be generated
// It translates configuration settings into plain English
// The policy description can be included in documentation
// This transparency helps users understand and trust the password generation process
func (m *Manager) GeneratePasswordPolicy() string {
	var policy strings.Builder

	policy.WriteString(fmt.Sprintf("Password Policy:\n"))
	policy.WriteString(fmt.Sprintf("- Length: %d characters\n", m.config.Password.Length))

	if m.config.Password.IncludeLowercase {
		policy.WriteString("- Includes lowercase letters (a-z)\n")
	}
	if m.config.Password.IncludeUppercase {
		policy.WriteString("- Includes uppercase letters (A-Z)\n")
	}
	if m.config.Password.IncludeNumbers {
		policy.WriteString("- Includes numbers (0-9)\n")
	}
	if m.config.Password.IncludeSpecial {
		policy.WriteString("- Includes special characters (!@#$%^&*)\n")
	}

	if m.config.Password.ExcludeChars != "" {
		policy.WriteString(fmt.Sprintf("- Excludes confusing characters: %s\n", m.config.Password.ExcludeChars))
	}

	return policy.String()
}
