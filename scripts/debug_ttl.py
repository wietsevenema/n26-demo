#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "google-cloud-firestore",
# ]
# ///

from google.cloud import firestore
from datetime import datetime, timezone

def debug_ttl():
    db = firestore.Client(project="venema-2026-1")
    collection_ref = db.collection("active_containers")
    
    docs = list(collection_ref.stream())
    
    print(f"Total documents in 'active_containers': {len(docs)}")
    print("-" * 60)
    
    now = datetime.now(timezone.utc)
    
    for doc in docs:
        data = doc.to_dict()
        instance_id = data.get("instance_id", "N/A")
        ttl = data.get("ttl")
        
        if ttl:
            # Firestore timestamps are returned as datetime objects
            status = "EXPIRED" if ttl < now else "ACTIVE"
            diff = ttl - now
            print(f"ID: {instance_id} | TTL: {ttl} ({status}) | Expires in: {diff}")
        else:
            print(f"ID: {instance_id} | TTL: MISSING")

if __name__ == "__main__":
    debug_ttl()
