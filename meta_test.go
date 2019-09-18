package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/screwdriver-cd/meta-cli/internal/fetch"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/termie/go-shutil"
)

const (
	testFile                = "meta"
	testDir                 = "./_test"
	testFilePath            = testDir + "/" + testFile + ".json"
	mockDir                 = "./mock"
	externalFile            = "sd@123:component"
	externalFilePath        = testDir + "/" + externalFile + ".json"
	doesNotExistFile        = "woof"
	kMockHttpDir            = "mockHttp"
	kJobsJsonFile           = "jobs.json"
	kLastSuccessfulMetaFile = "lastSuccessfulMeta.json"
)

type MetaSuite struct {
	suite.Suite
	MetaSpec               MetaSpec
	JobsJson               string
	LastSuccessfulMetaJson string
}

func (s *MetaSuite) SetupSuite() {
	data, err := ioutil.ReadFile(filepath.Join(kMockHttpDir, kJobsJsonFile))
	s.Require().NoError(err)
	s.JobsJson = string(data)

	data, err = ioutil.ReadFile(filepath.Join(kMockHttpDir, kLastSuccessfulMetaFile))
	s.Require().NoError(err)
	s.LastSuccessfulMetaJson = string(data)

}

func (s *MetaSuite) SetupTest() {
	s.MetaSpec = MetaSpec{
		MetaSpace: testDir,
		MetaFile:  testFile,
	}
	_, _ = s.MetaSpec.SetupDir()
}

func (s *MetaSuite) TearDownSuite() {
	_ = os.RemoveAll(s.MetaSpec.MetaSpace)
}

func TestMetaSuite(t *testing.T) {
	suite.Run(t, new(MetaSuite))
}

func (s *MetaSuite) CopyMockFile(metaFile string) error {
	metaFile = metaFile + ".json"
	return shutil.CopyFile(filepath.Join(mockDir, metaFile), filepath.Join(testDir, metaFile), false)
}

func (s *MetaSuite) TestSetupDir() {
	var err error
	var data []byte

	_, err = os.Stat(testFilePath)
	Assert := s.Assert()
	Assert.NoError(err, "could not create %s in %s", testFilePath, testDir)

	data, err = ioutil.ReadFile(testFilePath)
	Assert.NoError(err, "could not read %s in %s", testFilePath, testDir)
	Assert.Equal("{}", string(data), "%s does not have an empty JSON object: %v", testFilePath, string(data))
}

func (s *MetaSuite) TestExternalMetaFile() {
	s.MetaSpec.MetaFile = externalFile
	_, _ = s.MetaSpec.SetupDir()
	_ = os.Remove(externalFilePath)

	// Test set (meta file is not meta.json, should fail)
	err := s.MetaSpec.Set("str", "val")
	Require := s.Require()
	Require.Error(err, "error should be occured")

	// Test get
	Require.NoError(s.CopyMockFile(externalFile))

	got, err := s.MetaSpec.Get("str")
	Assert := s.Assert()
	Assert.Equal("meow", got)
}

func (s *MetaSuite) TestGetMetaNoFile() {
	_ = os.RemoveAll(testDir)
	got, err := s.MetaSpec.Get("woof")
	Assert := s.Assert()
	Assert.NoError(err)
	Assert.Equal("null", got)
}

func niceName(key, desc string) string {
	if desc == "" {
		return key
	}
	return key + ":" + desc
}

