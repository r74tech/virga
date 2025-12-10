#!/usr/bin/env node

// Get arguments from environment
const args = JSON.parse(process.env.EXTENSION_ARGS || '{}');

// Get name argument
const name = args.name || 'World';

// Generate output
const output = {
    success: true,
    output: `Hello, ${name}! This is JavaScript extension.`,
    exit_code: 0,
    data: {
        node_version: process.version,
        args_received: args
    }
};

// Print result as JSON
console.log(JSON.stringify(output));
process.exit(0);