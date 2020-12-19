package main

import (
	"fmt"
	"github.com/layeh/gopher-json"
	"github.com/yuin/gopher-lua"
	"io/ioutil"
)

type (
	LuaSpec struct {
		// Inputs
		MetaSpec       *MetaSpec
		EvaluateFile   string
		EvaluateString string
	}
)

const (
	luaMetaSpecTypeName = "MetaSpec"
)

// metaSpecGet(key) returns json.decode(meta.Get(key))
func metaSpecGet(L *lua.LState) int {
	meta := checkMetaSpec(L)
	if L.GetTop() != 2 {
		L.RaiseError("Require 1 arg, but %d were passed", L.GetTop()-1)
		return 0
	}
	got, err := meta.Get(L.CheckString(2))
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	jsonResponse, err := json.Decode(L, []byte(got))
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	L.Push(jsonResponse)
	return 1
}

// metaSpecSet(key, value) performs meta.Set(key, json.encode(value))
func metaSpecSet(L *lua.LState) int {
	meta := checkMetaSpec(L)
	if L.GetTop() != 3 {
		L.RaiseError("Require 2 args, but %d were passed", L.GetTop()-1)
		return 0
	}
	value := L.CheckAny(3)
	data, err := json.Encode(value)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	err = meta.Set(L.CheckString(2), string(data))
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	return 0
}

// metaSpecDump returns json.decode(meta.Dump())
func metaSpecDump(L *lua.LState) int {
	meta := checkMetaSpec(L)
	if L.GetTop() != 1 {
		L.RaiseError("Require 0 args, but %d were passed", L.GetTop()-1)
		return 0
	}
	data, err := meta.GetData()
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	decoded, err := json.Decode(L, data)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	L.Push(decoded)
	return 1
}

// metaSpecUndump(o) writes json.encode(o) to the meta.MetaFilePath()
func metaSpecUndump(L *lua.LState) int {
	meta := checkMetaSpec(L)
	if L.GetTop() != 2 {
		L.RaiseError("Require 1 args, but %d were passed", L.GetTop()-1)
		return 0
	}
	data, err := json.Encode(L.CheckAny(2))
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	err = ioutil.WriteFile(meta.MetaFilePath(), data, 0666)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	return 0
}

// metaSpecCloneOverride(o) clones the global spec and overrides the settings
func metaSpecCloneOverride(L *lua.LState) int {
	overrides := lua.LNil
	switch L.GetTop() {
	case 1:
	case 2:
		overrides = L.CheckTable(2)
	default:
		L.RaiseError("Require 0 or 1 args, but %d were passed", L.GetTop()-1)
		return 0
	}

	meta := L.GetGlobal("meta")
	L.Push(L.GetField(meta, "spec"))
	metaSpec := checkMetaSpec(L).CloneDefaultMeta()

	if overrides != lua.LNil {
		if field := L.GetField(overrides, "MetaSpace"); field != lua.LNil {
			metaSpec.MetaSpace = lua.LVAsString(field)
		}
		if field := L.GetField(overrides, "SkipFetchNonexistentExternal"); field != lua.LNil {
			metaSpec.SkipFetchNonexistentExternal = lua.LVAsBool(field)
		}
		if field := L.GetField(overrides, "MetaFile"); field != lua.LNil {
			metaSpec.MetaFile = lua.LVAsString(field)
		}
		// JSONValue must be true
		metaSpec.JSONValue = true
		if field := L.GetField(overrides, "SkipStoreExternal"); field != lua.LNil {
			metaSpec.SkipStoreExternal = lua.LVAsBool(field)
		}
		if field := L.GetField(overrides, "LastSuccessfulMetaRequest"); field != lua.LNil {
			if field = L.GetField(field, "SdToken"); field != lua.LNil {
				metaSpec.LastSuccessfulMetaRequest.SdToken = lua.LVAsString(field)
			}
			if field = L.GetField(field, "SdAPIURL"); field != lua.LNil {
				metaSpec.LastSuccessfulMetaRequest.SdAPIURL = lua.LVAsString(field)
			}
			if field = L.GetField(field, "DefaultSdPipelineID"); field != lua.LNil {
				metaSpec.LastSuccessfulMetaRequest.DefaultSdPipelineID = int64(lua.LVAsNumber(field))
			}
			// Transport can't be set
		}
		if field := L.GetField(overrides, "CacheLocal"); field != lua.LNil {
			metaSpec.CacheLocal = lua.LVAsBool(field)
		}
	}

	L.Push(metaSpecToLua(L, metaSpec))
	return 1
}

