#!/usr/bin/env python3
import os
import sys
import json

# Get arguments from environment
args_json = os.environ.get('EXTENSION_ARGS', '{}')
args = json.loads(args_json)

# Get name argument
name = args.get('name', 'World')

# Generate output
output = {
    "success": True,
    "output": f"Hello, {name}! This is Python extension.",
    "exit_code": 0,
    "data": {
        "python_version": sys.version.split()[0],
        "args_received": args
    }
}

# Print result as JSON
print(json.dumps(output))
sys.exit(0)