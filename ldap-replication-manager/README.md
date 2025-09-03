# 389DS LDAP Replication Password Manager

A Go application designed to help RHEL system administrators manage 389DS LDAP replication agreement passwords. This tool can detect error 49 (authentication failure) events and automatically update passwords across suppliers, hubs, and consumers.

## Overview

This application was built by a RHEL7/RHEL8 389DS server administrator learning Go. It demonstrates how to:

- Manage LDAP replication agreement passwords programmatically
- Detect error 49 authentication failures in real-time using GRPC
- Generate secure passwords following organizational policies
- Provide both automated and manual password update workflows
- Follow the KISS (Keep It Simple, Stupid) principle for maintainability

## Features

### Core Functionality
- **Automatic Discovery**: Finds all replication agreements on your 389DS servers
- **Predefined Passwords**: Uses passwords specified in your configuration file for each agreement
- **Dual Update Mode**: Updates both supplier/hub and all consumer passwords simultaneously
- **Dry-Run Mode**: Prints a detailed plan of what would be changed, including LDAP commands, without making any changes
- **Manual Commands**: Generates LDAP commands for manual execution

### Error 49 Detection
- **Real-time Monitoring**: Uses GRPC to monitor log files for authentication failures
- **Pattern Recognition**: Identifies error 49 events and associated replication agreements
- **Automated Response**: Can trigger password updates when errors are detected

### Educational Design
- **Extensive Comments**: Every function has detailed explanations
- **Clear Structure**: Modular design makes it easy to understand and modify
- **Configuration-Driven**: Non-programmers can customize behavior via YAML files
- **Transparent Operations**: Shows exactly what commands would be executed

## Understanding Error 49

Error 49 in 389DS logs indicates an authentication failure. In replication contexts, this typically means:

```
[01/Sep/2025:13:54:42 -0500] conn=123 op=456 RESULT err=49 tag=97 nentries=0 etime=0 - Invalid credentials for replication agreement: agreement-to-consumer1
```

This error occurs when:
- Replication agreement passwords have expired or been changed
- Consumer server passwords don't match supplier expectations
- Network issues cause authentication timeouts
- Certificate problems in TLS-enabled environments

## Installation

### Prerequisites
- Go 1.19 or later
- Access to 389DS servers with replication configured
- LDAP administrative credentials (typically Directory Manager)
- Read access to 389DS log files (for monitoring)

### Building the Application

1. Clone or download the source code:
```bash
cd /opt
git clone <repository-url> ldap-replication-manager
cd ldap-replication-manager
```

2. Initialize Go modules and download dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o ldap-replication-manager main.go
```

4. Make it executable:
```bash
chmod +x ldap-replication-manager
```

## Configuration

### Basic Setup

1. Copy the sample configuration:
```bash
cp config.yaml config-production.yaml
```

2. Edit the configuration file:
```bash
vi config-production.yaml
```

3. Set secure file permissions:
```bash
chmod 600 config-production.yaml
```

### Key Configuration Settings

#### LDAP Connection
```yaml
ldap:
  host: "your-ldap-server.example.com"
  port: 389
  bind_dn: "cn=Directory Manager"
  password: "your-secure-password"
```

### Password Management (Predefined Only)
```yaml
password:
  predefined_passwords:
    agreement-to-consumer1: "Consumer1SecurePass123!"
    agreement-to-consumer2: "Consumer2SecurePass456@"
  # Optionally, set a default_password for agreements not listed above
  # default_password: "DefaultReplicationPassword789#"
```

#### Monitoring Settings
```yaml
grpc:
  enabled: true
  port: 50051
  log_paths:
    - "/var/log/dirsrv/slapd-ldap/errors"
  check_interval: 5
```

## Usage

### Basic Password Update

1. **Dry-run mode** (recommended first step):
```bash
./ldap-replication-manager --config config-production.yaml --dry-run
```

This connects to your real LDAP servers, discovers actual replication agreements, and shows exactly what would be changed without making any modifications. This is the safest way to test your configuration.

2. **Apply changes**:
```bash
./ldap-replication-manager --config config-production.yaml
```

3. **With verbose logging**:
```bash
./ldap-replication-manager --config config-production.yaml --verbose
```

### Real-time Monitoring

Start the application with GRPC monitoring enabled:
```bash
./ldap-replication-manager --config config-production.yaml --monitor
```

This will:
- Monitor log files for error 49 events
- Provide real-time notifications of authentication failures
- Enable integration with monitoring systems

### Command Line Options

| Option | Description | Default |
|--------|-------------|---------|
| `--config` | Path to configuration file | `config.yaml` |
| `--edu` | Educational mode - simulated LDAP operations for learning | `false` |
| `--prod` | Production mode - real LDAP operations (requires real server) | `false` |
| `--dry-run` | Show changes without applying them | `false` |
| `--verbose` | Enable detailed logging | `false` |
| `--monitor` | Start GRPC monitoring | `false` |

**Note**: Only one mode can be active at a time (`--edu`, `--prod`, or `--dry-run`). If no mode is specified, educational mode is used by default for safety.

## Understanding the Output

### Discovery Phase
```
Step 1: Discovering replication agreements...
Found 2 replication agreements
  - agreement-to-consumer1: ldap.example.com -> consumer1.example.com
  - agreement-to-consumer2: ldap.example.com -> consumer2.example.com
```

### Password Assignment
```
Step 2: Assigning predefined passwords...
```

### Planned Changes
```
Step 3: Planned changes:
=======================