func (s *MetaSuite) TestGetMeta() {
	s.Require().NoError(s.CopyMockFile(testFile))

	for _, tt := range []struct {
		key      string
		desc     string
		expected string
		wantErr  bool
	}{
		{
			key:      `str`,
			expected: `fuga`,
		},
		{
			key:      `bool`,
			expected: `true`,
		},
		{
			key:      `int`,
			expected: `1234567`,
		},
		{
			key:      `float`,
			expected: `1.5`,
		},
		{
			key:      `foo.bar-baz`,
			expected: `dashed-key`,
		},
		{
			key:      `obj`,
			expected: `{"ccc":"ddd","momo":{"toke":"toke"}}`,
		},
		{
			key:      `obj.ccc`,
			expected: `ddd`,
		},
		{
			key:      `obj.momo`,
			expected: `{"toke":"toke"}`,
		},
		{
			key:      `ary`,
			expected: `["aaa","bbb",{"ccc":{"ddd":[1234567,2,3]}}]`,
		},
		{
			key:      `ary[0]`,
			expected: `aaa`,
		},
		{
			key:      `ary[2]`,
			expected: `{"ccc":{"ddd":[1234567,2,3]}}`,
		},
		{
			key:      `ary[2].ccc`,
			expected: `{"ddd":[1234567,2,3]}`,
		},
		{
			key:      `ary[2].ccc.ddd`,
			expected: `[1234567,2,3]`,
		},
		{
			key:      `ary[2].ccc.ddd[1]`,
			expected: `2`,
		},
		{
			key:      `nu`,
			expected: `null`,
		},
		{
			key:      `nu`,
			expected: `null`,
		},
		{
			key:      `notexist`,
			desc:     `The key does not exist in meta.json`,
			expected: `null`,
		},
		{
			key:      `ary[]`,
			desc:     `It makes golang zero-value`,
			expected: `aaa`,
		},
		{
			key:      `ary.aaa.bbb.ccc.ddd[10]`,
			desc:     `The key does not exist in meta.json`,
			expected: `null`,
		},
	} {
		s.Run(niceName(tt.key, tt.desc), func() {
			got, err := s.MetaSpec.Get(tt.key)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Assert().Equal(tt.expected, got)
		})
	}
}

func (s *MetaSuite) TestGetMeta_json_object() {
	s.MetaSpec.JsonValue = true
	s.Require().NoError(s.CopyMockFile(testFile))

	for _, tc := range []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "str",
			key:      "str",
			expected: `"fuga"`,
		},
		{
			name:     "foo",
			key:      "foo",
			expected: `{"bar-baz":"dashed-key"}`,
		},
	} {
		s.Run(tc.name, func() {
			got, err := s.MetaSpec.Get(tc.key)
			s.Require().NoError(err)
			s.Assert().Equal(tc.expected, got)
		})
	}
}

