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
	)
	flag.Parse()

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

	// If monitor flag is set, start GRPC monitoring for real-time error detection
	if *enableMonitor {
		fmt.Println("Starting GRPC monitoring for error 49 detection...")
		go monitor.StartGRPCMonitor(cfg)
	}

	// Create LDAP manager to handle all LDAP operations
	// This encapsulates LDAP complexity and provides simple methods
	ldapManager, err := ldap.NewManager(cfg)
	if err != nil {
		log.Fatalf("Failed to create LDAP manager: %v", err)
	}
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

	// If dry-run mode, exit without making changes
	if *dryRun {
		fmt.Println("\nDry-run mode: No changes were made.")
		fmt.Println("Use the LDAP commands above to make changes manually,")
		fmt.Println("or run without --dry-run flag to apply changes automatically.")
		return
	}

	// Ask for confirmation before making changes
	fmt.Print("\nDo you want to apply these changes? (y/N): ")
	var response string
	fmt.Scanln(&response)

	if response != "y" && response != "Y" {
		fmt.Println("Operation cancelled.")
		return
	}

	// Apply password changes
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
	fmt.Println("Monitor your /var/log/dirsrv/slapd-ldap/errors logs for any remaining error 49 messages.")
}
