# Deployment Guide

This guide provides step-by-step instructions to deploy the interactive Cloud Run demo to a fresh Google Cloud project.

## 1. Project Initialization

### Environment Variables
Set these variables in your terminal to make the commands easier to run:
```bash
PROJECT_ID="your-project-id"
REGION="us-west4" # Las Vegas
BUCKET_NAME="${PROJECT_ID}-frontend"
```

### Enable APIs
```bash
gcloud config set project $PROJECT_ID

gcloud services enable \
    run.googleapis.com \
    cloudbuild.googleapis.com \
    compute.googleapis.com \
    firestore.googleapis.com \
    artifactregistry.googleapis.com
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
Build the image locally (faster) and push it:
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

### Deploy Cloud Run Services
We deploy the same code to two services with different configurations:

**1. Attendee Backend (Isolated instances)**
```bash
gcloud run deploy attendee-backend \
    --image ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/backend \
    --concurrency 1 \
    --allow-unauthenticated \
    --region $REGION \
    --max-instances 1000
```

**2. Presentation Backend (Dashboard SSE stream)**
```bash
gcloud run deploy presentation-backend \
    --image ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/backend \
    --allow-unauthenticated \
    --region $REGION \
    --max-instances 10
```

---

## 3. Frontend Deployment

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

## 4. Global Load Balancer Setup

### Create NEGs and Backend Services
```bash
# NEGs
gcloud compute network-endpoint-groups create attendee-neg \
    --region=$REGION --network-endpoint-type=serverless --cloud-run-service=attendee-backend

gcloud compute network-endpoint-groups create presentation-neg \
    --region=$REGION --network-endpoint-type=serverless --cloud-run-service=presentation-backend

# Backend Services
gcloud compute backend-services create attendee-backend-service --global --load-balancing-scheme=EXTERNAL_MANAGED
gcloud compute backend-services add-backend attendee-backend-service --global \
    --network-endpoint-group=attendee-neg --network-endpoint-group-region=$REGION

gcloud compute backend-services create presentation-backend-service --global --load-balancing-scheme=EXTERNAL_MANAGED
gcloud compute backend-services add-backend presentation-backend-service --global \
    --network-endpoint-group=presentation-neg --network-endpoint-group-region=$REGION

# Backend Bucket (CDN)
gcloud compute backend-buckets create demo-backend-bucket --gcs-bucket-name=$BUCKET_NAME --enable-cdn
```

### Configure URL Map and Forwarding
```bash
# URL Map
gcloud compute url-maps create demo-url-map --default-backend-bucket=demo-backend-bucket

# Path Rules
gcloud compute url-maps add-path-matcher demo-url-map \
    --default-backend-bucket=demo-backend-bucket \
    --path-matcher-name=backend-matcher \
    --new-hosts="*" \
    --backend-service-path-rules="/ws=attendee-backend-service,/presentation=presentation-backend-service"

# Proxy and Forwarding Rule
gcloud compute target-http-proxies create demo-http-proxy --url-map=demo-url-map
gcloud compute forwarding-rules create demo-forwarding-rule --global \
    --target-http-proxy=demo-http-proxy --ports=80 --load-balancing-scheme=EXTERNAL_MANAGED
```

---

## 5. Manual Configuration (Crucial)

### Firestore Security Rules
Go to the [Google Cloud Console (Firestore Rules)](https://console.cloud.google.com/firestore/databases/-default-/rules) and apply these rules to allow the dashboard to read data:

```javascript
rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    match /active_containers/{container} {
      allow read: if true;
      allow write: if false; 
    }
  }
}
```

### Firestore TTL Policy
To automatically clean up stale instances (e.g., if a container crashes and misses its cleanup signal):
1. Go to [Firestore TTL Settings](https://console.cloud.google.com/firestore/databases/-default-/settings).
2. Click **Enable TTL**.
3. Select the collection: `active_containers`.
4. Select the timestamp field: `ttl`.
5. Click **Enable**.
*Note: It can take up to 72 hours for the initial cleanup to begin after enabling the policy.*

### Presentation Frontend
Update the `projectId` in `frontend/presentation.html` to match your project if it was hardcoded during development.

---

## 6. Testing

### Get Public IP
```bash
gcloud compute forwarding-rules describe demo-forwarding-rule --global --format="value(IPAddress)"
```

### Run Simulation
Install dependencies and run the script:
```bash
uv run simulate_attendees.py
```
Visit `http://<IP_ADDRESS>/presentation.html` to see the results.
