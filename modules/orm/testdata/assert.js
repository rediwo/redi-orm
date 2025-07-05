// Test assertion utilities that integrate with the test framework

const assert = require('assert');

// Re-export built-in assert functions
module.exports = assert;

// Add deepEqual as an alias (it's already in the built-in assert)
module.exports.deepEqual = assert.deepStrictEqual || assert.deepEqual;

// Add custom assertions
module.exports.match = function(actual, regex, message) {
    if (!regex.test(actual)) {
        throw new Error(message || `Expected "${actual}" to match ${regex}`);
    }
};

module.exports.includes = function(array, value, message) {
    if (!array.includes(value)) {
        throw new Error(message || `Expected array to include ${JSON.stringify(value)}`);
    }
};

module.exports.lengthOf = function(array, length, message) {
    if (array.length !== length) {
        throw new Error(message || `Expected array length ${length}, got ${array.length}`);
    }
};

module.exports.isNull = function(value, message) {
    if (value !== null) {
        throw new Error(message || `Expected null, got ${JSON.stringify(value)}`);
    }
};

module.exports.isNotNull = function(value, message) {
    if (value === null) {
        throw new Error(message || `Expected non-null value`);
    }
};

module.exports.isUndefined = function(value, message) {
    if (value !== undefined) {
        throw new Error(message || `Expected undefined, got ${JSON.stringify(value)}`);
    }
};

module.exports.isDefined = function(value, message) {
    if (value === undefined) {
        throw new Error(message || `Expected defined value`);
    }
};

module.exports.isAbove = function(actual, expected, message) {
    if (!(actual > expected)) {
        throw new Error(message || `Expected ${actual} to be above ${expected}`);
    }
};

module.exports.isBelow = function(actual, expected, message) {
    if (!(actual < expected)) {
        throw new Error(message || `Expected ${actual} to be below ${expected}`);
    }
};

module.exports.isAtLeast = function(actual, expected, message) {
    if (!(actual >= expected)) {
        throw new Error(message || `Expected ${actual} to be at least ${expected}`);
    }
};

module.exports.isAtMost = function(actual, expected, message) {
    if (!(actual <= expected)) {
        throw new Error(message || `Expected ${actual} to be at most ${expected}`);
    }
};

module.exports.instanceOf = function(object, constructor, message) {
    if (!(object instanceof constructor)) {
        throw new Error(message || `Expected instance of ${constructor.name}`);
    }
};

module.exports.property = function(object, property, message) {
    if (!(property in object)) {
        throw new Error(message || `Expected object to have property "${property}"`);
    }
};

module.exports.notProperty = function(object, property, message) {
    if (property in object) {
        throw new Error(message || `Expected object not to have property "${property}"`);
    }
};

// Helper to check if an error contains specific message
module.exports.errorContains = function(error, substring, message) {
    if (!error || !error.message || !error.message.includes(substring)) {
        throw new Error(message || `Expected error to contain "${substring}", got: ${error?.message || 'no error'}`);
    }
};

// Async assertion helper
module.exports.rejects = async function(promise, errorCheck, message) {
    try {
        await promise;
        throw new Error(message || 'Expected promise to reject');
    } catch (err) {
        if (errorCheck && typeof errorCheck === 'function') {
            errorCheck(err);
        } else if (errorCheck && typeof errorCheck === 'string') {
            if (!err.message.includes(errorCheck)) {
                throw new Error(`Expected error to contain "${errorCheck}", got: ${err.message}`);
            }
        }
    }
};

// Test result logging
module.exports.pass = function(testName) {
    console.log(`✓ ${testName}`);
};

module.exports.fail = function(testName, error) {
    console.error(`✗ ${testName}`);
    if (error) {
        console.error(`  ${error.message}`);
        if (error.stack) {
            console.error(error.stack.split('\n').slice(1).join('\n'));
        }
    }
};