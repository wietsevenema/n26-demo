# Cloud Run Interactive Demo - Cloud Next '26

An interactive, real-time visualization of Cloud Run's serverless scalability. This demo allows hundreds of attendees to spin up their own isolated container instances, customize their appearance, and see them appear on a shared live dashboard.

## Key Features

- **Isolated Instances:** Each attendee connection triggers a dedicated Cloud Run container (configured with `max-concurrency=1`).
- **Real-time Synchronization:** Container state (emoji, color, memory metrics) is synced to Firestore and streamed to the dashboard using the Firebase JS SDK.
- **Stable High-Density Grid:** A dynamic, viewport-optimized layout that maintains fixed positions for instances, filling gaps as attendees join and leave.
- **Interactive Customization:** Attendees can change their container's emoji and background color via WebSockets and HTMX.

## Architecture

- **Backend:** Go (Golang) server handling WebSockets, SSE, and Firestore state management.
- **Frontend:**
  - **Attendee UI:** HTMX for minimal, high-performance interactions.
  - **Presentation Dashboard:** Firebase JS SDK for seamless, zero-flicker real-time grid updates.
- **Infrastructure:**
  - **Cloud Run:** Highly scalable, isolated backend services.
  - **Cloud Storage + CDN:** High-performance static asset hosting.
  - **Global External Load Balancer:** Unified routing for WebSockets, SSE, and static files.
  - **Firestore:** Global, serverless real-time database.

## Quick Start

### 1. Prerequisites
- Google Cloud Project with Billing enabled.
- `gcloud` CLI installed and authenticated.
- Docker for local builds.
- Python 3.12+ (with `uv`) for simulation.

### 2. Deployment
Follow the detailed **[Project Setup (DEPLOY.md)](./DEPLOY.md)** for initial infrastructure provisioning and the **[Component Update Guide (DEPLOYMENT_GUIDE.md)](./DEPLOYMENT_GUIDE.md)** for pushing changes to the backend and frontend.

### 3. Simulating Attendees
The project includes a Python script to simulate hundreds of concurrent attendees. It uses the `uv` script runner to manage its own dependencies.

**Basic Usage:**
```bash
./simulate_attendees.py --count 100
```

**Available Flags:**
- `--count`: The target number of concurrent attendees to maintain (default: 30).
- `--url`: The WebSocket endpoint of your deployment (default: `ws://34.160.220.162/ws`).

### 4. Monitoring & Logs
You can monitor the live behavior of the container instances (connections, customizations, and terminations) using Cloud Logging.

**View Live Backend Logs:**
```bash
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=attendee-backend" --limit 50 --format="table(timestamp,textPayload)" --project venema-2026-1
```

**Cloud Console:**
Visit the [Cloud Run Console](https://console.cloud.google.com/run/detail/us-west4/attendee-backend/logs) to view the Logs Explorer with advanced filtering and real-time streaming.

## Repository Structure

```text
├── backend/            # Go backend source & Dockerfile
├── frontend/           # Static assets (Attendee & Presentation UI)
├── DEPLOY.md           # Step-by-step deployment instructions
├── simulate_attendees.py # Python script for load simulation
└── firestore.rules     # Security rules for public read access
```

## Credits
Built for **Google Cloud Next '26** to showcase the power and speed of Cloud Run.
