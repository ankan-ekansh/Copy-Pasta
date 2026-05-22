# 🚀 Copy-Pasta — Deployment Guide

## Architecture

Single container running on **Azure Container Apps**:
- Go binary serves both API (`/api/*`) and static React files (everything else)
- Scales 0–3 replicas based on traffic (0 = no cost when idle)
- Auto-TLS via Azure-managed certificates

## First-Time Setup

### Prerequisites
- [Azure CLI](https://docs.microsoft.com/cli/azure/install-azure-cli) installed
- Logged in: `az login`
- Docker installed and running
- OpenSSL (for password generation)

### 1. Create Azure Resources

```bash
# Generate a strong password for PostgreSQL (required)
export PG_ADMIN_PASSWORD=$(openssl rand -hex 20)

# Provision all resources (ACR, Container App, PostgreSQL)
./infra/setup-azure.sh
```

> ⚠️ `PG_ADMIN_PASSWORD` must be set before running the script. The password is stored
> as an encrypted Container App secret in Azure — you don't need to save it locally.
> To retrieve it later: `az containerapp secret show --name copy-pasta --resource-group rg-copy-pasta --secret-name database-url`

This creates:
- Resource group: `rg-copy-pasta` (centralindia)
- Container Registry: `copypastacr.azurecr.io`
- Container Apps Environment + App
- PostgreSQL Flexible Server (B1ms) + database + firewall rule
- `DATABASE_URL` wired as Container App secret

#### Security note

The setup script contains PostgreSQL server name, admin username, and database name in plain text. **This is intentional and safe:**

| Value | Why it's OK to commit |
|-------|----------------------|
| Server name (`pg-sv-copy-pasta`) | Publicly visible in the FQDN regardless |
| Admin user (`copypasta`) | Useless without password + network access |
| Database name (`copypasta`) | Just an identifier, no security value |

**What protects the database:**
1. **Password** — never in source code; passed via `PG_ADMIN_PASSWORD` env var at runtime
2. **Firewall** — only Azure services (Container App) can connect; public internet blocked
3. **SSL** — `sslmode=require` enforced in the connection string

#### Network security approach

The PostgreSQL server is created with `--public-access 0.0.0.0` which enables public networking but restricts connections to Azure-internal services only. The Azure CLI auto-adds a firewall rule for the provisioner's client IP during creation — the setup script **automatically removes this** after provisioning, leaving only the `AllowAzureServices` rule (0.0.0.0–0.0.0.0).

**Result:** Any Azure service (across all subscriptions) with the `AllowAzureServices` rule can reach the database. In practice, only our Container App connects because it has the credentials. No external (non-Azure) IP can connect.

If you need temporary local access for debugging:
```bash
# Add your IP temporarily
az postgres flexible-server firewall-rule create \
  --resource-group rg-copy-pasta --name pg-sv-copy-pasta \
  --rule-name TempLocalAccess --start-ip-address <your-ip> --end-ip-address <your-ip>

# Remove when done
az postgres flexible-server firewall-rule delete \
  --resource-group rg-copy-pasta --name pg-sv-copy-pasta \
  --rule-name TempLocalAccess --yes
```

### 2. Set Up GitHub Actions Secret

In your GitHub repo → **Settings → Secrets and variables → Actions → New repository secret**:

| Secret | How to get it |
|--------|---------------|
| `AZURE_CREDENTIALS` | Entire JSON output from: `az ad sp create-for-rbac --name "copy-pasta-github" --role contributor --scopes /subscriptions/{sub-id}/resourceGroups/rg-copy-pasta --sdk-auth` |

> ⚠️ Use a **repository secret** (not environment secret). Paste the full JSON blob as the value.

### 3. Push to Main

```bash
git push origin main
```

GitHub Actions will automatically build, push, and deploy.

## Manual Deployment

If you need to deploy manually:

```bash
# Build
docker build -t copypastacr.azurecr.io/copy-pasta:latest .

# Push
az acr login --name copypastacr
docker push copypastacr.azurecr.io/copy-pasta:latest

# Deploy
az containerapp update \
  --name copy-pasta \
  --resource-group rg-copy-pasta \
  --image copypastacr.azurecr.io/copy-pasta:latest
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server listen port |
| `STATIC_DIR` | `static` | Path to frontend build files |
| `CORS_ORIGINS` | `*` | Comma-separated allowed origins |

## Monitoring

```bash
# View logs
az containerapp logs show \
  --name copy-pasta \
  --resource-group rg-copy-pasta \
  --follow

# Check status
az containerapp show \
  --name copy-pasta \
  --resource-group rg-copy-pasta \
  --query "properties.runningStatus"
```

## Costs

With Azure Container Apps Consumption tier:
- **Idle**: $0 (scales to zero)
- **Light traffic**: ~$1-2/mo
- **Container Registry (Basic)**: ~$5/mo
- **Total estimate**: ~$5-7/mo for a side project
