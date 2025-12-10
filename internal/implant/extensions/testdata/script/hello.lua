#!/usr/bin/env lua

-- Get arguments from environment
local args_json = os.getenv("EXTENSION_ARGS") or "{}"

-- Simple JSON parsing (just extract name)
local name = args_json:match('"name"%s*:%s*"([^"]+)"') or "World"

-- Generate output
local output = string.format([[{
    "success": true,
    "output": "Hello, %s! This is Lua extension.",
    "exit_code": 0,
    "data": {
        "lua_version": "%s",
        "args_received": %s
    }
}]], name, _VERSION, args_json)

-- Print result
print(output)
os.exit(0)