Agreement: agreement-to-consumer1
  Supplier: ldap.example.com
  Consumer: consumer1.example.com
  New Password: Consumer1SecurePass123!
  Manual LDAP Commands:
    Supplier: ldapmodify -x -D "cn=Directory Manager" -W -H ldap://ldap.example.com:389 << EOF
              dn: cn=agreement-to-consumer1,cn=replica,cn=dc=example,dc=com,cn=mapping tree,cn=config
              changetype: modify
              replace: nsds5replicacredentials
              nsds5replicacredentials: Kx7#mP9$qR2@nL5!
              EOF
    Consumer: ldapmodify -x -D "cn=Directory Manager" -W -H ldap://consumer1.example.com:389 << EOF
              dn: cn=replication manager,cn=config
              changetype: modify
              replace: userPassword
              userPassword: Kx7#mP9$qR2@nL5!
              EOF
```

### Execution Phase
```
Step 4: Applying password changes...
Updating agreement: agreement-to-consumer1
  ✓ Successfully updated passwords for agreement-to-consumer1
```

## Manual LDAP Commands

The application generates standard LDAP commands that can be executed manually:

### Update Supplier Agreement Password
```bash
ldapmodify -x -D "cn=Directory Manager" -W -H ldap://supplier.example.com:389 << EOF
dn: cn=agreement-name,cn=replica,cn=dc=example,dc=com,cn=mapping tree,cn=config
changetype: modify
replace: nsds5replicacredentials
nsds5replicacredentials: new-password-here
EOF
```

### Update Consumer Replication Manager Password
```bash
ldapmodify -x -D "cn=Directory Manager" -W -H ldap://consumer.example.com:389 << EOF
dn: cn=replication manager,cn=config
changetype: modify
replace: userPassword
userPassword: new-password-here
EOF
```

## Troubleshooting

### Common Issues

#### Configuration File Not Found
```
Failed to load configuration: failed to read config file config.yaml: no such file or directory
```
**Solution**: Ensure the configuration file exists and the path is correct.

#### LDAP Connection Failed
```
Failed to create LDAP manager: failed to connect to LDAP server: LDAP host not configured
```
**Solution**: Check LDAP host, port, and credentials in the configuration file.

#### Permission Denied
```
Failed to update supplier password: insufficient access rights
```
**Solution**: Ensure the bind DN has permission to modify replication configuration.

### Log File Monitoring Issues

#### Log Files Not Accessible
```
Error checking log file /var/log/dirsrv/slapd-ldap/errors: permission denied
```
**Solution**: Ensure the application has read access to log files:
```bash
# Add user to dirsrv group
usermod -a -G dirsrv your-username

# Or adjust log file permissions
chmod 644 /var/log/dirsrv/slapd-ldap/errors
```

### Debugging Steps

1. **Test with dry-run mode**:
```bash
./ldap-replication-manager --dry-run --verbose
```

2. **Check LDAP connectivity**:
```bash
ldapsearch -x -D "cn=Directory Manager" -W -H ldap://your-server:389 -b "cn=config" "(objectclass=nsds5replicationagreement)"
```

3. **Verify log file access**:
```bash
tail -f /var/log/dirsrv/slapd-ldap/errors
```

## Security Considerations

### Configuration File Security
- Store configuration files with restricted permissions (600)
- Consider using environment variables for sensitive data
- Rotate LDAP service account passwords regularly

### Network Security
- Use TLS/LDAPS for production environments
- Implement firewall rules for GRPC monitoring ports
- Monitor application logs for security events

### Password Security
- Passwords are only set from your configuration file (predefined)
- No passwords are generated by the application
- Old passwords are immediately replaced, not stored

## Architecture

### Component Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Config        │    │   LDAP          │    │   Password      │
│   Management    │────│   Manager       │────│   Generator     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │   GRPC          │
                    │   Monitor       │
                    └─────────────────┘
```

### Key Design Principles

1. **Separation of Concerns**: Each component has a single responsibility
2. **Configuration-Driven**: Behavior controlled through YAML files
3. **Error Handling**: Comprehensive error checking and reporting
4. **Educational**: Extensive comments and clear structure
5. **Testable**: Modular design enables easy testing

## Development

### Project Structure
```
ldap-replication-manager/
├── main.go                          # Application entry point
├── go.mod                           # Go module definition
├── config.yaml                      # Sample configuration
├── README.md                        # This documentation
├── internal/
│   ├── config/
│   │   └── config.go               # Configuration management
│   ├── ldap/
│   │   └── manager.go              # LDAP operations
│   ├── password/
│   │   └── generator.go            # Password generation
│   └── monitor/
│       └── grpc.go                 # GRPC monitoring
```

### Adding New Features

1. **New Configuration Options**: Add to `config.go` structures
2. **LDAP Operations**: Extend the `ldap.Manager` type
3. **Password Policies**: Modify `password.Manager` methods
4. **Monitoring**: Enhance `monitor.GRPCMonitor` capabilities

### Testing

Run the application in dry-run mode to test without making changes:
```bash
go run main.go --dry-run --verbose
```

## Contributing

This project is designed to be educational and easily modifiable. When making changes:

1. Maintain extensive comments explaining the "why" not just the "what"
2. Follow the KISS principle - keep solutions simple
3. Ensure non-programmers can understand the logic
4. Test thoroughly with dry-run mode
5. Update documentation for any new features

## License

This educational project is provided as-is for learning purposes. Use at your own risk in production environments.

## Support

For questions or issues:
1. Check the troubleshooting section above
2. Review the extensive code comments
3. Test with dry-run and verbose modes
4. Consult 389DS documentation for LDAP-specific issues

Remember: This tool modifies critical authentication infrastructure. Always test thoroughly in non-production environments first!
