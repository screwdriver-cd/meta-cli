-- luacheck: globals meta
local test = {}

-- test starting key is empty
function test:foo_starts_as_nil()
    local foo = meta.get("foo")
    assert(foo == nil, foo)
end

-- test setting foo to "bar" then getting "foo" returns bar
function test:set_get_string()
    meta.set("foo", "bar")
    foo = meta.get("foo")
    assert(foo == "bar", tostring(foo))
end

-- test setting a variable to number and then getting and doing arithmetic works
function test:get_set_num()
    meta.set("num", 123)
    local num = meta.get("num")
    assert(num + 2 == 125)
    assert(tonumber(num) == 123)
end

-- test "ternary" for unset
function test:ternary_for_unset()
    local yada = meta.get("yada")
    assert(yada == nil, tostring(yada))
    local num = yada == nil and 0 or tonumber(yada)
    assert(num == 0, tostring(num))
end

-- test "ternary" for set
function test:ternary_for_set()
    meta.set("num", 123)
    local num = meta.get("num") == nil and 0 or tonumber(meta.get("num"))
    assert(num == 123, tostring(num))
end

-- test inspiratinal use-case
function test:increment_use_case()
    -- [[inc safely]]
    local function inc(key)
        local ret = (meta.get(key) or 0) + 1
        meta.set(key, ret)
        return ret
    end

    local missing = meta.get("missing")
    assert(missing == nil, tostring(missing))
    missing = inc("missing")
    assert(missing == 1, tostring(missing))
    missing = inc("missing")
    assert(missing == 2, tostring(missing))
end

-- test dump
function test:set_then_dump_yields_all()
    assert(meta.set("yowza", "abc") == nil)
    local d, err = meta.dump()
    assert(err == nil)
    assert(d["yowza"] == "abc", string.format("d.yowza=%s", d["yowza"]))
end

-- test other types
-- number
function test:number()
    meta.set("abc", 123)
    assert(meta.get("abc") == 123, tostring(meta.get("abc")))

    meta.set("def", 543.21)
    assert(meta.get("def") == 543.21, tostring(meta.get("def")))
end

-- nested table
function test:nested_table()
    local json = require('json')
    myvals = { foo = "bar", yada = { yada = "yada" } }
    meta.set("table", myvals)
    assert(meta.get("table.foo") == myvals.foo, tostring(meta.get("table.foo")))
    assert(meta.get("table.yada.yada") == myvals.yada.yada, tostring(meta.get("table.yada.yada")))
    assert(json.encode(meta.get("table")) == json.encode(myvals),
            string.format("%s != %s", json.encode(meta.get("table")), json.encode(myvals)))
end

-- test cloning & setting variables
function test:cloning_meta()
    local m2 = meta.clone()
    assert(m2 ~= nil)
    assert(m2.MetaFile == "meta")
    m2.MetaFile = "meta2"
    assert(m2.MetaFile == "meta2")
    assert(m2.MetaSpace ~= "")
    assert(m2.MetaSpace ~= "/dev/null")
    m2.MetaSpace = "/dev/null"
    assert(m2.MetaSpace == "/dev/null")
end

function test:cloning_LastSuccessfulMetaRequest()
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
end

-- test that JSONValue cannot be set
function test:JSONValue_cannot_be_set()
    local ran, errorMsg = pcall(function()
        meta.spec.JSONValue = false
    end)
    assert(not ran, tostring(ran))
    assert(errorMsg:find("(JSONValue cannot be set)"), tostring(errorMsg))
end

-- test enforcement of undump matching dump
-- undump of dump is allowed
function test:undump_of_dump_allowed()
    local ran, errorMessage = pcall(function()
        meta.undump(meta.dump())
    end)
    assert(ran)
    assert(not errorMessage, tostring(errorMessage))
end

-- undump of cloned dump is not allowed
function test:undump_of_cloned_dump_is_not_allowed()
    local ran, errorMessage = pcall(function()
        meta.undump(meta.clone():dump())
    end)
    assert(not ran, tostring(ran))
    assert(errorMessage:find("(object passed to undump must have been dumped by same spec)"), tostring(errorMsg))
end

-- Ensure that the spec metatable doesn't leak into the dumped object
function test:spec_metatable_doesnt_leak_into_dumped_object()
    local dump = meta.dump()
    assert(not dump.spec, string.format("dump.spec=%s", dump.spec))
    assert(getmetatable(dump).spec, string.format("getmetatable(dump).spec=%s", getmetatable(dump).spec))
    assert(meta.spec == getmetatable(dump).spec)
end

-- test undumping a plain table
function test:undump_plain_table_not_allowed()
    local ran, errorMessage = pcall(function()
        meta.undump({ foo = "bar" })
    end)
    assert(not ran, tostring(ran))
    assert(errorMessage:find("(object passed to undump must have been dumped by same spec)"), tostring(errorMessage))
end

-- test workaround for undumping
function test:undump_workaround()
    local ran, errorMessage = pcall(function()
        local d = { workaround = "achievement unlocked!" }
        setmetatable(d, { spec = meta.spec })
        meta.spec:undump(d)
    end)
    assert(ran)
    assert(not errorMessage, tostring(errorMessage))
    local workaround = meta.get("workaround")
    assert(workaround == "achievement unlocked!", tostring(workaround))
end

-- test metaFilePath works
function test:metaFilePath_returns_non_empty()
    assert(meta.metaFilePath())
    assert(meta.metaFilePath() == meta.spec:metaFilePath(),
            string.format("%s != %s", meta.metaFilePath(), meta.spec:metaFilePath()))
end

return test
