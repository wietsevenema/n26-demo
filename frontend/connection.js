// frontend/connection.js

// Pure function for easy unit testing
function calculateBackoffDelay(retryCount) {
    // Base delay: 1s, 2s, 4s, 8s, 16s, max 32s
    const baseDelay = Math.min(32000, 1000 * Math.pow(2, retryCount));
    // Jitter: 0 to 20% of base delay
    const jitter = Math.floor(Math.random() * (baseDelay * 0.2));
    return baseDelay + jitter;
}

// Only execute DOM-related code if we are in a browser environment
if (typeof window !== 'undefined') {
    let retryInterval;

    window.setConnectionStatus = function(message, type = 'info') {
        let statusEl = document.getElementById('connection-status');
        let indicator = document.getElementById('pulse-indicator');

        // If the backend replaced container-preview, the status element is gone.
        // We recreate the original structure to hide the stale container and show the status.
        if (!statusEl && type !== 'connected') {
            const previewWrapper = document.getElementById('preview-wrapper');
            if (previewWrapper) {
                previewWrapper.innerHTML = `
                    <div id="pulse-indicator" class="pulse-dot"></div>
                    <div id="container-preview">
                        <div id="connection-status" class="status-placeholder">${message}</div>
                    </div>
                `;
                statusEl = document.getElementById('connection-status');
                indicator = document.getElementById('pulse-indicator');
            }
        }

        if (statusEl) {
            statusEl.innerHTML = message;
        }

        if (indicator) {
            indicator.className = 'pulse-dot'; // Reset
            if (type === 'error') indicator.classList.add('error');
            else if (type === 'connecting') indicator.classList.add('connecting');
            else if (type === 'connected') indicator.classList.add('connected');
        }
    };
    window.startContainer = function() {
        document.getElementById('setup-view').style.display = 'none';
        const activeView = document.getElementById('active-view');
        activeView.style.display = 'block';
        
        setConnectionStatus('Booting container...', 'connecting');
        
        activeView.setAttribute('hx-ext', 'ws');
        activeView.setAttribute('ws-connect', '/ws');
        
        if (window.htmx) {
            htmx.process(activeView);
        }
    };

    window.handleSelection = function(event) {
        const btn = event.target.closest('button');
        if (!btn) return;
        
        // Remove selected from siblings
        const grid = btn.parentElement;
        Array.from(grid.querySelectorAll('button')).forEach(b => b.classList.remove('selected'));
        
        // Add to clicked
        btn.classList.add('selected');
    };

    window.updateSelectionVisuals = function(emoji, color) {
        // Update Emoji
        const emojiGrid = document.querySelector('.emoji-grid');
        if (emojiGrid) {
            Array.from(emojiGrid.querySelectorAll('button')).forEach(b => {
                if (b.getAttribute('value') === emoji) b.classList.add('selected');
                else b.classList.remove('selected');
            });
        }
        
        // Update Color
        const colorGrid = document.querySelector('.color-grid');
        if (colorGrid) {
            Array.from(colorGrid.querySelectorAll('button')).forEach(b => {
                if (b.getAttribute('value') === color) b.classList.add('selected');
                else b.classList.remove('selected');
            });
        }
    };

    const hideStatus = function() {
        console.log("Connection verified, hiding status bar.");
        clearInterval(retryInterval);
        setConnectionStatus('', 'connected');
    };

    document.addEventListener('htmx:wsOpen', hideStatus);
    document.addEventListener('htmx:wsAfterMessage', hideStatus);
    document.addEventListener('htmx:wsConfigSend', hideStatus);

    // Check for htmx globally
    if (window.htmx || document.readyState === 'loading') {
        const setupHtmx = () => {
            if (window.htmx) {
                htmx.config.wsReconnectDelay = function(retryCount) {
                    const totalDelayMs = calculateBackoffDelay(retryCount);
                    let secondsLeft = Math.ceil(totalDelayMs / 1000);
                    
                    setConnectionStatus(`Connection failed. Retrying in <span id="retry-timer">${secondsLeft}</span>s...`, 'error');
                    
                    clearInterval(retryInterval);
                    retryInterval = setInterval(() => {
                        secondsLeft--;
                        const timerSpan = document.getElementById('retry-timer');
                        if (secondsLeft > 0 && timerSpan) {
                            timerSpan.innerText = secondsLeft;
                        } else {
                            setConnectionStatus("Reconnecting now...", 'connecting');
                            clearInterval(retryInterval);
                        }
                    }, 1000);
                    
                    return totalDelayMs;
                };
            }
        };
        
        if (window.htmx) {
            setupHtmx();
        } else {
            document.addEventListener('DOMContentLoaded', setupHtmx);
        }
    }
}

// Export for Node.js testing
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { calculateBackoffDelay };
}