// registerMetaSpecType registers the MetaSpec type and adds methods via __index meta method
func registerMetaSpecType(L *lua.LState) *lua.LTable {
	mt := L.NewTypeMetatable(luaMetaSpecTypeName)
	L.SetGlobal(luaMetaSpecTypeName, mt)

	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"get":           metaSpecGet,
		"set":           metaSpecSet,
		"dump":          metaSpecDump,
		"undump":        metaSpecUndump,
		"cloneOverride": metaSpecCloneOverride,
	}))
	return mt
}

// metaSpecToLua converts the spec to a lua.LUserData by attaching it to the Value and setting its metatable.
func metaSpecToLua(L *lua.LState, spec *MetaSpec) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = spec
	L.SetMetatable(ud, L.GetTypeMetatable(luaMetaSpecTypeName))
	return ud
}

// checkMetaSpec like lua.LState.Check methods, this ensures the args is UserData and casts to *MetaSpec then returns it.
func checkMetaSpec(L *lua.LState) *MetaSpec {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*MetaSpec); ok {
		return v
	}
	L.ArgError(1, "MetaSpec expected")
	return nil
}

// callMethod calls methodName on the ud object, pushing the function, ud, and args on the stack for the new call
func callMethod(L *lua.LState, ud *lua.LUserData, methodName string, nret int) int {
	top := L.GetTop()
	f := L.GetField(ud, methodName)
	L.Push(f)
	L.Push(ud)
	for i := 0; i < top; i++ {
		L.Push(L.Get(- top - 2))
	}
	L.Call(top+1, nret)
	return nret
}

// callMethodLGFunction returns a lua.LGFunction that calls methodName on the ud object
func callMethodLGFunction(ud *lua.LUserData, methodName string, nret int) lua.LGFunction {
	return func(L *lua.LState) int {
		return callMethod(L, ud, methodName, nret)
	}
}

// initState initializes the state |L|.
func (l *LuaSpec) initState(L *lua.LState) error {
	// Preload the json library
	json.Preload(L)

	// Register the MetaSpec TypeMetatable
	registerMetaSpecType(L)

	// Create a lua object for our MetaSpec
	ud := metaSpecToLua(L, l.MetaSpec)

	// Register methods on global "meta" that call the lua MetaData object
	meta := L.RegisterModule("meta", map[string]lua.LGFunction{
		"get":           callMethodLGFunction(ud, "get", 1),
		"set":           callMethodLGFunction(ud, "set", 0),
		"dump":          callMethodLGFunction(ud, "dump", 1),
		"undump":        callMethodLGFunction(ud, "undump", 0),
		"cloneOverride": callMethodLGFunction(ud, "cloneOverride", 1),
	})

	// Register our lua MetaSpec as a field "spec". calling meta.get("key") is identical to meta.spec:get("key")
	L.SetField(meta, "spec", ud)

	// No error
	return nil
}

// Do invokes either DoFile if EvaluateFile is set otherwise DoString.
func (l *LuaSpec) Do() error {
	// Every call will use the gopher-json library to serialize between lua and go, ensure JSONValue is on.
	l.MetaSpec.JSONValue = true

	// Create a lua state valid for this function call
	L := lua.NewState()
	defer L.Close()

	// Initialize the state with json package and our methods and globals.
	if err := l.initState(L); err != nil {
		return err
	}

	// Do the right thing
	if l.EvaluateFile != "" {
		return L.DoFile(l.EvaluateFile)
	}
	if l.EvaluateString != "" {
		return L.DoString(l.EvaluateString)
	}

	// Didn't find anything to do; report error
	return fmt.Errorf("one of EvaluateFile or EvaluateString must be non-empty")
}
