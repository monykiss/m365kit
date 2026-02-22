# Enterprise Deployment Guide

## Quick Start (Single Machine)

```bash
# macOS / Linux
brew install monykiss/tap/m365kit
kit config init

# Windows (PowerShell, no admin)
iwr https://raw.githubusercontent.com/monykiss/m365kit/main/packaging/windows/install.ps1 | iex
kit config init
```

## Mass Deployment Options

### Windows (GPO / SCCM / Intune)

**Option A: MSI / NSIS Installer (admin, silent)**
1. Download `M365Kit-{version}-Setup.exe` from [GitHub Releases](https://github.com/monykiss/m365kit/releases)
2. Silent install: `M365Kit-Setup.exe /S /D="C:\Program Files\M365Kit"`
3. Deploy via SCCM as an application or via Intune as a Win32 app
4. The installer adds `kit.exe` to the system PATH automatically

**Option B: PowerShell (no admin, user-scoped)**
```powershell
iwr https://raw.githubusercontent.com/monykiss/m365kit/main/packaging/windows/install.ps1 | iex
```
Installs to `%LOCALAPPDATA%\M365Kit` and adds to user PATH.

**Pre-configuration via Group Policy:**
Push `%USERPROFILE%\.kit\config.yaml` via Group Policy Preferences (File).

### macOS (MDM / Jamf)

**Option A: Homebrew (managed Brewfile)**
```ruby
# In your managed Brewfile
tap "monykiss/tap"
brew "monykiss/tap/m365kit"
```

**Option B: Direct binary**
```bash
curl -L https://github.com/monykiss/m365kit/releases/latest/download/m365kit_Darwin_arm64.tar.gz | tar xz -C /usr/local/bin/
```

**Pre-configuration via script:**
```bash
#!/bin/bash
mkdir -p ~/.kit
cat > ~/.kit/config.yaml << 'EOF'
provider: anthropic
model: claude-sonnet-4-20250514
azure:
  client_id: "YOUR-ORG-CLIENT-ID"
EOF
chmod 600 ~/.kit/config.yaml
```

### Linux (apt/yum/systemd)

```bash
# Download binary
curl -L https://github.com/monykiss/m365kit/releases/latest/download/m365kit_Linux_amd64.tar.gz | tar xz -C /usr/local/bin/

# System-wide config (optional)
mkdir -p /etc/kit
cat > /etc/kit/config.yaml << 'EOF'
provider: anthropic
azure:
  client_id: "YOUR-ORG-CLIENT-ID"
EOF
```

## Organization-Wide Configuration (v1.1+)

Deploy an org config to control settings, lock values, and enable audit logging for all users.

**Path:** `/etc/kit/org.yaml` (macOS/Linux) or `C:\ProgramData\M365Kit\org.yaml` (Windows)

```bash
# Generate a template
kit org init --org-name "Acme Corp" --domain acme.com > /etc/kit/org.yaml

# Validate before deploying
kit org validate /etc/kit/org.yaml

# Users can check their policy status
kit org status
```

**Example org config:**

```yaml
org_name: "Acme Corp"
org_domain: "acme.com"

azure:
  client_id: "your-azure-app-client-id"
  tenant_id: "your-tenant-id"

ai:
  provider: anthropic
  model: claude-sonnet-4-20250514

locked:
  azure_client_id: true   # Users cannot override
  ai_provider: true       # Users cannot switch providers

audit:
  enabled: true
  file_path: "~/.kit/audit.log"
  level: command

telemetry:
  enabled: false
```

**Deployment methods:**
- **Windows GPO:** Push to `C:\ProgramData\M365Kit\org.yaml` via Group Policy Preferences (File)
- **macOS MDM / Jamf:** Deploy to `/etc/kit/org.yaml` via configuration profile or script
- **Linux:** Deploy to `/etc/kit/org.yaml` via Ansible, Chef, or Puppet

### Audit Logging

When `audit.enabled: true` in org config, every command is logged to a JSONL file:

```bash
# View recent audit entries
kit audit log --last 20

# Filter by user or command
kit audit log --user alice@acme.com --since 2026-01-01

# Check log size
kit audit status

# IT admin: aggregated usage statistics
kit admin stats --since 2026-01-01
kit admin users
```

Sensitive arguments (API keys, tokens, passwords) are automatically redacted in audit logs.

### Telemetry

Local telemetry stores anonymous, aggregate command usage data (no user IDs, no file paths).

```bash
kit admin telemetry status   # Check store size
kit admin telemetry clear    # Delete local telemetry
```

## Pre-Configuration Template

Create this file at `~/.kit/config.yaml` before first use:

```yaml
# AI Provider
provider: anthropic
model: claude-sonnet-4-20250514
api_keys:
  anthropic: ""  # Set via ANTHROPIC_API_KEY env var (recommended)

# Microsoft 365
azure:
  client_id: "your-org-azure-app-client-id"

# Email (optional)
smtp:
  host: "smtp.office365.com"
  port: "587"
  username: ""  # Set via KIT_SMTP_USERNAME env var
```

**Best practice:** Use environment variables for secrets, config file for non-sensitive settings.

## Azure AD App Registration (IT Admin)

Register once for your entire organization:

1. Go to [Azure Portal](https://portal.azure.com) > Azure Active Directory > App Registrations > New Registration
2. Name: "M365Kit CLI"
3. Supported account types: "Accounts in this organizational directory only"
4. Redirect URI: Public client/native > `http://localhost`
5. API Permissions (Delegated):
   - `Files.ReadWrite` — OneDrive file access
   - `Sites.ReadWrite.All` — SharePoint access
   - `User.Read` — User profile
   - `Chat.ReadWrite` — Teams DMs
   - `ChannelMessage.Send` — Teams channel posts
   - `Team.ReadBasic.All` — List joined teams
   - `offline_access` — Refresh tokens
6. Click "Grant admin consent" for your organization
7. Go to Authentication > Advanced > Enable "Allow public client flows"
8. Copy the Application (client) ID
9. Distribute `KIT_AZURE_CLIENT_ID` via environment variable or config file

## Security Notes

| Item | Location | Permissions | Notes |
|------|----------|-------------|-------|
| Config file | `~/.kit/config.yaml` | 0600 | May contain API keys |
| Auth token | `~/.kit/token.json` | 0600 | OAuth access + refresh tokens |
| API keys | Environment or config | — | Prefer env vars in CI/CD |

**Important:**
- Auth tokens are **never** stored in `config.yaml` (separate file)
- API keys in config.yaml are stored with 0600 permissions
- Use environment variables for secrets in CI/CD pipelines
- Token auto-refreshes within 5 minutes of expiry

## CI/CD Integration

```yaml
# GitHub Actions example
- name: Install M365Kit
  run: go install github.com/monykiss/m365kit@latest

- name: Process documents
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
    KIT_AZURE_CLIENT_ID: ${{ secrets.KIT_AZURE_CLIENT_ID }}
  run: |
    kit word read contract.docx | kit ai summarize > summary.txt
    kit fs scan ./documents -r --json > manifest.json
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `kit auth login` hangs | Check firewall allows `login.microsoftonline.com` |
| `kit onedrive ls` returns 403 | Ensure admin consent is granted for your Azure app |
| `kit teams post` returns 403 | Add `ChannelMessage.Send` permission and re-consent |
| Config not loading | Run `kit config path` to verify location |
| Update check slow | Set `KIT_NO_UPDATE_CHECK=1` to disable |
