package main

import (
	"github.com/stretchr/testify/suite"
	lua "github.com/yuin/gopher-lua"
	"os"
	"testing"
)

type LuaSuite struct {
	suite.Suite
	LuaSpec
}

func (s *LuaSuite) SetupTest() {
	s.LuaSpec = LuaSpec{
		MetaSpec: &MetaSpec{
			MetaSpace: testDir,
			MetaFile:  testFile,
		},
	}

	_, err := s.MetaSpec.SetupDir()
	s.Require().NoError(err)
}

func (s *LuaSuite) TearDownTest() {
	s.Require().NoError(os.RemoveAll(s.MetaSpec.MetaSpace))
}

func TestLuaSuite(t *testing.T) {
	suite.Run(t, new(LuaSuite))
}

func (s *LuaSuite) TestLua() {
	require := s.Require()

	s.LuaSpec.EvaluateFunction = func(L *lua.LState) int {
		// Clear the stack
		L.Pop(L.GetTop())

		// Load the test cases from file
		require.NoError(L.DoFile("test.lua"))
		require.Equal(1, L.GetTop())
		test := L.CheckTable(1)
		L.Pop(1)

		// For each method on the returned test object, invoke it safely with PCall.
		testCount := 0
		test.ForEach(func(key lua.LValue, value lua.LValue) {
			if value.Type() != lua.LTFunction {
				return
			}
			testCount++
			s.Run(lua.LVAsString(key), func() {
				s.SetupTest()
				defer s.TearDownTest()

				require := s.Require()
				L.Push(value)
				L.Push(test)
				require.NoError(L.PCall(1, 0, nil))
			})
		})

		// Ensure we ran non-zero tests.
		s.Assert().NotEqual(0, testCount, "test should not be empty")

		return 0
	}

	// Run it
	require.NoError(s.LuaSpec.Do())
}

func (s *LuaSuite) TestArg_Passing() {
	s.Assert().NoError(s.LuaSpec.Do("test-arg-passing.lua", "foo", "bar", "baz"))
}

func (s *LuaSuite) TestArg_Passing_Json() {
	s.Assert().NoError(s.LuaSpec.Do("test-arg-passing-json.lua", `{"foo": "bar", "bar": [1, 2, 3.45]}`))
}
