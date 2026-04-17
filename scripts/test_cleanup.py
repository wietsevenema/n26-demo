#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "google-cloud-firestore",
# ]
# ///

import time
from datetime import datetime, timedelta, timezone
from google.cloud import firestore

PROJECT_ID = os.getenv("GOOGLE_CLOUD_PROJECT")
TEST_ID = "test-zombie-999"

def run_test():
    db = firestore.Client(project=PROJECT_ID)
    doc_ref = db.collection("active_containers").document(TEST_ID)
    
    # 1. Insert an expired record (expired 5 minutes ago)
    expired_ttl = datetime.now(timezone.utc) - timedelta(minutes=5)
    print(f"[*] Inserting expired record {TEST_ID} with TTL: {expired_ttl}")
    
    doc_ref.set({
        "instance_id": TEST_ID,
        "emoji": "💀",
        "color": "#000000",
        "status": "zombie",
        "ttl": expired_ttl
    })

    print("[*] Waiting 70 seconds for the Cloud Scheduler job to run...")
    # Cloud Scheduler runs on the minute (e.g., 10:21:00, 10:22:00)
    # Waiting 70s guarantees we cross a minute boundary
    time.sleep(70)

    # 2. Check if it's gone
    doc = doc_ref.get()
    if not doc.exists:
        print(f"[SUCCESS] Record {TEST_ID} was successfully cleaned up!")
    else:
        print(f"[FAILURE] Record {TEST_ID} still exists in Firestore.")
        print(f"Data: {doc.to_dict()}")

if __name__ == "__main__":
    run_test()
