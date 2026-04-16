# Cloud Run Interactive Demo - Cloud Next '26

An interactive, real-time visualization of Cloud Run's serverless scalability. This demo allows hundreds of attendees to spin up their own isolated container instances, customize their appearance, and see them appear on a shared live dashboard.

![Stable High-Density Grid](https://via.placeholder.com/1920x1080?text=Live+Dashboard+Visualization)

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
Follow the detailed **[Deployment Guide (DEPLOY.md)](./DEPLOY.md)** for step-by-step instructions on setting up the infrastructure, services, and security rules.

### 3. Running the Simulation
To simulate a large-scale audience (e.g., 600 concurrent attendees):
```bash
./simulate_attendees.py --count 600
```

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
