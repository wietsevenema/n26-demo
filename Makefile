# Deployment Configuration
PROJECT_ID ?= venema-next-2026-demo
REGION ?= us-west4
BUCKET_NAME ?= $(PROJECT_ID)-frontend
DOMAIN ?= n26.wietsevenema.eu

# Colors for output
BLUE := \033[0;34m
NC := \033[0m

.PHONY: all deploy backend frontend cleanup rules-deploy

all: deploy

deploy: backend frontend cleanup rules-deploy
	@echo "$(BLUE)Full deployment complete!$(NC)"
	@echo "IP Address: $$(gcloud compute forwarding-rules describe demo-https-rule --global --format='value(IPAddress)')"

backend:
	@echo "$(BLUE)Building and deploying backend services...$(NC)"
	gcloud auth configure-docker $(REGION)-docker.pkg.dev --quiet
	cd backend && docker build --platform linux/amd64 -t $(REGION)-docker.pkg.dev/$(PROJECT_ID)/cloud-run-demo/backend .
	docker push $(REGION)-docker.pkg.dev/$(PROJECT_ID)/cloud-run-demo/backend
	gcloud run deploy attendee-backend \
		--image $(REGION)-docker.pkg.dev/$(PROJECT_ID)/cloud-run-demo/backend \
		--service-account="attendee-sa@$(PROJECT_ID).iam.gserviceaccount.com" \
		--region $(REGION) --quiet

frontend:
	@echo "$(BLUE)Preparing and uploading frontend assets...$(NC)"
	# Create a temporary directory for processed frontend files
	mkdir -p .tmp-frontend
	cp frontend/* .tmp-frontend/
	# Replace Project ID placeholder in presentation.html
	sed -i '' 's/YOUR_PROJECT_ID/$(PROJECT_ID)/g' .tmp-frontend/presentation.html
	# Upload to GCS
	gcloud storage cp .tmp-frontend/* gs://$(BUCKET_NAME)/
	# Invalidate CDN cache
	gcloud compute url-maps invalidate-cdn-cache demo-url-map --path "/*" --async
	rm -rf .tmp-frontend

cleanup:
	@echo "$(BLUE)Building and deploying cleanup service...$(NC)"
	cd cleanup && docker build --platform linux/amd64 -t $(REGION)-docker.pkg.dev/$(PROJECT_ID)/cloud-run-demo/cleanup .
	docker push $(REGION)-docker.pkg.dev/$(PROJECT_ID)/cloud-run-demo/cleanup
	gcloud run deploy cleanup-backend \
		--image $(REGION)-docker.pkg.dev/$(PROJECT_ID)/cloud-run-demo/cleanup \
		--service-account="cleanup-sa@$(PROJECT_ID).iam.gserviceaccount.com" \
		--region $(REGION) --quiet

rules-deploy:
	@echo "$(BLUE)Deploying Firestore security rules...$(NC)"
	firebase deploy --only firestore:rules --project $(PROJECT_ID)

logs:
	@echo "$(BLUE)Fetching latest logs for attendee-backend...$(NC)"
	$(eval REVISION=$(shell gcloud run revisions list --service attendee-backend --region $(REGION) --limit 1 --format="value(name)"))
	gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=attendee-backend AND resource.labels.revision_name=$(REVISION)" --limit 50 --format="table(timestamp,textPayload)"
