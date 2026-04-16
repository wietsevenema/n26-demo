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

# Configuration
DEFAULT_WS_URL = "ws://34.160.220.162/ws"
DEFAULT_TARGET = 30
MAX_ALLOWED = 1000

# Shared state
active_threads = []
stop_event = threading.Event()

EMOJIS = ["🐶", "🐱", "🐭", "🐹", "🐰", "🦊", "🐻", "🐼", "🐻‍❄️", "🐨", "🐯", "🦁", "🐮", "🐷", "🐸", "🐵", "🦄", "🐝", "🐙"]
COLORS = ["#1967D2", "#C5221F", "#F29900", "#188038", "#5F6368"]

def attendee_worker(attendee_id, ws_url):
    """Simulates a single attendee lifecycle: join, interact, leave."""
    duration = random.uniform(30, 120) # Stay for 30-120 seconds for larger simulations
    start_time = time.time()
    
    try:
        ws = websocket.create_connection(ws_url)
        
        # Initial customization
        ws.send(json.dumps({
            "emoji": random.choice(EMOJIS),
            "color": random.choice(COLORS)
        }))

        # Interaction loop
        while not stop_event.is_set() and (time.time() - start_time) < duration:
            try:
                ws.settimeout(1.0)
                ws.recv() # Drain metrics
            except websocket.WebSocketTimeoutException:
                pass
            except Exception:
                break
            
            # 5% chance to interact to keep the grid "alive" without flooding
            if random.random() < 0.05:
                ws.send(json.dumps({
                    "emoji": random.choice(EMOJIS),
                    "color": random.choice(COLORS)
                }))
            
            time.sleep(random.uniform(5, 15))
            
        ws.close()
    except Exception as e:
        pass

def signal_handler(sig, frame):
    print("\nStopping simulation gracefully...")
    stop_event.set()
    # No need to join threads here, we'll just exit
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
    
    print(f"Starting simulation: Target {target} concurrent @ {ws_url}")
    
    attendee_counter = 0
    while not stop_event.is_set():
        # Clean up finished threads
        active_threads = [t for t in active_threads if t.is_alive()]
        current_count = len(active_threads)
        
        if current_count < target:
            # Scale spawning rate based on distance to target
            diff = target - current_count
            to_spawn = max(1, min(10, diff // 2)) 
            
            for _ in range(to_spawn):
                attendee_counter += 1
                t = threading.Thread(target=attendee_worker, args=(attendee_counter, ws_url))
                t.daemon = True
                t.start()
                active_threads.append(t)
            
            print(f"Current: {len(active_threads)} / Target: {target}", end='\r')
        
        time.sleep(0.5)
