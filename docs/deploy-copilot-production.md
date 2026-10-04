# Deploy Tayooli Copilot to Production

## Target Server: tayooli-server (GCP)
- IP: 104.197.178.237
- Zone: us-central1-c
- Project: mbg-waste-tracker-503014

## Step 1: Build & Upload Backend

```bash
# From your local machine (where code is)
cd backend/go-core

# Build the binary
GOOS=linux GOARCH=amd64 go build -o tayooli-api ./cmd/api

# Upload to server
gcloud compute scp tayooli-api tayooli-server:/tmp/tayooli-api --zone=us-central1-c

# SSH into server and replace binary
gcloud compute ssh tayooli-server --zone=us-central1-c --command="sudo cp /tmp/tayooli-api /opt/tayooli/tayooli-api && sudo chmod 755 /opt/tayooli/tayooli-api && sudo systemctl restart tayooli-backend"
```

## Step 2: Build & Upload Frontend

```bash
# From your local machine
cd frontend  # or root if frontend is in root

# Build
npm run build

# Upload .next folder
gcloud compute scp --recurse .next tayooli-server:/opt/tayooli/frontend/.next --zone=us-central1-c

# Upload other necessary files (package.json, next.config.js, etc.)
gcloud compute scp package.json next.config.js tayooli-server:/opt/tayooli/frontend/ --zone=us-central1-c

# SSH and restart frontend
gcloud compute ssh tayooli-server --zone=us-central1-c --command="sudo systemctl restart tayooli-frontend"
```

## Step 3: Verify Deployment

```bash
# Check backend health
curl -sf http://104.197.178.237:8081/health

# Test copilot endpoint
curl -sf -X POST http://104.197.178.237:8081/api/v1/copilot/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"halo","context":{"tenantId":"550e8400-e29b-41d4-a716-446655440000","userRole":"admin"}}'

# Check live domain
curl -sf https://tayooli.my.id/api/v1/copilot/chat \
  -X POST -H "Content-Type: application/json" \
  -d '{"message":"halo","context":{"tenantId":"550e8400-e29b-41d4-a716-446655440000","userRole":"admin"}}'
```

## Step 4: Set GEMINI_API_KEY on Server

```bash
# SSH into server
gcloud compute ssh tayooli-server --zone=us-central1-c

# Add to systemd service environment
sudo systemctl edit tayooli-backend --force
# Add: Environment="GEMINI_API_KEY=your_key_here"

# Or set in /etc/default/tayooli-backend
echo "GEMINI_API_KEY=your_key_here" | sudo tee /etc/default/tayooli-backend

# Restart
sudo systemctl restart tayooli-backend
```

## Step 5: Full Browser Test

```bash
# After deployment, run the browser test against live domain
node test-copilot-browser.js --live
```
