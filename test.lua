-- luacheck: globals meta
-- test starting key is empty
local foo, err = meta.get("foo")
assert(err == nil)
assert(foo == nil, foo)

-- test setting key then getting it yields proper result
meta.set("foo", "bar")
foo, err = meta.get("foo")
assert(err == nil)
assert(foo == "bar")

-- test setting a variable to number and then getting and doing arithmetic works
meta.set("num", 123)
local num, err = meta.get("num")
assert(num + 2 == 125)
assert(tonumber(num) == 123)

-- test using number directly - first value is used; error is ignored
assert((meta.get("num") + 2) == 125)

-- test "ternary" for unset
assert(meta.get("yada") == nil)
assert(meta.get("yada") == nil and 0 or tonumber(meta.get("yada")) == 0)

-- test "ternary" for set
assert(meta.get("num") == nil and 0 or tonumber(meta.get("num")) == 123)

-- test inspiratinal use-case
-- [[inc safely]]
function inc(key)
    local value, err = meta.get(key)
    if err ~= nil or value == nil then
        value = 1
    else
        value = tonumber(value) + 1
    end
    return value, meta.set(key, value)
end

assert(meta.get("missing") == nil)
assert(inc("missing") == 1)
assert(inc("missing") == 2)

-- test dump
assert(meta.set("yowza", "abc") == nil)
local d, err = meta.dump()
assert(err == nil)
json = require("json")
print(json.encode(d))
assert(d["yowza"] == "abc", string.format("d.yowza=%s", d["yowza"]))

-- test other types
-- number
assert(meta.set("abc", 123) == nil)
assert(meta.get("abc") == 123)
assert(meta.set("def", 543.21) == nil)
assert(meta.get("def") == 543.21)

-- nested table
myvals = { foo = "bar", yada = { yada = "yada" } }
assert(meta.set("table", myvals) == nil)
assert(meta.get("table.foo") == myvals.foo)
assert(meta.get("table.yada.yada") == myvals.yada.yada)
assert(json.encode(meta.get("table")) == json.encode(myvals))

-- test cloning & setting variables
local m2 = meta.clone()
assert(m2 ~= nil)
assert(m2.MetaFile == "meta")
m2.MetaFile = "meta2"
assert(m2.MetaFile == "meta2")
assert(m2.MetaSpace ~= "")
assert(m2.MetaSpace ~= "/dev/null")
m2.MetaSpace = "/dev/null"
assert(m2.MetaSpace == "/dev/null")

meta.spec.LastSuccessfulMetaRequest.SdToken = 123
assert(meta.spec.LastSuccessfulMetaRequest.SdToken == "123",
    string.format("SdToken=%s", meta.spec.LastSuccessfulMetaRequest.SdToken))
local l2 = meta.spec.LastSuccessfulMetaRequest:clone()
assert(l2.SdToken == "123")
l2.SdToken = 543
assert(l2.SdToken == "543")
assert(meta.spec.LastSuccessfulMetaRequest.SdToken == "123")
meta.spec.LastSuccessfulMetaRequest = l2
assert(meta.spec.LastSuccessfulMetaRequest.SdToken == "543",
    string.format("SdToken=%s", meta.spec.LastSuccessfulMetaRequest.SdToken))

-- test that JSONValue cannot be set
local ran, errorMsg = pcall(function()
    meta.spec.JSONValue = false
end)
assert(not ran)
assert(errorMsg:find("(JSONValue cannot be set)"), string.format("errorMsg=%s", errorMsg))

-- test enforcement of undump matching dump
-- undump of dump is allowed
local ran, errorMessage = pcall(function()
    meta.undump(meta.dump())
end)
assert(ran)
assert(not errorMessage)
-- undump of cloned dump is not allowed
local ran, errorMessage = pcall(function()
    meta.undump(meta.clone():dump())
end)
assert(not ran)
assert(errorMessage:find("(object passed to undump must have been dumped by same spec)"),
    string.format("errorMsg=%s", errorMsg))
-- Ensure that the spec metatable doesn't leak into the dumped object
local dump = meta.dump()
assert(not dump.spec, string.format("dump.spec=%s", dump.spec))
assert(getmetatable(dump).spec, string.format("getmetatable(dump).spec=%s", getmetatable(dump).spec))
assert(meta.spec == getmetatable(dump).spec)

-- test undumping a plain table
local ran, errorMessage = pcall(function()
    meta.undump({ foo = "bar" })
end)
assert(not ran)
assert(errorMessage:find("(object passed to undump must have been dumped by same spec)"),
    string.format("errorMsg=%s", errorMessage))

-- test workaround for undumping
local ran, errorMessage = pcall(function()
    local d = { workaround = "achievement unlocked!" }
    setmetatable(d, { spec = meta.spec })
    meta.spec:undump(d)
end)
assert(ran)
assert(not errorMessage, string.format("errorMsg=%s", errorMessage))
local workaround = meta.get("workaround")
assert(workaround == "achievement unlocked!", string.format("workaround=%s", workaround))

-- test metaFilePath works
assert(meta.metaFilePath())
assert(meta.metaFilePath() == meta.spec:metaFilePath())