func (s *MetaSuite) TestSetMeta() {
	type set struct {
		key   string
		value string
	}
	for _, tc := range []struct {
		name     string
		sets     []set
		expected string
	}{
		{
			name:     "bool",
			sets:     []set{{"bool", "true"}},
			expected: `{"bool":true}`,
		},
		{
			name: "number",
			sets: []set{
				{"int", "10"},
				{"float", "15.5"},
			},
			expected: `{"float":15.5,"int":10}`,
		},
		{
			name:     "string",
			sets:     []set{{"str", "val"}},
			expected: `{"str":"val"}`,
		},
		{
			name:     "flexibleKey",
			sets:     []set{{"foo-bar", "val"}},
			expected: `{"foo-bar":"val"}`,
		},
		{
			name:     "array",
			sets:     []set{{"array[]", "arg"}},
			expected: `{"array":["arg"]}`,
		},
		{
			name: "array_with_index_to_string",
			sets: []set{
				{"array[1]", "arg"},
				{"array", "str"},
			},
			expected: `{"array":"str"}`,
		},
		{
			name: "object_to_string",
			sets: []set{
				{"foo.bar", "baz"},
				{"foo", "baz"},
			},
			expected: `{"foo":"baz"}`,
		},
	} {
		s.Run(tc.name, func() {
			_ = os.RemoveAll(testDir)
			Require := s.Require()
			for _, set := range tc.sets {
				Require.NoError(s.MetaSpec.Set(set.key, set.value))
			}
			out, err := ioutil.ReadFile(testFilePath)
			Require.NoError(err, "Meta file did not create")
			s.Assert().Equal(tc.expected, string(out))
		})
	}
}
func (s *MetaSuite) TestSetMeta_sequential() {
	type set struct {
		key      string
		value    string
		expected string
	}
	for _, tc := range []struct {
		name string
		sets []set
	}{
		{
			name: "array_with_index",
			sets: []set{
				{
					key:      "array[1]",
					value:    "arg",
					expected: `{"array":[null,"arg"]}`,
				},
				{
					key:      "array[2]",
					value:    "argarg",
					expected: `{"array":[null,"arg","argarg"]}`,
				},
			},
		},
		{
			name: "object",
			sets: []set{
				{
					key:      "foo.bar",
					value:    "baz",
					expected: `{"foo":{"bar":"baz"}}`,
				},
				{
					key:      "foo.barbar",
					value:    "bazbaz",
					expected: `{"foo":{"bar":"baz","barbar":"bazbaz"}}`,
				},
				{
					key:      "foo.bar.baz",
					value:    "piyo",
					expected: `{"foo":{"bar":{"baz":"piyo"},"barbar":"bazbaz"}}`,
				},
				{
					key:      "foo.bar-baz",
					value:    "dashed-key",
					expected: `{"foo":{"bar":{"baz":"piyo"},"bar-baz":"dashed-key","barbar":"bazbaz"}}`,
				},
			},
		},
		{
			name: "array_with_object",
			sets: []set{
				{
					key:      "foo[1].bar",
					value:    "baz",
					expected: `{"foo":[null,{"bar":"baz"}]}`,
				},
				{
					key:      "foo.bar[1]",
					value:    "baz",
					expected: `{"foo":{"bar":[null,"baz"]}}`,
				},
				{
					key:      "foo[1].bar[1]",
					value:    "baz",
					expected: `{"foo":[null,{"bar":[null,"baz"]}]}`,
				},
				{
					key:      "foo[0].bar[1]",
					value:    "baz",
					expected: `{"foo":[{"bar":[null,"baz"]},{"bar":[null,"baz"]}]}`,
				},
				{
					key:      "foo[1].bar[0]",
					value:    "ba",
					expected: `{"foo":[{"bar":[null,"baz"]},{"bar":["ba","baz"]}]}`,
				},
				{
					key:      "foo[1].bar[2]",
					value:    "bazbaz",
					expected: `{"foo":[{"bar":[null,"baz"]},{"bar":["ba","baz","bazbaz"]}]}`,
				},
				{
					key:      "foo[1].bar[3].baz[1]",
					value:    "qux",
					expected: `{"foo":[{"bar":[null,"baz"]},{"bar":["ba","baz","bazbaz",{"baz":[null,"qux"]}]}]}`,
				},
				{
					key:      "foo[1].bar[3].baz[0]",
					value:    "quxqux",
					expected: `{"foo":[{"bar":[null,"baz"]},{"bar":["ba","baz","bazbaz",{"baz":["quxqux","qux"]}]}]}`,
				},
				{
					key:      "foo[0].bar[3].baz[1]",
					value:    "qux",
					expected: `{"foo":[{"bar":[null,"baz",null,{"baz":[null,"qux"]}]},{"bar":["ba","baz","bazbaz",{"baz":["quxqux","qux"]}]}]}`,
				},
			},
		},
		{
			name: "object_with_array",
			sets: []set{
				{
					key:      "foo.bar[1]",
					value:    "baz",
					expected: `{"foo":{"bar":[null,"baz"]}}`,
				},
				{
					key:      "foo.bar[0]",
					value:    "baz0",
					expected: `{"foo":{"bar":["baz0","baz"]}}`,
				},
				{
					key:      "foo.barbar[2]",
					value:    "bazbaz",
					expected: `{"foo":{"bar":["baz0","baz"],"barbar":[null,null,"bazbaz"]}}`,
				},
			},
		},
	} {
		s.Run(tc.name, func() {
			_ = os.RemoveAll(testDir)
			for _, set := range tc.sets {
				s.Run(set.key, func() {
					Require := s.Require()
					Require.NoError(s.MetaSpec.Set(set.key, set.value))
					out, err := ioutil.ReadFile(testFilePath)
					Require.NoError(err, "Meta file did not create")
					s.Assert().Equal(set.expected, string(out))
				})
			}
		})
	}
}

