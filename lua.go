package main

import (
	json "github.com/layeh/gopher-json"
	"github.com/yuin/gopher-lua"
)

type (
	LuaSpec struct {
		// Inputs
		*MetaSpec
		EvaluateFile string

		// Valid during run
		L *lua.LState
	}
)

const (
	luaMetaSpecTypeName = "MetaSpec"
)

func metaSpecGet(L *lua.LState) int {
	meta := checkMeta(L)
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

func metaSpecSet(L *lua.LState) int {
	meta := checkMeta(L)
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

func metaSpecDump(L *lua.LState) int {
	meta := checkMeta(L)
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

func registerMetaSpecType(L *lua.LState) *lua.LTable {
	mt := L.NewTypeMetatable(luaMetaSpecTypeName)
	L.SetGlobal(luaMetaSpecTypeName, mt)
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"get":  metaSpecGet,
		"set":  metaSpecSet,
		"dump": metaSpecDump,
	}))
	return mt
}

func metaSpecToLua(L *lua.LState, spec *MetaSpec) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = spec
	L.SetMetatable(ud, L.GetTypeMetatable(luaMetaSpecTypeName))
	return ud
}

func checkMeta(L *lua.LState) *MetaSpec {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*MetaSpec); ok {
		return v
	}
	L.ArgError(1, "MetaSpec expected")
	return nil
}

func callMethod(L *lua.LState, o *lua.LUserData, fname string, nret int) int {
	top := L.GetTop()
	f := L.GetField(o, fname)
	L.Push(f)
	L.Push(o)
	for i := 0; i < top; i++ {
		L.Push(L.Get(- top - 2))
	}
	L.Call(top+1, nret)
	return nret
}

func (l *LuaSpec) init() error {
	json.Preload(l.L)
	registerMetaSpecType(l.L)
	ud := metaSpecToLua(l.L, l.MetaSpec)
	meta := l.L.RegisterModule("meta", map[string]lua.LGFunction{
		"get": func(L *lua.LState) int {
			return callMethod(L, ud, "get", 1)
		},
		"set": func(L *lua.LState) int {
			return callMethod(L, ud, "set", 0)
		},
		"dump": func(L *lua.LState) int {
			return callMethod(L, ud, "dump", 1)
		},
	})
	l.L.SetField(meta, "global", ud)
	return nil
}

func (l *LuaSpec) Run() error {
	l.MetaSpec.JSONValue = true

	L := lua.NewState()
	defer L.Close()

	l.L = L
	if err := l.init(); err != nil {
		return err
	}

	return L.DoFile(l.EvaluateFile)
}
