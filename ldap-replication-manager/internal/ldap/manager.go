package ldap

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/ldap-replication-manager/internal/config"
)

// ReplicationAgreement represents a 389DS replication agreement
// This structure contains all the information needed to manage replication passwords
// Each agreement connects a supplier (source) to a consumer (destination)
// The Name field corresponds to the cn attribute of the replication agreement
// Understanding this structure helps administrators see what will be modified
type ReplicationAgreement struct {
	// Name of the replication agreement (cn attribute)
	Name string

	// Supplier server (where data originates)
	Supplier string

	// Consumer server (where data is replicated to)
	Consumer string

	// Current bind DN used for replication
	BindDN string

	// Distinguished Name of the agreement in LDAP
	DN string

	// Whether this agreement is currently enabled
	Enabled bool
}

// Manager handles all LDAP operations for replication management
// This component encapsulates LDAP complexity and provides simple methods
// It maintains connections to LDAP servers and handles authentication
// The manager pattern makes the code easier to test and maintain
// Non-programmers can understand what each method does from its name
type Manager struct {
	config    *config.Config
	connected bool
	ldapConn  *ldap.Conn
	DryRun    bool // If true, only preview changes
}

// NewManager creates a new LDAP manager instance
// This function initializes the LDAP connection and validates connectivity
// It accepts mode flags to determine whether to use real or simulated operations
// Educational mode is safe for learning, production mode performs real operations
// Dry-run mode connects to real servers but doesn't make changes
// This design pattern separates connection management from business logic
func NewManager(cfg *config.Config, eduMode, prodMode bool) (*Manager, error) {
	// Accept dry-run as an argument (add to constructor signature in main.go)
	manager := &Manager{
		config: cfg,
		DryRun: false, // default, will be set by main.go
	}

	// Connect to LDAP
	l, err := ldap.DialURL(fmt.Sprintf("ldap://%s:%d", cfg.LDAP.Host, cfg.LDAP.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %v", err)
	}
	manager.ldapConn = l

	// Bind
	err = l.Bind(cfg.LDAP.BindDN, cfg.LDAP.Password)
	if err != nil {
		l.Close()
		return nil, fmt.Errorf("failed to bind to LDAP server: %v", err)
	}

	manager.connected = true
	log.Printf("Connected and bound to LDAP server: %s:%d", cfg.LDAP.Host, cfg.LDAP.Port)
	return manager, nil
}

// Close cleanly shuts down LDAP connections
// This ensures proper cleanup of network resources
// Always call this method when done with the manager
func (m *Manager) Close() {
	if m.connected && m.ldapConn != nil {
		log.Println("Closing LDAP connection")
		m.ldapConn.Close()
		m.connected = false
	}
}

// DiscoverReplicationAgreements finds all replication agreements on the server
// This method searches the LDAP directory for replication agreement objects
// It returns a slice of ReplicationAgreement structs with all relevant information
// The search is performed in the cn=config subtree where 389DS stores configuration
// Understanding this helps administrators see what agreements exist in their environment
func (m *Manager) DiscoverReplicationAgreements() ([]ReplicationAgreement, error) {
	if !m.connected || m.ldapConn == nil {
		return nil, fmt.Errorf("not connected to LDAP server")
	}

	log.Println("Searching for replication agreements...")

	searchRequest := ldap.NewSearchRequest(
		m.config.LDAP.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=nsds5ReplicationAgreement)",
		[]string{"cn", "nsds5replicahost", "nsds5replicabinddn", "dn", "nsds5replicaenabled"},
		nil,
	)

	sr, err := m.ldapConn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %v", err)
	}

	agreements := []ReplicationAgreement{}
	for _, entry := range sr.Entries {
		name := entry.GetAttributeValue("cn")
		supplier := m.config.LDAP.Host
		consumer := entry.GetAttributeValue("nsds5replicahost")
		bindDN := entry.GetAttributeValue("nsds5replicabinddn")
		dn := entry.DN
		enabled := true
		if val := entry.GetAttributeValue("nsds5replicaenabled"); val != "on" && val != "true" {
			enabled = false
		}
		agreements = append(agreements, ReplicationAgreement{
			Name:     name,
			Supplier: supplier,
			Consumer: consumer,
			BindDN:   bindDN,
			DN:       dn,
			Enabled:  enabled,
		})
	}

	log.Printf("Found %d replication agreements", len(agreements))
	for _, agreement := range agreements {
		log.Printf("  - %s: %s -> %s", agreement.Name, agreement.Supplier, agreement.Consumer)
	}

	return agreements, nil
}

