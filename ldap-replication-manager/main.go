package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ldap-replication-manager/internal/config"
	"github.com/ldap-replication-manager/internal/ldap"
	"github.com/ldap-replication-manager/internal/monitor"
	"github.com/ldap-replication-manager/internal/password"
)

// main is the entry point of the 389DS LDAP Replication Password Manager
// This application helps RHEL administrators manage replication agreement passwords
// It can detect error 49 (authentication failures) and update passwords automatically
// The program follows the KISS principle to remain simple and educational
// It supports both dry-run mode for testing and actual password updates
func main() {
	// Define command line flags for easy configuration
	// These flags allow non-programmers to use the tool without modifying code
	var (
		configFile    = flag.String("config", "config.yaml", "Path to configuration file")
		dryRun        = flag.Bool("dry-run", false, "Show what would be changed without making changes")
		verbose       = flag.Bool("verbose", false, "Enable verbose logging")
		enableMonitor = flag.Bool("monitor", false, "Start GRPC monitoring for error 49 detection")
		eduMode       = flag.Bool("edu", false, "Educational mode - uses simulated LDAP operations for learning")
		prodMode      = flag.Bool("prod", false, "Production mode - performs real LDAP operations (requires real LDAP server)")
	)
	flag.Parse()

	// Validate mode flags - only one mode can be active at a time
	// This ensures clear operation mode and prevents confusion
	modeCount := 0
	var operationMode string

	if *eduMode {
		modeCount++
		operationMode = "educational"
	}
	if *prodMode {
		modeCount++
		operationMode = "production"
	}
	if *dryRun {
		modeCount++
		operationMode = "dry-run"
	}

	// Default to educational mode if no mode is specified
	// This makes the application safe by default for learning
	if modeCount == 0 {
		*eduMode = true
		operationMode = "educational (default)"
	} else if modeCount > 1 {
		log.Fatalf("Error: Only one mode can be specified at a time (--edu, --prod, or --dry-run)")
	}

	// Load configuration from file
	// This separates configuration from code, making it easier for non-developers to modify
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up logging based on verbose flag
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("Verbose logging enabled")
	}

	fmt.Println("389DS LDAP Replication Password Manager")
	fmt.Println("======================================")
	fmt.Printf("Operation Mode: %s\n", operationMode)

	// Display mode-specific information
	if *eduMode {
		fmt.Println("📚 EDUCATIONAL MODE: Using simulated LDAP operations for learning")
		fmt.Println("   - No real LDAP connections will be made")
		fmt.Println("   - Safe for learning and testing concepts")
		fmt.Println("   - Use --prod flag for real operations")
	} else if *prodMode {
		fmt.Println("🔧 PRODUCTION MODE: Performing real LDAP operations")
		fmt.Println("   - Will connect to actual LDAP servers")
		fmt.Println("   - Will make real password changes")
		fmt.Println("   - Ensure your configuration is correct!")
	} else if *dryRun {
		fmt.Println("🔍 DRY-RUN MODE: Showing planned changes without execution")
		fmt.Println("   - Will connect to LDAP servers for discovery")
		fmt.Println("   - Will NOT make any password changes")
		fmt.Println("   - Safe for testing configuration")
	}

	// If monitor flag is set, start GRPC monitoring for real-time error detection
	if *enableMonitor {
		fmt.Println("Starting GRPC monitoring for error 49 detection...")
		go monitor.StartGRPCMonitor(cfg)
	}

	// Create LDAP manager to handle all LDAP operations
	// This encapsulates LDAP complexity and provides simple methods
	// Pass the operation mode to determine if we use real or simulated LDAP operations
	ldapManager, err := ldap.NewManager(cfg, *eduMode, *prodMode)
	if err != nil {
		log.Fatalf("Failed to create LDAP manager: %v", err)
	}
