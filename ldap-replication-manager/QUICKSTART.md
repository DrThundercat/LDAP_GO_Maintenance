# Quick Start Guide

## 389DS LDAP Replication Password Manager

This guide gets you up and running quickly with the 389DS LDAP Replication Password Manager.

## Prerequisites Checklist

- [ ] Go 1.19+ installed
- [ ] Access to 389DS servers with replication configured
- [ ] Directory Manager or equivalent LDAP admin credentials
- [ ] Read access to `/var/log/dirsrv/slapd-ldap/errors` (for monitoring)

## 5-Minute Setup

### 1. Configure the Application

Edit `config.yaml` with your environment details:

```yaml
ldap:
  host: "your-ldap-server.example.com"    # Your supplier/hub server
  bind_dn: "cn=Directory Manager"         # Your admin DN
  password: "your-admin-password"         # Your admin password
```

### 2. Test Configuration (Dry Run)

```bash
# On Linux/Mac:
go run main.go --dry-run --verbose

# On Windows:
ldap-replication-manager.exe --dry-run --verbose
```

This shows what would be changed without making actual changes.

### 3. Apply Changes (When Ready)

```bash
# On Linux/Mac:
go run main.go --verbose

# On Windows:
ldap-replication-manager.exe --verbose
```

## Common Use Cases

### Scenario 1: Error 49 in Logs
You see this in `/var/log/dirsrv/slapd-ldap/errors`:
```
[01/Sep/2025:13:54:42 -0500] conn=123 op=456 RESULT err=49 tag=97 nentries=0 etime=0 - Invalid credentials for replication agreement: agreement-to-consumer1
```

**Solution**: Run the password manager to update all replication passwords:
```bash
./ldap-replication-manager.exe --verbose
```

### Scenario 2: Proactive Password Rotation
Schedule regular password updates (e.g., monthly):
```bash
# Add to cron (Linux) or Task Scheduler (Windows)
./ldap-replication-manager.exe --config /path/to/config.yaml
```

### Scenario 3: Real-time Monitoring
Monitor for error 49 events in real-time:
```bash
./ldap-replication-manager.exe --monitor --verbose
```

## Understanding the Output

### What You'll See (Dry Run)
```
389DS LDAP Replication Password Manager
======================================

Step 1: Discovering replication agreements...
Found 2 replication agreements
  - agreement-to-consumer1: ldap.example.com -> consumer1.example.com
  - agreement-to-consumer2: ldap.example.com -> consumer2.example.com

Step 2: Generating new passwords...

Step 3: Planned changes:
=======================

Agreement: agreement-to-consumer1
  Supplier: ldap.example.com
  Consumer: consumer1.example.com
  New Password: rX@q>+P9:U%+3?t,
  Manual LDAP Commands:
    Supplier: ldapmodify -x -D "cn=Directory Manager" -W -H ldap://ldap.example.com:389 << EOF
              dn: cn=agreement-to-consumer1,cn=replica,cn=dc=example,dc=com,cn=mapping tree,cn=config
              changetype: modify
              replace: nsds5replicacredentials
              nsds5replicacredentials: rX@q>+P9:U%+3?t,
              EOF
    Consumer: ldapmodify -x -D "cn=Directory Manager" -W -H ldap://consumer1.example.com:389 << EOF
              dn: cn=replication manager,cn=config
              changetype: modify
              replace: userPassword
              userPassword: rX@q>+P9:U%+3?t,
              EOF

Dry-run mode: No changes were made.
Use the LDAP commands above to make changes manually,
or run without --dry-run flag to apply changes automatically.
```

### What Each Step Does

1. **Discovery**: Finds all replication agreements on your LDAP server
2. **Password Generation**: Creates secure, random passwords for each agreement
3. **Planning**: Shows exactly what will be changed and provides manual LDAP commands
4. **Execution**: Updates both supplier and consumer passwords simultaneously

## Manual Execution (If Preferred)

If you prefer to run LDAP commands manually, copy the generated commands:

```bash
# Update supplier agreement password
ldapmodify -x -D "cn=Directory Manager" -W -H ldap://supplier.example.com:389 << EOF
dn: cn=agreement-name,cn=replica,cn=dc=example,dc=com,cn=mapping tree,cn=config
changetype: modify
replace: nsds5replicacredentials
nsds5replicacredentials: new-password-here
EOF

# Update consumer replication manager password
ldapmodify -x -D "cn=Directory Manager" -W -H ldap://consumer.example.com:389 << EOF
dn: cn=replication manager,cn=config
changetype: modify
replace: userPassword
userPassword: new-password-here
EOF
```

## Troubleshooting

### "Configuration file not found"
```bash
# Copy and edit the sample config
cp config.yaml my-config.yaml
# Edit my-config.yaml with your settings
./ldap-replication-manager.exe --config my-config.yaml --dry-run
```

### "LDAP connection failed"
- Verify host, port, and credentials in config.yaml
- Test LDAP connectivity: `ldapsearch -x -D "cn=Directory Manager" -W -H ldap://your-server:389 -b "cn=config"`

### "Permission denied"
- Ensure bind DN has replication management privileges
- For monitoring: ensure read access to log files

## Security Notes

- Set config file permissions: `chmod 600 config.yaml`
- Use dedicated service accounts instead of Directory Manager when possible
- Test in non-production environments first
- Monitor logs after password changes

## Next Steps

1. **Read the full README.md** for detailed documentation
2. **Customize password policies** in config.yaml
3. **Set up monitoring** with `--monitor` flag
4. **Schedule regular updates** via cron/Task Scheduler
5. **Integrate with your monitoring systems** using GRPC

## Support

- Check the troubleshooting section in README.md
- Use `--verbose` flag for detailed logging
- Test with `--dry-run` before making changes
- Review the extensive code comments for understanding

Remember: This tool modifies critical authentication infrastructure. Always test thoroughly!
