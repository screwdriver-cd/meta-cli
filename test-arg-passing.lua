-- test arg passing
assert(arg)
assert(arg[0] == "test-arg-passing.lua")
assert(#arg == 3)
assert(arg[1] == "foo", arg[1])
assert(arg[2] == "bar", arg[2])
assert(arg[3] == "baz", arg[3])

-- test setting via args
meta.set("test-arg-passing", { arg[1], arg[2], arg[3] })
returnedArgs = meta.get("test-arg-passing")
assert(#returnedArgs == 3, #returnedArgs)
assert(returnedArgs[1] == "foo")
assert(returnedArgs[2] == "bar")
assert(returnedArgs[3] == "baz")