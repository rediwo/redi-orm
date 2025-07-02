// Simple assert module for testing
module.exports = {
    // Basic assertion
    assert: function(condition, message) {
        if (!condition) {
            throw new Error(message || 'Assertion failed');
        }
    },
    
    // Strict equality
    strictEqual: function(actual, expected, message) {
        if (actual !== expected) {
            throw new Error(message || `Expected ${expected}, got ${actual}`);
        }
    },
    
    // Deep equality (simplified)
    deepEqual: function(actual, expected, message) {
        if (JSON.stringify(actual) !== JSON.stringify(expected)) {
            throw new Error(message || `Expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
        }
    },
    
    // Fail explicitly
    fail: function(message) {
        throw new Error(message || 'Test failed');
    }
};

// Export individual functions too
module.exports.ok = module.exports.assert;
module.exports.equal = module.exports.strictEqual;