package main

import (
	"fmt"
	"io/ioutil"

	json "github.com/layeh/gopher-json"
	"github.com/screwdriver-cd/meta-cli/internal/fetch"
	lua "github.com/yuin/gopher-lua"
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
	luaMetaSpecTypeName                  = "MetaSpec"
	luaLastSuccessfulMetaRequestTypeName = "LastSuccessfulMetaRequest"
)

// metaSpecMetaFilePath calls MetaFilePath()
func metaSpecMetaFilePath(L *lua.LState) int {
	meta := checkMetaSpec(L, 1)
	if L.GetTop() != 1 {
		L.RaiseError("Require 0 args, but %d were passed", L.GetTop()-1)
		return 0
	}
	metaFilePath := meta.MetaFilePath()
	L.Push(lua.LString(metaFilePath))
	return 1
}

// metaSpecGet(key) returns json.decode(meta.Get(key))
func metaSpecGet(L *lua.LState) int {
	meta := checkMetaSpec(L, 1)
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
	meta := checkMetaSpec(L, 1)
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
	meta := checkMetaSpec(L, 1)
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

	// Add a metatable with spec referring to the spec, which dumped this
	mt := L.CreateTable(0, 1)
	mt.RawSetString("spec", L.Get(1))
	L.SetMetatable(decoded, mt)

	L.Push(decoded)
	return 1
}

// metaSpecUndump(o) writes json.encode(o) to the meta.MetaFilePath()
func metaSpecUndump(L *lua.LState) int {
	meta := checkMetaSpec(L, 1)
	if L.GetTop() != 2 {
		L.RaiseError("Require 1 args, but %d were passed", L.GetTop()-1)
		return 0
	}
	decoded := L.CheckAny(2)

	// Check that the argument passed for undumping was created with this meta instance.
	if !L.Equal(L.Get(1), L.GetMetaField(decoded, "spec")) {
		L.ArgError(2, "object passed to undump must have been dumped by same spec")
		return 0
	}

	data, err := json.Encode(decoded)
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

// metaSpecClone(spec) clones the spec
func metaSpecClone(L *lua.LState) int {
	if L.GetTop() != 1 {
		L.RaiseError("Require 0 args, but %d were passed", L.GetTop()-1)
		return 0
	}

	oldSpec := checkMetaSpec(L, 1)
	newSpec := oldSpec.CloneDefaultMeta()

	L.Push(metaSpecToLua(L, newSpec))
	return 1
}

// registerLastSuccessfulMetaRequest registers LastSuccessfulMetaRequest type
func registerLastSuccessfulMetaRequest(L *lua.LState) *lua.LTable {
	// Create a new type metatable and add to global namespace
	mt := L.NewTypeMetatable(luaLastSuccessfulMetaRequestTypeName)
	L.SetGlobal(luaLastSuccessfulMetaRequestTypeName, mt)

	// methods - will return these from __index func in default case; set this way because SetFuncs wraps raw funcs.
	funcs := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"clone": lastSuccessfulMetaRequestClone,
	})

	// Get fields
	L.SetField(mt, "__index", L.NewFunction(func(L *lua.LState) int {
		lastSuccessfulMetaRequest := checkLastSuccessfulMetaRequest(L, 1)
		k := L.CheckString(2)
		switch k {
		case "SdToken":
			L.Push(lua.LString(lastSuccessfulMetaRequest.SdToken))
		case "SdAPIURL":
			L.Push(lua.LString(lastSuccessfulMetaRequest.SdAPIURL))
		case "DefaultSdPipelineID":
			L.Push(lua.LNumber(lastSuccessfulMetaRequest.DefaultSdPipelineID))
		default:
			// If unknown field, delegate to the function map.
			L.Push(L.GetField(funcs, k))
		}
		return 1
	}))

	// Set fields
	L.SetField(mt, "__newindex", L.NewFunction(func(L *lua.LState) int {
		lastSuccessfulMetaRequest := checkLastSuccessfulMetaRequest(L, 1)
		k := L.CheckString(2)
		switch k {
		case "SdToken":
			lastSuccessfulMetaRequest.SdToken = L.CheckString(3)
		case "SdAPIURL":
			lastSuccessfulMetaRequest.SdAPIURL = L.CheckString(3)
		case "DefaultSdPipelineID":
			lastSuccessfulMetaRequest.DefaultSdPipelineID = L.CheckInt64(3)
		default:
			return 0
		}
		return 0
	}))

	// Return the metadata
	return mt
}

