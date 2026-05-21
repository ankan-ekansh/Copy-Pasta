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

### 1. Create Azure Resources

```bash
./infra/setup-azure.sh
```

This creates:
- Resource group: `rg-copy-pasta`
- Container Registry: `copypastacr.azurecr.io`
- Container Apps Environment + App

### 2. Set Up GitHub Actions Secrets

In your GitHub repo → Settings → Secrets → Actions, add:

| Secret | How to get it |
|--------|---------------|
| `AZURE_CREDENTIALS` | `az ad sp create-for-rbac --name "copy-pasta-cicd" --role contributor --scopes /subscriptions/{sub-id}/resourceGroups/rg-copy-pasta --json-auth` |

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
