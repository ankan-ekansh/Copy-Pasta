#!/usr/bin/env bash
# =============================================================================
# Copy-Pasta — Azure Infrastructure Setup
# Creates all Azure resources needed for deployment.
#
# Prerequisites:
#   - Azure CLI installed and logged in (az login)
#   - OpenSSL (for password generation: openssl rand -hex 20)
#   - A subscription with credits available
#
# Note: This script is idempotent but does NOT migrate existing resources to a
# new region. To change regions, delete the resource group and re-run.
#
# Usage:
#   chmod +x infra/setup-azure.sh
#   ./infra/setup-azure.sh
# =============================================================================

set -euo pipefail

# --- Configuration (customize these) ---
RESOURCE_GROUP="rg-copy-pasta"
LOCATION="centralindia"
ACR_NAME="copypastacr"  # must be globally unique, lowercase, no hyphens
CONTAINER_APP_ENV="cae-copy-pasta"
CONTAINER_APP_NAME="copy-pasta"
IMAGE_NAME="copy-pasta"
IMAGE_TAG="latest"
PG_SERVER_NAME="pg-sv-copy-pasta"
PG_ADMIN_USER="copypasta"
PG_DB_NAME="copypasta"

echo "🍝 Copy-Pasta Azure Infrastructure Setup"
echo "========================================="
echo "Resource Group: $RESOURCE_GROUP"
echo "Location:       $LOCATION"
echo "ACR:            $ACR_NAME.azurecr.io"
echo ""

# --- Step 0: Register required resource providers ---
PROVIDERS=("Microsoft.App" "Microsoft.OperationalInsights" "Microsoft.ContainerRegistry" "Microsoft.DBforPostgreSQL")
for provider in "${PROVIDERS[@]}"; do
  state=$(az provider show --namespace "$provider" --query "registrationState" -o tsv 2>/dev/null || echo "NotRegistered")
  if [ "$state" != "Registered" ]; then
    echo "🔧 Registering resource provider $provider..."
    az provider register -n "$provider" --wait
  else
    echo "✓ Provider $provider already registered."
  fi
done
echo ""

# --- Step 1: Resource Group ---
if az group show --name "$RESOURCE_GROUP" &>/dev/null; then
  echo "📦 Resource group '$RESOURCE_GROUP' already exists — skipping."
else
  echo "📦 Creating resource group..."
  az group create \
    --name "$RESOURCE_GROUP" \
    --location "$LOCATION" \
    --output none
fi

# --- Step 2: Azure Container Registry ---
if az acr show --name "$ACR_NAME" &>/dev/null; then
  echo "🐳 Container registry '$ACR_NAME' already exists — skipping."
else
  echo "🐳 Creating container registry..."
  az acr create \
    --resource-group "$RESOURCE_GROUP" \
    --name "$ACR_NAME" \
    --sku Basic \
    --admin-enabled true \
    --output none
fi

# Get ACR credentials for later use
ACR_LOGIN_SERVER=$(az acr show --name "$ACR_NAME" --query loginServer -o tsv)
ACR_USERNAME=$(az acr credential show --name "$ACR_NAME" --query username -o tsv)
ACR_PASSWORD=$(az acr credential show --name "$ACR_NAME" --query "passwords[0].value" -o tsv)

echo "   Registry: $ACR_LOGIN_SERVER"

# --- Step 3: Container Apps Environment ---
if az containerapp env show --name "$CONTAINER_APP_ENV" --resource-group "$RESOURCE_GROUP" &>/dev/null; then
  echo "🌐 Container Apps environment '$CONTAINER_APP_ENV' already exists — skipping."
else
  echo "🌐 Creating Container Apps environment..."
  az containerapp env create \
    --name "$CONTAINER_APP_ENV" \
    --resource-group "$RESOURCE_GROUP" \
    --location "$LOCATION" \
    --output none
fi

# --- Step 4: Container App (initial deployment with placeholder) ---
if az containerapp show --name "$CONTAINER_APP_NAME" --resource-group "$RESOURCE_GROUP" &>/dev/null; then
  echo "🚀 Container App '$CONTAINER_APP_NAME' already exists — skipping."