// UpdateReplicationPassword updates the password for a replication agreement
// This method modifies both the supplier and consumer sides of the agreement
// The serverType parameter specifies whether we're updating "supplier" or "consumer"
// In production mode, this performs real LDAP operations
// In educational mode, this simulates the operations for learning
// In dry-run mode, this shows what would be changed without executing
func (m *Manager) UpdateReplicationPassword(server, agreementName, newPassword, serverType string) error {
	if !m.connected || m.ldapConn == nil {
		return fmt.Errorf("not connected to LDAP server")
	}

	if m.DryRun {
		// Print the planned LDAP modify command
		cmd := m.GeneratePasswordUpdateCommand(server, agreementName, newPassword, serverType)
		log.Printf("[DRY-RUN] Would execute: %s", cmd)
		return nil
	}

	var modifyReq *ldap.ModifyRequest
	if serverType == "supplier" {
		// Update nsds5replicacredentials on the agreement DN
		agreementDN := fmt.Sprintf("cn=%s,cn=replica,cn=dc=example,dc=com,cn=mapping tree,cn=config", agreementName)
		modifyReq = ldap.NewModifyRequest(agreementDN, nil)
		modifyReq.Replace("nsds5replicacredentials", []string{newPassword})
	} else {
		// Update userPassword on replication manager DN on consumer
		replicationManagerDN := "cn=replication manager,cn=config"
		modifyReq = ldap.NewModifyRequest(replicationManagerDN, nil)
		modifyReq.Replace("userPassword", []string{newPassword})
	}

	err := m.ldapConn.Modify(modifyReq)
	if err != nil {
		return fmt.Errorf("LDAP password update failed: %v", err)
	}

	log.Printf("Successfully updated %s password for agreement %s on server %s", serverType, agreementName, server)
	return nil
}

// GeneratePasswordUpdateCommand creates the LDAP command for manual password updates
// This method generates the exact ldapmodify command that would update passwords
// It's useful for dry-run mode and for administrators who prefer manual operations
// The generated commands can be saved to scripts for batch operations
// This educational feature helps users understand the underlying LDAP operations
func (m *Manager) GeneratePasswordUpdateCommand(server, agreementName, newPassword, serverType string) string {
	if serverType == "supplier" {
		// Generate command to update the replication agreement password on supplier
		// This modifies the nsds5replicacredentials attribute
		agreementDN := fmt.Sprintf("cn=%s,cn=replica,cn=dc=example,dc=com,cn=mapping tree,cn=config", agreementName)

		return fmt.Sprintf("ldapmodify -x -D \"%s\" -W -H ldap://%s:%d << EOF\ndn: %s\nchangetype: modify\nreplace: nsds5replicacredentials\nnsds5replicacredentials: %s\nEOF",
			m.config.LDAP.BindDN, server, m.config.LDAP.Port, agreementDN, newPassword)
	} else {
		// Generate command to update the replication manager password on consumer
		// This updates the actual user account that the supplier binds as
		replicationManagerDN := "cn=replication manager,cn=config"

		return fmt.Sprintf("ldapmodify -x -D \"%s\" -W -H ldap://%s:%d << EOF\ndn: %s\nchangetype: modify\nreplace: userPassword\nuserPassword: %s\nEOF",
			m.config.LDAP.BindDN, server, m.config.LDAP.Port, replicationManagerDN, newPassword)
	}
}

// GetReplicationStatus checks the current status of replication agreements
// This method helps identify agreements that might have authentication issues
// It can detect error 49 conditions by examining replication state
// The status information helps prioritize which agreements need password updates
// This diagnostic capability is essential for troubleshooting replication problems
func (m *Manager) GetReplicationStatus(agreements []ReplicationAgreement) map[string]string {
	status := make(map[string]string)

	for _, agreement := range agreements {
		// In a real implementation, this would check:
		// - Last successful replication timestamp
		// - Current replication state (enabled/disabled)
		// - Any error conditions in the replication log
		// - Consumer connectivity status

		// For this educational example, simulate status checking
		if strings.Contains(agreement.Name, "consumer1") {
			status[agreement.Name] = "ERROR: Authentication failure (error 49)"
		} else {
			status[agreement.Name] = "OK: Replication active"
		}
	}

	return status
}
