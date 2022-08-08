#!./meta-cli
-- This file gives a demo of using argparse to parse CLI when using meta as a shebang (#62)

local json = require 'json'
local argparse = require 'argparse'

-- Parse CLI
local parser = argparse(arg[0], 'Test argparse')
parser:flag('-t --test', 'Test arg')
parser:option('-d --default', 'Option with default', 'default')
parser:option('-c --choice', 'Option with choice')
      :choices { 'FOO', 'BAR', 'BAZ' }
parser:argument('rest', 'Remaining args'):args '*'
local args = parser:parse()

-- Print parsed args
print(json.encode(args))