else
  echo "🚀 Creating Container App..."
  az containerapp create \
    --name "$CONTAINER_APP_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --environment "$CONTAINER_APP_ENV" \
    --image "mcr.microsoft.com/azuredocs/containerapps-helloworld:latest" \
    --target-port 8080 \
    --ingress external \
    --min-replicas 0 \
    --max-replicas 3 \
    --cpu 0.25 \
    --memory 0.5Gi \
    --registry-server "$ACR_LOGIN_SERVER" \
    --registry-username "$ACR_USERNAME" \
    --registry-password "$ACR_PASSWORD" \
    --output none
fi

# Get the app URL
APP_URL=$(az containerapp show \
  --name "$CONTAINER_APP_NAME" \
  --resource-group "$RESOURCE_GROUP" \
  --query "properties.configuration.ingress.fqdn" -o tsv)

# --- Step 5: PostgreSQL Flexible Server ---
if az postgres flexible-server show --name "$PG_SERVER_NAME" --resource-group "$RESOURCE_GROUP" &>/dev/null; then
  echo "🐘 PostgreSQL server '$PG_SERVER_NAME' already exists — skipping."
else
  # Require password from environment
  if [ -z "${PG_ADMIN_PASSWORD:-}" ]; then
    echo "❌ Error: PG_ADMIN_PASSWORD environment variable is required."
    echo "   Set it before running: export PG_ADMIN_PASSWORD=\$(openssl rand -hex 20)"
    exit 1
  fi

  # Validate password is hex-only (URL-safe for postgres:// URI)
  if ! printf '%s' "$PG_ADMIN_PASSWORD" | grep -qE '^[0-9a-fA-F]+$'; then
    echo "❌ Error: PG_ADMIN_PASSWORD must be hex-only (0-9, a-f)."
    echo "   Generate with: export PG_ADMIN_PASSWORD=\$(openssl rand -hex 20)"
    exit 1
  fi

  echo "🐘 Creating PostgreSQL Flexible Server (B1ms — free tier eligible)..."
  az postgres flexible-server create \
    --resource-group "$RESOURCE_GROUP" \
    --name "$PG_SERVER_NAME" \
    --location "$LOCATION" \
    --admin-user "$PG_ADMIN_USER" \
    --admin-password "$PG_ADMIN_PASSWORD" \
    --sku-name Standard_B1ms \
    --tier Burstable \
    --storage-size 32 \
    --version 16 \
    --public-access 0.0.0.0 \
    --yes \
    --output none
fi

# Ensure database exists (idempotent)
DB_EXISTS=$(az postgres flexible-server db list \
  --resource-group "$RESOURCE_GROUP" \
  --server-name "$PG_SERVER_NAME" \
  --query "[?name=='$PG_DB_NAME'] | length(@)" -o tsv)
if [ "$DB_EXISTS" = "0" ]; then
  echo "   Creating database '$PG_DB_NAME'..."
  az postgres flexible-server db create \
    --resource-group "$RESOURCE_GROUP" \
    --server-name "$PG_SERVER_NAME" \
    --database-name "$PG_DB_NAME" \
    --output none
else
  echo "   Database '$PG_DB_NAME' already exists."
fi

# Ensure firewall is hardened (idempotent)
echo "   Ensuring firewall is hardened (Azure-only access)..."
# Remove any rule whose IP range isn't exactly 0.0.0.0–0.0.0.0
# (This keeps any rule with that exact range, regardless of name)
UNWANTED_RULES=$(az postgres flexible-server firewall-rule list \
  --resource-group "$RESOURCE_GROUP" \
  --name "$PG_SERVER_NAME" \
  --query "[?(startIpAddress!='0.0.0.0' || endIpAddress!='0.0.0.0')].name" -o tsv)
if [ -n "$UNWANTED_RULES" ]; then
  while IFS= read -r rule; do
    echo "   Removing firewall rule: $rule"
    az postgres flexible-server firewall-rule delete \
      --resource-group "$RESOURCE_GROUP" \
      --name "$PG_SERVER_NAME" \
      --rule-name "$rule" \
      --yes \
      --output none
  done <<< "$UNWANTED_RULES"