// registerMetaSpecType registers the MetaSpec type and adds methods via __index meta method
func registerMetaSpecType(L *lua.LState) *lua.LTable {
	mt := L.NewTypeMetatable(luaMetaSpecTypeName)
	L.SetGlobal(luaMetaSpecTypeName, mt)

	// methods - will return these from __index func in default case; set this way because SetFuncs wraps raw funcs.
	funcs := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"get":          metaSpecGet,
		"set":          metaSpecSet,
		"dump":         metaSpecDump,
		"undump":       metaSpecUndump,
		"clone":        metaSpecClone,
		"metaFilePath": metaSpecMetaFilePath,
	})

	// Get fields
	L.SetField(mt, "__index", L.NewFunction(func(state *lua.LState) int {
		metaSpec := checkMetaSpec(L, 1)
		k := L.CheckString(2)
		switch k {
		case "MetaSpace":
			L.Push(lua.LString(metaSpec.MetaSpace))
		case "SkipFetchNonexistentExternal":
			L.Push(lua.LBool(metaSpec.SkipFetchNonexistentExternal))
		case "MetaFile":
			L.Push(lua.LString(metaSpec.MetaFile))
		case "JSONValue":
			L.Push(lua.LBool(metaSpec.JSONValue))
		case "SkipStoreExternal":
			L.Push(lua.LBool(metaSpec.SkipStoreExternal))
		case "LastSuccessfulMetaRequest":
			L.Push(lastSuccessfulMetaRequestToLua(L, &metaSpec.LastSuccessfulMetaRequest))
		case "CacheLocal":
			L.Push(lua.LBool(metaSpec.CacheLocal))
		default:
			// If unknown field, delegate to the function map.
			L.Push(L.GetField(funcs, k))
		}
		return 1
	}))

	// Set fields
	L.SetField(mt, "__newindex", L.NewFunction(func(state *lua.LState) int {
		metaSpec := checkMetaSpec(L, 1)
		k := L.CheckString(2)
		switch k {
		case "MetaSpace":
			metaSpec.MetaSpace = L.CheckString(3)
		case "SkipFetchNonexistentExternal":
			metaSpec.SkipFetchNonexistentExternal = L.CheckBool(3)
		case "MetaFile":
			metaSpec.MetaFile = L.CheckString(3)
		case "JSONValue":
			L.ArgError(3, "JSONValue cannot be set")
			return 0
		case "SkipStoreExternal":
			metaSpec.SkipStoreExternal = L.CheckBool(3)
		case "LastSuccessfulMetaRequest":
			metaSpec.LastSuccessfulMetaRequest = *checkLastSuccessfulMetaRequest(L, 3)
		case "CacheLocal":
			metaSpec.CacheLocal = L.CheckBool(3)
		}
		return 0
	}))

	// Return the metadata
	return mt
}

// metaSpecToLua converts the spec to a lua.LUserData by attaching it to the Value and setting its metatable.
func metaSpecToLua(L *lua.LState, spec *MetaSpec) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = spec
	L.SetMetatable(ud, L.GetTypeMetatable(luaMetaSpecTypeName))
	return ud
}

// lastSuccessfulMetaRequestToLua converts request to lua.LUserData, attaching it to the Value and setting metatable.
func lastSuccessfulMetaRequestToLua(L *lua.LState, request *fetch.LastSuccessfulMetaRequest) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = request
	L.SetMetatable(ud, L.GetTypeMetatable(luaLastSuccessfulMetaRequestTypeName))
	return ud
}

// lastSuccessfulMetaRequestClone(request) clones the request
func lastSuccessfulMetaRequestClone(L *lua.LState) int {
	if L.GetTop() != 1 {
		L.RaiseError("Require 0 arg, but %d were passed", L.GetTop()-1)
		return 0
	}

	oldRequest := checkLastSuccessfulMetaRequest(L, 1)
	newRequest := *oldRequest

	L.Push(lastSuccessfulMetaRequestToLua(L, &newRequest))
	return 1
}

// checkMetaSpec like lua.LState.Check methods, this ensures the args is UserData and casts to *MetaSpec then returns it.
func checkMetaSpec(L *lua.LState, n int) *MetaSpec {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(*MetaSpec); ok {
		return v
	}
	L.ArgError(n, "MetaSpec expected")
	return nil
}

// checkLastSuccessfulMetaRequest like lua.LState.Check methods, this ensures the args is UserData and casts to
// *fetch.LastSuccessfulMetaRequest then returns it.
func checkLastSuccessfulMetaRequest(L *lua.LState, n int) *fetch.LastSuccessfulMetaRequest {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(*fetch.LastSuccessfulMetaRequest); ok {
		return v
	}
	L.ArgError(n, "LastSuccessfulMetaRequest expected")
	return nil
}

// callMethod calls methodName on the ud object, pushing the function, ud, and args on the stack for the new call
func callMethod(L *lua.LState, ud *lua.LUserData, methodName string, nret int) int {
	top := L.GetTop()
	f := L.GetField(ud, methodName)
	L.Push(f)
	L.Push(ud)
	for i := 0; i < top; i++ {
		L.Push(L.Get(-top - 2))
	}
	L.Call(top+1, nret)
	return nret
}

// callMethodLGFunction returns a lua.LGFunction that calls methodName on the ud object (with ud in first arg)
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
	registerLastSuccessfulMetaRequest(L)

	// Create a lua object for our MetaSpec
	ud := metaSpecToLua(L, l.MetaSpec)

	// Register methods on global "meta" that call the lua MetaData object
	meta := L.RegisterModule("meta", map[string]lua.LGFunction{
		"get":          callMethodLGFunction(ud, "get", 1),
		"set":          callMethodLGFunction(ud, "set", 0),
		"dump":         callMethodLGFunction(ud, "dump", 1),
		"undump":       callMethodLGFunction(ud, "undump", 0),
		"clone":        callMethodLGFunction(ud, "clone", 1),
		"metaFilePath": callMethodLGFunction(ud, "metaFilePath", 1),
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
