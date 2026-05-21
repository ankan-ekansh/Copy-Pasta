#!/usr/bin/env bash
# =============================================================================
# Copy-Pasta — Azure Infrastructure Setup
# Creates all Azure resources needed for deployment.
#
# Prerequisites:
#   - Azure CLI installed and logged in (az login)
#   - A subscription with credits available
#
# Usage:
#   chmod +x infra/setup-azure.sh
#   ./infra/setup-azure.sh
# =============================================================================

set -euo pipefail

# --- Configuration (customize these) ---
RESOURCE_GROUP="rg-copy-pasta"
LOCATION="eastus"
ACR_NAME="copypastacr"  # must be globally unique, lowercase, no hyphens
CONTAINER_APP_ENV="cae-copy-pasta"
CONTAINER_APP_NAME="copy-pasta"
IMAGE_NAME="copy-pasta"
IMAGE_TAG="latest"

echo "🍝 Copy-Pasta Azure Infrastructure Setup"
echo "========================================="
echo "Resource Group: $RESOURCE_GROUP"
echo "Location:       $LOCATION"
echo "ACR:            $ACR_NAME.azurecr.io"
echo ""

# --- Step 0: Register required resource providers ---
PROVIDERS=("Microsoft.App" "Microsoft.OperationalInsights" "Microsoft.ContainerRegistry")
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
