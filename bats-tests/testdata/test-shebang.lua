#!./meta-cli
local json = require 'json'

print("hello world")
-- split arg[0] out, and copy remaining args to a pure array so json can encode them
print(arg[0])
print(json.encode({ unpack(arg) }))