ldapManager.DryRun = *dryRun // Ensure dry-run mode is set
defer ldapManager.Close()

	// Create password manager to handle password generation and updates
	// This component ensures secure password generation and proper updates
	passwordManager := password.NewManager(cfg)

	// Main workflow: discover agreements, generate passwords, and update
	fmt.Println("\nStep 1: Discovering replication agreements...")
	agreements, err := ldapManager.DiscoverReplicationAgreements()
	if err != nil {
		log.Fatalf("Failed to discover replication agreements: %v", err)
	}

	if len(agreements) == 0 {
		fmt.Println("No replication agreements found.")
		os.Exit(0)
	}

	fmt.Printf("Found %d replication agreements\n", len(agreements))

	// Generate new passwords for all agreements
	fmt.Println("\nStep 2: Generating new passwords...")
	newPasswords := passwordManager.GeneratePasswords(agreements)

	// Display what will be changed (always show this for transparency)
	fmt.Println("\nStep 3: Planned changes:")
	fmt.Println("=======================")

	for _, agreement := range agreements {
		newPassword := newPasswords[agreement.Name]
		fmt.Printf("\nAgreement: %s\n", agreement.Name)
		fmt.Printf("  Supplier: %s\n", agreement.Supplier)
		fmt.Printf("  Consumer: %s\n", agreement.Consumer)
		fmt.Printf("  New Password: %s\n", newPassword)

		// Generate LDAP commands for manual execution
		supplierCmd := ldapManager.GeneratePasswordUpdateCommand(agreement.Supplier, agreement.Name, newPassword, "supplier")
		consumerCmd := ldapManager.GeneratePasswordUpdateCommand(agreement.Consumer, agreement.Name, newPassword, "consumer")

		fmt.Printf("  Manual LDAP Commands:\n")
		fmt.Printf("    Supplier: %s\n", supplierCmd)
		fmt.Printf("    Consumer: %s\n", consumerCmd)
	}

	// Handle different operation modes
	if *dryRun {
		// Dry-run mode: show what would be changed by calling the update methods
		// The LDAP manager will handle dry-run mode by showing changes without executing
		fmt.Println("\nStep 4: Dry-run simulation - showing what would be changed...")
		for _, agreement := range agreements {
			newPassword := newPasswords[agreement.Name]

			fmt.Printf("Processing agreement: %s\n", agreement.Name)

			// Call update methods - they will show what would be changed in dry-run mode
			if err := ldapManager.UpdateReplicationPassword(agreement.Supplier, agreement.Name, newPassword, "supplier"); err != nil {
				log.Printf("Error in dry-run simulation for supplier %s: %v", agreement.Name, err)
				continue
			}

			if err := ldapManager.UpdateReplicationPassword(agreement.Consumer, agreement.Name, newPassword, "consumer"); err != nil {
				log.Printf("Error in dry-run simulation for consumer %s: %v", agreement.Name, err)
				continue
			}

			fmt.Printf("  ✓ Dry-run simulation completed for %s\n", agreement.Name)
		}

		fmt.Println("\nDry-run completed: No actual changes were made.")
		fmt.Println("Use the LDAP commands shown above to make changes manually,")
		fmt.Println("or run with --prod flag to apply changes automatically.")
		return
	}

	// Production and educational modes: ask for confirmation before proceeding
	fmt.Print("\nDo you want to apply these changes? (y/N): ")
	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" {
		fmt.Println("Operation cancelled.")
		return
	}

	// Apply password changes (production mode will execute, educational mode will simulate)
	fmt.Println("\nStep 4: Applying password changes...")
	for _, agreement := range agreements {
		newPassword := newPasswords[agreement.Name]

		fmt.Printf("Updating agreement: %s\n", agreement.Name)

		// Update supplier password
		if err := ldapManager.UpdateReplicationPassword(agreement.Supplier, agreement.Name, newPassword, "supplier"); err != nil {
			log.Printf("Failed to update supplier password for %s: %v", agreement.Name, err)
			continue
		}

		// Update consumer password
		if err := ldapManager.UpdateReplicationPassword(agreement.Consumer, agreement.Name, newPassword, "consumer"); err != nil {
			log.Printf("Failed to update consumer password for %s: %v", agreement.Name, err)
			continue
		}

		fmt.Printf("  ✓ Successfully updated passwords for %s\n", agreement.Name)
	}

	fmt.Println("\nPassword update completed!")
	if *prodMode {
		fmt.Println("Monitor your /var/log/dirsrv/slapd-ldap/errors logs for any remaining error 49 messages.")
	} else {
		fmt.Println("Educational mode completed - no real changes were made.")
	}
}
