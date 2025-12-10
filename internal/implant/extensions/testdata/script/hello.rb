#!/usr/bin/env ruby
require 'json'

# Get arguments from environment
args_json = ENV['EXTENSION_ARGS'] || '{}'
args = JSON.parse(args_json)

# Get name argument
name = args['name'] || 'World'

# Generate output
output = {
  'success' => true,
  'output' => "Hello, #{name}! This is Ruby extension.",
  'exit_code' => 0,
  'data' => {
    'ruby_version' => RUBY_VERSION,
    'args_received' => args
  }
}

# Print result as JSON
puts JSON.generate(output)
exit 0