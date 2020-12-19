package main

import (
	json "github.com/layeh/gopher-json"
	"github.com/yuin/gopher-lua"
)

type (
	GLuaSpec struct {
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
		L.RaiseError("Require 1 arg, but %d were passed", L.GetTop())
		return 0
	}
	got, err := meta.Get(L.CheckString(2))
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	L.Push(lua.LString(got))
	return 1
}

func metaSpecSet(L *lua.LState) int {
	meta := checkMeta(L)
	if L.GetTop() != 3 {
		L.RaiseError("Require 2 args, but %d were passed", L.GetTop())
		return 0
	}
	err := meta.Set(L.CheckString(2), L.CheckString(3))
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	return 0
}

func registerMetaSpecType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaMetaSpecTypeName)
	L.SetGlobal(luaMetaSpecTypeName, mt)
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"get": metaSpecGet,
		"set": metaSpecSet,
	}))
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

func (l *GLuaSpec) init() error {
	json.Preload(l.L)
	registerMetaSpecType(l.L)
	ud := metaSpecToLua(l.L, l.MetaSpec)
	meta := l.L.RegisterModule("meta", map[string]lua.LGFunction{
		"get": func(L *lua.LState) int {
			k := L.Get(1)
			L.Pop(1)
			L.Push(ud)
			L.Push(k)
			return metaSpecGet(L)
		},
		"set": func(L *lua.LState) int {
			k := L.Get(1)
			v := L.Get(2)
			L.Pop(2)
			L.Push(ud)
			L.Push(k)
			L.Push(v)
			return metaSpecSet(L)
		},
	})
	l.L.SetField(meta, "global", ud)
	return nil
}

func (l *GLuaSpec) Run() error {
	L := lua.NewState()
	defer L.Close()

	l.L = L
	if err := l.init(); err != nil {
		return err
	}

	return L.DoFile(l.EvaluateFile)
}