func (s *MetaSuite) TestSetMeta_json_object() {
	s.MetaSpec.JsonValue = true

	for _, tc := range []struct {
		name     string
		key      string
		value    string
		expected string
		wantErr  bool
	}{
		{
			name:     `json object`,
			key:      "key",
			value:    `{"foo":"bar"}`,
			expected: `{"key":{"foo":"bar"}}`,
		},
		{
			name:     `json array`,
			key:      "key",
			value:    `[1, 2, 3]`,
			expected: `{"key":[1,2,3]}`,
		},
		{
			name:     `json string (quoted)`,
			key:      "key",
			value:    `"foo"`,
			expected: `{"key":"foo"}`,
		},
		{
			name:    `want error with bogus json`,
			key:     "key",
			value:   `"mismatched": "json"}`,
			wantErr: true,
		},
	} {
		s.Run(tc.name, func() {
			defer func() {
				r := recover()
				require.Equal(s.T(), tc.wantErr, r != nil, "wantErr %v; err %v", tc.wantErr, r)
			}()
			_, err := s.MetaSpec.SetupDir()
			Require := s.Require()
			Require.NoError(err)
			Require.NoError(os.Remove(testFilePath))
			Require.NoError(s.MetaSpec.Set(tc.key, tc.value))
			out, err := ioutil.ReadFile(testFilePath)
			Require.NoError(err)
			s.Assert().Equal(tc.expected, string(out))
		})
	}
}

func (s *MetaSuite) TestValidateMetaKeyWithAccept() {
	for _, tc := range []struct {
		key string
	}{
		{`foo`},
		{`f-o-o`},
		{`foo[]`},
		{`f-o-o[]`},
		{`foo[0]`},
		{`foo[1]`},
		{`foo[10]`},
		{`a[10][20]`},
		{`a-b[10][20]`},
		{`foo.bar`},
		{`foo[].bar`},
		{`foo[1].bar`},
		{`foo.bar[]`},
		{`foo.bar[1]`},
		{`foo[].bar[]`},
		{`foo[1].bar[1]`},
		{`foo.bar.baz`},
		{`foo.bar-baz`},
		{`f-o-o.bar--baz`},
		{`foo[1].bar[2].baz[3]`},
		{`foo[1].bar-baz[2]`},
		{`f-o-o[1].bar--baz[2]`},
		{`1.2.3`},
	} {
		s.Run(tc.key, func() {
			Require := s.Require()
			Require.True(validateMetaKey(tc.key), "'%v' is should be accepted", tc.key)
		})
	}
}

func (s *MetaSuite) TestValidateMetaKeyWithReject() {
	for _, tc := range []struct {
		key string
	}{
		{`foo[[`},
		{`foo[]]`},
		{`foo[[]]`},
		{`foo[1e]`},
		{`foo[01]`},
		{`foo.`},
		{`foo.[]`},
		{`-foo`},
		{`foo-[]`},
		{`foo.-bar`},
		{`foo.bar-[]`},
	} {
		s.Run(tc.key, func() {
			Require := s.Require()
			Require.False(validateMetaKey(tc.key), "'%v' is should be rejected", tc.key)
		})
	}
}

func (s *MetaSuite) TestIndexOfFirstRightBracket() {
	for _, tc := range []struct {
		key      string
		expected int
	}{
		{
			key:      "foo[1]",
			expected: 5,
		},
		{
			key:      "foo[10]",
			expected: 6,
		},
	} {
		s.Run(tc.key, func() {
			s.Assert().Equal(tc.expected, indexOfFirstRightBracket(tc.key))
		})
	}
}

func (s *MetaSuite) TestMetaIndexFromKey() {
	for _, tc := range []struct {
		key      string
		expected int
	}{
		{
			key:      "foo[1]",
			expected: 1,
		},
		{
			key:      "foo[10]",
			expected: 10,
		},
		{
			key:      "foo[10].bar[4].baz",
			expected: 10,
		},
		{
			key:      "foo[]",
			expected: 0,
		},
	} {
		s.Run(tc.key, func() {
			s.Assert().Equal(tc.expected, metaIndexFromKey(tc.key))
		})
	}
}

