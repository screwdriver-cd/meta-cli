package main

import (
	"bufio"
	"github.com/stretchr/testify/suite"
	lua "github.com/yuin/gopher-lua"
	"os"
	"os/exec"
	"strings"
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
		require.NoError(L.DoFile("testdata/test.lua"))
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
	s.Assert().NoError(s.LuaSpec.Do("testdata/test-arg-passing.lua", "foo", "bar", "baz"))
}

func (s *LuaSuite) TestArg_Passing_Json() {
	s.Assert().NoError(s.LuaSpec.Do("testdata/test-arg-passing-json.lua", `{"foo": "bar", "bar": [1, 2, 3.45]}`))
}

func (s *LuaSuite) TestCLI() {
	type testCase struct {
		name      string
		cliName   string
		cliArgs   []string
		wantErr   bool
		expectErr string
		verifyErr func(s *LuaSuite, tc *testCase, stdout, stderr string)
		verify    func(s *LuaSuite, tc *testCase, lines []string)
	}
	tests := []testCase{
		{
			name:    "test-shebang.lua no args",
			cliName: "testdata/test-shebang.lua",
			verify: func(s *LuaSuite, tc *testCase, lines []string) {
				require := s.Require()
				require.Len(lines, 3)
				assert := s.Assert()
				assert.Equal(lines[0], "hello world")
				assert.Equal(lines[1], tc.cliName)
				assert.JSONEq(lines[2], `[]`)
			},
		},
		{
			name:    "test-shebang.lua --flagarg argvalue",
			cliName: "testdata/test-shebang.lua",
			cliArgs: []string{"--flagarg", "argvalue"},
			verify: func(s *LuaSuite, tc *testCase, lines []string) {
				require := s.Require()
				require.Len(lines, 3)
				assert := s.Assert()
				assert.Equal(lines[0], "hello world")
				assert.Equal(lines[1], tc.cliName)
				assert.JSONEq(lines[2], `["--flagarg", "argvalue"]`)
			},
		},
		{
			name:    "test-shebang-argparse.lua",
			cliName: "testdata/test-shebang-argparse.lua",
			verify: func(s *LuaSuite, tc *testCase, lines []string) {
				require := s.Require()
				require.Len(lines, 1)
				assert := s.Assert()
				assert.JSONEq(lines[0], `
{
	"default": "default",
	"rest": []
}
`)
			},
		},
		{
			name:    "test-shebang-argparse.lua -t",
			cliName: "testdata/test-shebang-argparse.lua",
			cliArgs: []string{"-t"},
			verify: func(s *LuaSuite, tc *testCase, lines []string) {
				require := s.Require()
				require.Len(lines, 1)
				assert := s.Assert()
				assert.JSONEq(lines[0], `
{
	"default": "default",
	"test": true,
	"rest": []
}
`)
			},
		},
		{
			name:    "test-shebang-argparse.lua -c FOO",
			cliName: "testdata/test-shebang-argparse.lua",
			cliArgs: []string{"-c", "FOO"},
			verify: func(s *LuaSuite, tc *testCase, lines []string) {
				require := s.Require()
				require.Len(lines, 1)
				assert := s.Assert()
				assert.JSONEq(lines[0], `
{
	"default": "default",
	"choice": "FOO",
	"rest": []
}
`)
			},
		},
		{
			name:    "test-shebang-argparse.lua -c BAR",
			cliName: "testdata/test-shebang-argparse.lua",
			cliArgs: []string{"-c", "BAR"},
			verify: func(s *LuaSuite, tc *testCase, lines []string) {
				require := s.Require()
				require.Len(lines, 1)
				assert := s.Assert()
				assert.JSONEq(lines[0], `
{
	"default": "default",
	"choice": "BAR",
	"rest": []
}
`)
			},
		},
		{
			name:    "test-shebang-argparse.lua -c BAZ",
			cliName: "testdata/test-shebang-argparse.lua",
			cliArgs: []string{"-c", "BAZ"},
			verify: func(s *LuaSuite, tc *testCase, lines []string) {
				require := s.Require()
				require.Len(lines, 1)
				assert := s.Assert()
				assert.JSONEq(lines[0], `
{
	"default": "default",
	"choice": "BAZ",
	"rest": []
}
`)
			},
		},
		{
			name:      "test-shebang-argparse.lua -c BAD_CHOICE fails",
			cliName:   "testdata/test-shebang-argparse.lua",
			cliArgs:   []string{"-c", "BAD_CHOICE"},
			wantErr:   true,
			expectErr: "exit status 1",
			verifyErr: func(s *LuaSuite, tc *testCase, stdout, stderr string) {
				assert := s.Assert()
				assert.Regexp("^Usage: "+tc.cliName, stderr)
				assert.Regexp("must be one of 'FOO', 'BAR', 'BAZ'", stderr)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()
			defer s.TearDownTest()

			require := s.Require()
			cmd := exec.Command(tt.cliName, tt.cliArgs...)
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			err := cmd.Run()
			if tt.wantErr {
				require.Error(err)
				if tt.expectErr != "" {
					require.EqualError(err, tt.expectErr)
				}
				if tt.verifyErr != nil {
					tt.verifyErr(s, &tt, stdout.String(), stderr.String())
				}
				return
			}
			var lines []string
			scanner := bufio.NewScanner(strings.NewReader(stdout.String()))
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			require.NotNil(tt.verify)
			tt.verify(s, &tt, lines)
		})
	}
}