fi

# Ensure AllowAzureServices rule exists
AZURE_RULE_EXISTS=$(az postgres flexible-server firewall-rule list \
  --resource-group "$RESOURCE_GROUP" \
  --name "$PG_SERVER_NAME" \
  --query "[?name=='AllowAzureServices'] | length(@)" -o tsv)
if [ "$AZURE_RULE_EXISTS" = "0" ]; then
  az postgres flexible-server firewall-rule create \
    --resource-group "$RESOURCE_GROUP" \
    --name "$PG_SERVER_NAME" \
    --rule-name AllowAzureServices \
    --start-ip-address 0.0.0.0 \
    --end-ip-address 0.0.0.0 \
    --output none
fi

# Build DATABASE_URL
# NOTE: Currently uses admin user for simplicity. For production workloads,
# create a dedicated app role with limited permissions (SELECT, INSERT, UPDATE, DELETE).
PG_FQDN=$(az postgres flexible-server show \
  --name "$PG_SERVER_NAME" \
  --resource-group "$RESOURCE_GROUP" \
  --query "fullyQualifiedDomainName" -o tsv)

# --- Step 6: Set DATABASE_URL as Container App secret ---
if [ -n "${PG_ADMIN_PASSWORD:-}" ]; then
  # Validate hex-only (same check as Step 5, covers the case where server
  # already existed but user is updating the secret with a new password)
  if ! printf '%s' "$PG_ADMIN_PASSWORD" | grep -qE '^[0-9a-fA-F]+$'; then
    echo "❌ Error: PG_ADMIN_PASSWORD must be hex-only (0-9, a-f)."
    echo "   Generate with: export PG_ADMIN_PASSWORD=\$(openssl rand -hex 20)"
    exit 1
  fi

  DATABASE_URL="postgres://${PG_ADMIN_USER}:${PG_ADMIN_PASSWORD}@${PG_FQDN}:5432/${PG_DB_NAME}?sslmode=require"

  echo "🔗 Setting DATABASE_URL on Container App..."
  az containerapp secret set \
    --name "$CONTAINER_APP_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --secrets "database-url=$DATABASE_URL" \
    --output none

  az containerapp update \
    --name "$CONTAINER_APP_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --set-env-vars "DATABASE_URL=secretref:database-url" \
    --output none
else
  # Verify the secret and env var binding exist when skipping password setup
  SECRET_EXISTS=$(az containerapp secret list \
    --name "$CONTAINER_APP_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --query "[?name=='database-url'] | length(@)" -o tsv)
  if [ "$SECRET_EXISTS" = "0" ]; then
    echo "❌ Error: PG_ADMIN_PASSWORD not set and 'database-url' secret is missing!"
    echo "   Set PG_ADMIN_PASSWORD and re-run to configure the database connection."
    exit 1
  else
    # Ensure env var binding is intact (repairs partial/failed previous runs)
    echo "⏭️  DATABASE_URL secret exists — ensuring env var binding is intact..."
    az containerapp update \
      --name "$CONTAINER_APP_NAME" \
      --resource-group "$RESOURCE_GROUP" \
      --set-env-vars "DATABASE_URL=secretref:database-url" \
      --output none
  fi
fi

echo ""
echo "✅ Infrastructure ready!"
echo "========================================="
echo "App URL:      https://$APP_URL"
echo "ACR:          $ACR_LOGIN_SERVER"
echo ""
echo "Next steps:"
echo "  1. Build & push your image:"
echo "     docker build -t $ACR_LOGIN_SERVER/$IMAGE_NAME:$IMAGE_TAG ."
echo "     az acr login --name $ACR_NAME"
echo "     docker push $ACR_LOGIN_SERVER/$IMAGE_NAME:$IMAGE_TAG"
echo ""
echo "  2. Update the container app:"
echo "     az containerapp update \\"
echo "       --name $CONTAINER_APP_NAME \\"
echo "       --resource-group $RESOURCE_GROUP \\"
echo "       --image $ACR_LOGIN_SERVER/$IMAGE_NAME:$IMAGE_TAG"
echo ""
echo "  3. Or just push to main — GitHub Actions will do it for you!"
