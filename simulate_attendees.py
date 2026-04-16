#!/usr/bin/env -S uv run
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "websocket-client",
# ]
# ///

import websocket
import threading
import time
import signal
import sys
import random
import json
import argparse
import logging

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s [%(levelname)s] %(message)s',
    handlers=[logging.StreamFileHandler(sys.stdout) if hasattr(logging, "StreamFileHandler") else logging.StreamHandler(sys.stdout)]
)
logger = logging.getLogger("simulation")

# Configuration
DEFAULT_WS_URL = "ws://34.160.220.162/ws"
DEFAULT_TARGET = 30
MAX_ALLOWED = 1000

# Shared state
active_threads = []
stop_event = threading.Event()

EMOJIS = ["🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🐻", "🐼", "🐻‍❄️", "🐨", "🐯", "🦁", "🐮", "🐷", "🐸", "🐵", "🦄", "🐝", "🐙"]
COLORS = ["#1967D2", "#C5221F", "#F29900", "#188038"]

def attendee_worker(attendee_id, ws_url):
    """Simulates a single attendee lifecycle: join, interact, leave."""
    duration = random.uniform(300, 900) # Stay for 5-15 minutes for stability
    start_time = time.time()
    
    try:
        ws = websocket.create_connection(ws_url, timeout=10)
        logger.info(f"Attendee {attendee_id}: Connected.")
        
        # Initial customization
        ws.send(json.dumps({
            "emoji": random.choice(EMOJIS),
            "color": random.choice(COLORS)
        }))

        # Interaction loop
        while not stop_event.is_set() and (time.time() - start_time) < duration:
            try:
                # 1% chance to interact (very human, less flickery)
                if random.random() < 0.01:
                    if random.random() < 0.5:
                        ws.send(json.dumps({"emoji": random.choice(EMOJIS)}))
                    else:
                        ws.send(json.dumps({"color": random.choice(COLORS)}))
                
                # Drain metrics with a short timeout to stay responsive
                ws.settimeout(1.0)
                ws.recv() 
            except websocket.WebSocketTimeoutException:
                pass
            except Exception as e:
                logger.warning(f"Attendee {attendee_id}: Connection error: {e}")
                break
            
            time.sleep(random.uniform(10, 30))
            
        ws.close()
        logger.info(f"Attendee {attendee_id}: Disconnected normally.")
    except Exception as e:
        logger.error(f"Attendee {attendee_id}: Failed to connect: {e}")

def signal_handler(sig, frame):
    print("\nStopping simulation gracefully...")
    stop_event.set()
    sys.exit(0)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Simulate Cloud Run demo attendees.')
    parser.add_argument('--count', type=int, default=DEFAULT_TARGET, 
                        help=f'Target number of concurrent attendees (max {MAX_ALLOWED})')
    parser.add_argument('--url', type=str, default=DEFAULT_WS_URL, 
                        help='WebSocket URL')
    
    args = parser.parse_args()
    target = min(args.count, MAX_ALLOWED)
    ws_url = args.url

    signal.signal(signal.SIGINT, signal_handler)
    
    logger.info(f"Starting simulation: Target {target} concurrent @ {ws_url}")
    
    attendee_counter = 0
    while not stop_event.is_set():
        active_threads = [t for t in active_threads if t.is_alive()]
        current_count = len(active_threads)
        
        if current_count < target:
            diff = target - current_count
            to_spawn = max(1, min(10, diff // 2)) 
            
            for _ in range(to_spawn):
                attendee_counter += 1
                t = threading.Thread(target=attendee_worker, args=(attendee_counter, ws_url))
                t.daemon = True
                t.start()
                active_threads.append(t)
            
            logger.info(f"Simulation Status: {len(active_threads)} / {target}")
        
        time.sleep(1.0)