func (s *MetaSuite) TestSymmetry_json_object() {
	nonJsonMetaSpec := s.MetaSpec
	nonJsonMetaSpec.JsonValue = false
	s.MetaSpec.JsonValue = true

	for _, tc := range []struct {
		name                   string
		key                    string
		expectJsonEqualNonJson bool
	}{
		{
			name:                   `string value`,
			key:                    "str",
			expectJsonEqualNonJson: false,
		},
		{
			name:                   `object value`,
			key:                    "foo",
			expectJsonEqualNonJson: true,
		},
		{
			name:                   `array value`,
			key:                    "ary",
			expectJsonEqualNonJson: true,
		},
	} {
		s.Run(tc.name, func() {
			Require := s.Require()
			Require.NoError(s.CopyMockFile(testFile))

			// Get the non-json value from mock
			nonJsonValue, err := nonJsonMetaSpec.Get(tc.key)
			Require.NoError(err)

			// Get the json value from mock
			jsonValue, err := s.MetaSpec.Get(tc.key)
			Require.NoError(err)

			Assert := s.Assert()
			// Compare starting condition
			if tc.expectJsonEqualNonJson {
				Assert.Equal(jsonValue, nonJsonValue)
			} else {
				Assert.NotEqual(jsonValue, nonJsonValue)
			}

			// Reset the output for writing
			_, err = s.MetaSpec.SetupDir()
			Require.NoError(err)
			Require.NoError(os.Remove(testFilePath))

			// Set and get the jsonValue to/from writable file with jsonValue true
			Require.NoError(s.MetaSpec.Set(tc.key, jsonValue))
			newJsonValue, err := s.MetaSpec.Get(tc.key)
			Assert.Equal(jsonValue, newJsonValue)
		})
	}
}

func (s *MetaSuite) TestMetaSpec_IsExternal() {
	for _, tt := range []struct {
		name     string
		metaSpec MetaSpec
		want     bool
	}{
		{
			name: "meta",
			metaSpec: MetaSpec{
				MetaFile: "meta",
			},
			want: false,
		},
		{
			name: "sd@123:component",
			metaSpec: MetaSpec{
				MetaFile: "sd@123:component",
			},
			want: true,
		},
		{
			name: "component",
			metaSpec: MetaSpec{
				MetaFile: "component",
			},
			want: true,
		},
	} {
		s.Run(tt.name, func() {
			got := tt.metaSpec.IsExternal()
			s.Assert().Equal(tt.want, got)
		})
	}
}

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

func (s *MetaSuite) TestMetaSpec_GetExternalData() {
	for _, tt := range []struct {
		name     string
		external string
		expected string
		wantErr  bool
	}{
		{
			name:     "sd@1016708:job1",
			external: "sd@1016708:job1",
			expected: s.LastSuccessfulMetaJson,
		},
		{
			name:     "sd@123:missing",
			external: "sd@123:missing",
			wantErr:  true,
		},
	} {
		s.Run(tt.name, func() {
			var mockHandler MockHandler
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/pipelines/1016708/jobs"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), s.JobsJson)
				})
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/jobs/392525/lastSuccessfulMeta"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), s.LastSuccessfulMetaJson)
				})
			testServer := httptest.NewServer(&mockHandler)
			defer testServer.Close()

			tempDir, err := ioutil.TempDir("", "test")
			Require := s.Require()
			Require.NoError(err)
			defer func() { _ = os.RemoveAll(tempDir) }()

			metaSpec := MetaSpec{
				MetaFile:  tt.external,
				MetaSpace: tempDir,
				LastSuccessfulMetaRequest: fetch.LastSuccessfulMetaRequest{
					SdApiUrl:  testServer.URL + "/v4/",
					Transport: testServer.Client().Transport,
				},
			}

			got, err := metaSpec.GetExternalData()
			if tt.wantErr {
				Require.Error(err)
				return
			}
			Require.NoError(err)
			s.Assert().Equal(tt.expected, string(got))
			mockHandler.AssertExpectations(s.T())
		})
	}
}
