package main

import (
	"fmt"
	"github.com/Shopify/go-lua"
	"github.com/sirupsen/logrus"
	"strings"
)

type (
	Cmd     func(...string) (interface{}, error)
	LuaSpec struct {
		*MetaSpec
		EvaluateFile string

		vm     *lua.State
		cmdMap map[string]Cmd
	}
)

func (l *LuaSpec) getCmd(args ...string) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("get requires exactly one argument; the key; not %d", len(args))
	}
	got, err := l.Get(args[0])
	if err != nil {
		return nil, err
	}
	if got == "null" {
		return nil, nil
	}
	return got, nil
}

func (l *LuaSpec) setCmd(args ...string) (interface{}, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("set requires exactly two argument; the key and value; not %d", len(args))
	}
	return nil, l.Set(args[0], args[1])
}

func (l *LuaSpec) pushList(L *lua.State, res []string) {
	L.CreateTable(len(res), 0)
	for i, s := range res {
		L.PushString(s)
		L.RawSetInt(-2, i+1)
	}
}

func (l *LuaSpec) pushMap(L *lua.State, res map[string]interface{}) {
	L.CreateTable(0, len(res))
	for k, v := range res {
		L.PushString(k)
		switch v := v.(type) {
		case int64:
			L.PushInteger(int(v))
		case string:
			L.PushString(v)
		case map[string]interface{}:
			l.pushMap(L, v)
		}
		L.RawSet(-3)
	}
}

func (l *LuaSpec) execCmdInLuaScript(curCmd string) func(L *lua.State) int {
	return func(L *lua.State) int {
		var args []string
		nargs := L.Top()
		for i := 1; i <= nargs; i++ {
			luaType := L.TypeOf(i)
			switch luaType {
			case lua.TypeNumber:
				fallthrough
			case lua.TypeString:
				if s, ok := lua.ToStringMeta(L, i); ok {
					args = append(args, s)
				}
			case lua.TypeNil:

			default:
				// arg x is one based, like other stuff in lua land
				L.PushFString("The type of arg %d is incorrect, only number and string are acceptable", i)
				L.Error()
			}
		}
		// we have checked the existence of 'curCmd' before
		f, _ := l.cmdMap[curCmd]
		res, err := f(args...)
		if err != nil {
			L.PushNil()
			L.PushString(err.Error())
			return 2
		}
		switch res := res.(type) {
		case bool:
			L.PushBoolean(res)
		case []byte:
			L.PushString(string(res))
		case string:
			L.PushString(res)
		case []string:
			l.pushList(L, res)
		case map[string]interface{}:
			l.pushMap(L, res)
		case int:
			L.PushInteger(res)
		case nil:
			L.PushNil()
		default:
			L.PushFString("The type of result returned(%s) from command '%s' with args %s is unsupported",
				fmt.Sprintf("%T", res), curCmd, fmt.Sprint(args))
			L.Error()
		}
		return 1
	}
}

func (l *LuaSpec) dispatchCmd(L *lua.State) int {
	// ignore the meta table itself (the first arg)
	if s, ok := lua.ToStringMeta(L, 2); ok {
		s = strings.ToLower(s)
		_, ok = l.cmdMap[s]
		if ok {
			L.PushGoFunction(l.execCmdInLuaScript(s))
			return 1
		}
	}
	// it is equal to return nil
	return 0
}

func (l *LuaSpec) injectAPI() error {
	L := l.vm
	L.CreateTable(0, 1)

	L.CreateTable(0, 1)
	L.PushGoFunction(l.dispatchCmd)
	L.SetField(-2, "__index")
	L.SetMetaTable(-2)

	// inject global api namespace
	L.Global("package")
	L.Field(-1, "loaded")
	L.PushValue(-3)
	L.SetField(-2, "meta")
	L.Pop(2)
	L.SetGlobal("meta")

	return nil
}

func (l *LuaSpec) initLua() error {
	//metaSpec := *l.MetaSpec
	//l.MetaSpec = &metaSpec
	//l.MetaSpec.JSONValue = true
	l.vm = lua.NewState()
	l.cmdMap = map[string]Cmd{
		"get": l.getCmd,
		"set": l.setCmd,
	}
	lua.OpenLibraries(l.vm)
	return l.injectAPI()
}

func (l *LuaSpec) Run() error {
	logrus.Debugf("Running lua evaluating '%s'", l.EvaluateFile)
	if err := l.initLua(); err != nil {
		return err
	}
	if l.EvaluateFile == "" {
		return fmt.Errorf("nothing to evaluate")
	}
	return lua.DoFile(l.vm, l.EvaluateFile)
}
