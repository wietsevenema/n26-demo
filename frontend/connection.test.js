// frontend/connection.test.js
const test = require('node:test');
const assert = require('node:assert');
const { calculateBackoffDelay } = require('./connection.js');

test('calculateBackoffDelay - Retry 0 (1 second base)', (t) => {
    // Retry 0: baseDelay = 1000. Jitter = 0 to 200. Total = 1000 to 1200.
    const delay = calculateBackoffDelay(0);
    assert.ok(delay >= 1000 && delay <= 1200, `Delay ${delay} should be between 1000 and 1200`);
});

test('calculateBackoffDelay - Retry 1 (2 second base)', (t) => {
    // Retry 1: baseDelay = 2000. Jitter = 0 to 400. Total = 2000 to 2400.
    const delay = calculateBackoffDelay(1);
    assert.ok(delay >= 2000 && delay <= 2400, `Delay ${delay} should be between 2000 and 2400`);
});

test('calculateBackoffDelay - Retry 5 (Maximum 32 second base)', (t) => {
    // Math.pow(2, 5) * 1000 = 32000.
    // Retry 5: baseDelay = 32000. Jitter = 0 to 6400. Total = 32000 to 38400.
    const delay = calculateBackoffDelay(5);
    assert.ok(delay >= 32000 && delay <= 38400, `Delay ${delay} should be between 32000 and 38400`);
});

test('calculateBackoffDelay - Retry 10 (Should still cap at 32 seconds)', (t) => {
    // Math.pow(2, 10) * 1000 = 1024000, but cap is 32000.
    // Retry 10: baseDelay = 32000. Jitter = 0 to 6400. Total = 32000 to 38400.
    const delay = calculateBackoffDelay(10);
    assert.ok(delay >= 32000 && delay <= 38400, `Delay ${delay} should be between 32000 and 38400`);
});
