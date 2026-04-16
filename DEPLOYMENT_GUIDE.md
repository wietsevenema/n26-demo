# Deployment Guide: Updating Components

This document explains how to deploy changes to the various parts of the Cloud Run demo.

## Prerequisites

Ensure you have your environment variables set for the current session:

```bash
PROJECT_ID="venema-2026-1"
REGION="us-west4"
```

---

## 1. Backend Updates (Cloud Run)

The backend code (Go) is shared between two services: `attendee-backend` and `presentation-backend`. Both must be updated when the Go code changes.

### Build and Push Image
```bash
cd backend
docker build --platform linux/amd64 -t ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/backend .
docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/backend
cd ..
```

### Update Services
```bash
# Update Attendee Service (Concurrency = 1)
gcloud run deploy attendee-backend \
    --image ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/backend \
    --region $REGION --project $PROJECT_ID

# Update Presentation Service
gcloud run deploy presentation-backend \
    --image ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/backend \
    --region $REGION --project $PROJECT_ID
```

---

## 2. Frontend Updates (Static Assets)

Static files are hosted in Google Cloud Storage and served via a Global External Load Balancer with Cloud CDN.

### Upload Files
```bash
gcloud storage cp frontend/* gs://${PROJECT_ID}-frontend/ --project $PROJECT_ID
```

### Invalidate CDN Cache
After uploading, you must invalidate the cache to see changes immediately. This can take a few minutes to propagate globally.
```bash
gcloud compute url-maps invalidate-cdn-cache demo-url-map \
    --path "/*" --async --project $PROJECT_ID
```

**Tip:** During development, you can bypass the cache by adding a version parameter to the URL: `http://<IP>/presentation.html?v=2`.

---

## 3. Cleanup Service Updates

If you modify the logic in the `cleanup/` directory that handles long-term ghost record removal:

```bash
cd cleanup
docker build --platform linux/amd64 -t ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/cleanup .
docker push ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/cleanup
gcloud run deploy cleanup-backend \
    --image ${REGION}-docker.pkg.dev/${PROJECT_ID}/cloud-run-demo/cleanup \
    --region $REGION --project $PROJECT_ID
cd ..
```

---

## 4. Firestore Security Rules

If you modify `firestore.rules`, apply them via the [Cloud Console](https://console.cloud.google.com/firestore/databases/-default-/rules). 

**Current Required Rule:**
Allows public reads for the presentation dashboard.
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

---

## 5. Summary Table

| Component | Target | Deployment Tool |
| :--- | :--- | :--- |
| **Backend** | Cloud Run | Docker + `gcloud run deploy` |
| **Frontend** | GCS + CDN | `gcloud storage cp` + Cache Invalidation |
| **Cleanup** | Cloud Run | Docker + `gcloud run deploy` |
| **Rules** | Firestore | Cloud Console |
