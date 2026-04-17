# Deployment Guide

This guide provides step-by-step instructions to deploy the interactive Cloud Run demo to a fresh Google Cloud project.

## 1. Project Initialization

### Environment Variables
Set these variables in your terminal to make the commands easier to run:
```bash
PROJECT_ID="your-project-id"
REGION="us-west4" # Las Vegas
BUCKET_NAME="${PROJECT_ID}-frontend"
DOMAIN="n26.wietsevenema.eu" # Optional: For HTTPS setup
```

### Enable APIs
```bash
gcloud config set project $PROJECT_ID

gcloud services enable \
    run.googleapis.com \
    cloudbuild.googleapis.com \
    compute.googleapis.com \
    firestore.googleapis.com \
    artifactregistry.googleapis.com \
    cloudscheduler.googleapis.com
```

### Initialize Firestore
1. Create the Firestore database in **Native Mode**:
```bash
gcloud firestore databases create --location=$REGION --type=firestore-native
```

2. Enable the **TTL policy** to automatically clean up "ghost" records using the `ttl` field:
```bash
gcloud alpha firestore fields ttls update ttl \
    --collection-group=active_containers \
    --database='(default)' \
    --enable-ttl
```

---

## 2. Backend Deployment

### Create Artifact Registry
```bash
gcloud artifacts repositories create cloud-run-demo \
    --repository-format=docker \
    --location=$REGION \
    --description="Docker repository for Cloud Run Demo"
```

### Build and Push Image
Build the image locally and push it. A local build is **always faster** than Google Cloud Build, as the latter does not cache layers between builds.

```bash
# Authenticate Docker
gcloud auth configure-docker ${REGION}-docker.pkg.dev

# Build (from project root)
cd backend
docker build --platform linux/amd64 -t ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/backend .

# Push
docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/backend
cd ..
```

### Deploy Cloud Run Service

**Attendee Backend (Isolated instances)**
```bash
gcloud run deploy attendee-backend \
    --image ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/backend \
    --concurrency 1 \
    --cpu 0.08 \
    --memory 128Mi \
    --timeout 900 \
    --allow-unauthenticated \
    --region $REGION \
    --max-instances 1000
```

---

## 3. Frontend Deployment

### Update Project ID
Update the `projectId` in `frontend/presentation.html` to match your `$PROJECT_ID`.

### Create and Configure Bucket
```bash
# Create bucket
gcloud storage buckets create gs://$BUCKET_NAME --location=$REGION

# Enable website hosting
gcloud storage buckets update gs://$BUCKET_NAME --web-main-page-suffix=index.html

# Make public
gcloud storage buckets add-iam-policy-binding gs://$BUCKET_NAME \
    --member=allUsers \
    --role=roles/storage.objectViewer

# Upload files
gcloud storage cp frontend/* gs://$BUCKET_NAME/
```

---

## 4. Global Load Balancer & HTTPS Setup

### Create NEGs and Backend Services
```bash
# NEG
gcloud compute network-endpoint-groups create attendee-neg \
    --region=$REGION --network-endpoint-type=serverless --cloud-run-service=attendee-backend

# Backend Service
gcloud compute backend-services create attendee-backend-service --global --load-balancing-scheme=EXTERNAL_MANAGED
gcloud compute backend-services add-backend attendee-backend-service --global \
    --network-endpoint-group=attendee-neg --network-endpoint-group-region=$REGION

# Backend Bucket (CDN)
gcloud compute backend-buckets create demo-backend-bucket --gcs-bucket-name=$BUCKET_NAME --enable-cdn
```

### Reserve Static IP
```bash
gcloud compute addresses create demo-ip --global --network-tier=PREMIUM
```

### Configure Main URL Map (HTTPS)
```bash
# URL Map
gcloud compute url-maps create demo-url-map --default-backend-bucket=demo-backend-bucket

# Path Rules
gcloud compute url-maps add-path-matcher demo-url-map \
    --default-backend-bucket=demo-backend-bucket \
    --path-matcher-name=backend-matcher \
    --new-hosts="*" \
    --backend-service-path-rules="/ws=attendee-backend-service"

# SSL Certificate (Google-managed)
gcloud compute ssl-certificates create demo-ssl-cert --domains="$DOMAIN" --global

# Target HTTPS Proxy
gcloud compute target-https-proxies create demo-https-proxy --url-map=demo-url-map --ssl-certificates=demo-ssl-cert

# Forwarding Rule (Port 443)
gcloud compute forwarding-rules create demo-https-rule --global \
    --target-https-proxy=demo-https-proxy --ports=443 --load-balancing-scheme=EXTERNAL_MANAGED --address=demo-ip
```

### Configure HTTP-to-HTTPS Redirect (Port 80)
```bash
# Create Redirect URL Map
cat << EOF > web-map-http.yaml
kind: compute#urlMap
name: web-map-http
defaultUrlRedirect:
  redirectResponseCode: MOVED_PERMANENTLY_DEFAULT
  httpsRedirect: True
EOF
gcloud compute url-maps import web-map-http --source web-map-http.yaml --global

# Target HTTP Proxy
gcloud compute target-http-proxies create demo-http-redirect-proxy --url-map=web-map-http

# Forwarding Rule (Port 80)
gcloud compute forwarding-rules create demo-http-rule --global \
    --target-http-proxy=demo-http-redirect-proxy --ports=80 --load-balancing-scheme=EXTERNAL_MANAGED --address=demo-ip
```

---

## 5. Security and Maintenance

### Firestore Security Rules
Use the Firebase CLI to deploy rules safely:

1. Create a `firebase.json`:
```json
{
  "firestore": {
    "rules": "firestore.rules"
  }
}
```

2. Deploy:
```bash
firebase deploy --only firestore:rules --project $PROJECT_ID
```

### Fast TTL Cleanup (Cloud Run + Scheduler)
Since native Firestore TTL is slow, we use a dedicated internal service for 1-minute cleanup:

1. **Build and Deploy Cleanup Service:**
   Local builds are **always faster** than Google Cloud Build, as the latter does not cache layers.
   ```bash
   cd cleanup
   docker build --platform linux/amd64 -t ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/cleanup .
   docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/cleanup
   gcloud run deploy cleanup-backend \
       --image ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/cleanup \
       --region $REGION --no-allow-unauthenticated --ingress internal
   ```

2. **Configure Scheduler:**
   ```bash
   # Create Service Account
   gcloud iam service-accounts create sweep-scheduler-sa --display-name="TTL Cleanup Scheduler"

   # Grant Permissions
   gcloud run services add-iam-policy-binding cleanup-backend \
       --region $REGION --member="serviceAccount:sweep-scheduler-sa@${PROJECT_ID}.iam.gserviceaccount.com" \
       --role="roles/run.invoker"

   # Create Job
   CLEANUP_URL=$(gcloud run services describe cleanup-backend --region $REGION --format="value(status.url)")
   gcloud scheduler jobs create http cleanup-zombies \
       --location $REGION --schedule="* * * * *" --uri="$CLEANUP_URL/" \
       --http-method=POST --oidc-service-account-email="sweep-scheduler-sa@${PROJECT_ID}.iam.gserviceaccount.com" \
       --oidc-token-audience="$CLEANUP_URL/"
   ```

---

## 6. Testing

### Get Public IP
```bash
gcloud compute forwarding-rules describe demo-https-rule --global --format="value(IPAddress)"
```

### Run Simulation
Install dependencies and run the script:
```bash
uv run simulate_attendees.py
```
Visit `https://$DOMAIN/presentation.html` to see the results.
