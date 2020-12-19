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
	testFile               = "meta"
	testDir                = "./_test"
	testFilePath           = testDir + "/" + testFile + ".json"
	mockDir                = "./mock"
	externalFile           = "sd@123:component"
	externalFilePath       = testDir + "/" + externalFile + ".json"
	externalFile2          = "sd@123:has-sd"
	externalFile2Path      = testDir + "/" + externalFile2 + ".json"
	doesNotExistFile       = "woof"
	mockHTTPDir            = "mockHttp"
	jobsJSONFile           = "jobs.json"
	lastSuccessfulMetaFile = "lastSuccessfulMeta.json"
)

type MetaSuite struct {
	suite.Suite
	MetaSpec               MetaSpec
	JobsJSON               string
	LastSuccessfulMetaJSON string
}

func (s *MetaSuite) SetupSuite() {
	data, err := ioutil.ReadFile(filepath.Join(mockHTTPDir, jobsJSONFile))
	s.Require().NoError(err)
	s.JobsJSON = string(data)

	data, err = ioutil.ReadFile(filepath.Join(mockHTTPDir, lastSuccessfulMetaFile))
	s.Require().NoError(err)
	s.LastSuccessfulMetaJSON = string(data)

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
	Require.NoError(err)
	Assert := s.Assert()
	Assert.Equal("meow", got)
}

func (s *MetaSuite) TestExternalMetaFileDeletesSd() {
	s.MetaSpec.MetaFile = externalFile2
	s.Require().NoError(s.CopyMockFile(externalFile2))

	// Test set (meta file is not meta.json, should fail)
	got, err := s.MetaSpec.Get("sd")
	s.Require().NoError(err)
	s.Assert().Equal("null", got)
}

func (s *MetaSuite) TestExternalMetaDoesntCreateFile() {
	var mockHandler MockHandler
	mockHandler.On("ServeHTTP", mock.Anything, mock.Anything).
		Once().
		Run(func(args mock.Arguments) {
			_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), s.JobsJSON)
		})
	mockHandler.On("ServeHTTP", mock.Anything, mock.Anything).
		Once().
		Run(func(args mock.Arguments) {
			_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), `{"foo":"bar"}`)
		})
	testServer := httptest.NewServer(&mockHandler)
	defer testServer.Close()

	s.MetaSpec.MetaFile = `sd@1016708:job1`
	s.MetaSpec.LastSuccessfulMetaRequest.Transport = testServer.Client().Transport
	s.MetaSpec.LastSuccessfulMetaRequest.SdAPIURL = testServer.URL + "/v4/"

	// Test set (meta file is not meta.json, should fail)
	got, err := s.MetaSpec.Get("sd")
	s.Require().NoError(err)
	s.Assert().Equal("null", got)
	_, err = os.Stat(s.MetaSpec.MetaFilePath())
	s.Assert().True(os.IsNotExist(err), "%s should not exist", s.MetaSpec.MetaFilePath())
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

	tests := []struct {
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
	}

	for _, tt := range tests {
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
	s.MetaSpec.JSONValue = true
	s.Require().NoError(s.CopyMockFile(testFile))

	tests := []struct {
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
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := s.MetaSpec.Get(tt.key)
			s.Require().NoError(err)
			s.Assert().Equal(tt.expected, got)
		})
	}
}

func (s *MetaSuite) TestSetMeta() {
	type set struct {
		key   string
		value string
	}

	tests := []struct {
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
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_ = os.RemoveAll(testDir)
			Require := s.Require()
			for _, set := range tt.sets {
				Require.NoError(s.MetaSpec.Set(set.key, set.value))
			}
			out, err := ioutil.ReadFile(testFilePath)
			Require.NoError(err, "Meta file did not create")
			s.Assert().Equal(tt.expected, string(out))
		})
	}
}
func (s *MetaSuite) TestSetMeta_sequential() {
	type set struct {
		key      string
		value    string
		expected string
	}

	tests := []struct {
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
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			_ = os.RemoveAll(testDir)
			for _, set := range tt.sets {
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
	s.MetaSpec.JSONValue = true

	tests := []struct {
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
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			defer func() {
				r := recover()
				require.Equal(s.T(), tt.wantErr, r != nil, "wantErr %v; err %v", tt.wantErr, r)
			}()
			_, err := s.MetaSpec.SetupDir()
			Require := s.Require()
			Require.NoError(err)
			Require.NoError(os.Remove(testFilePath))
			Require.NoError(s.MetaSpec.Set(tt.key, tt.value))
			out, err := ioutil.ReadFile(testFilePath)
			Require.NoError(err)
			s.Assert().Equal(tt.expected, string(out))
		})
	}
}

func (s *MetaSuite) TestValidateMetaKeyWithAccept() {
	tests := []struct {
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
	}

	for _, tt := range tests {
		s.Run(tt.key, func() {
			Require := s.Require()
			Require.True(validateMetaKey(tt.key), "'%v' is should be accepted", tt.key)
		})
	}
}

func (s *MetaSuite) TestValidateMetaKeyWithReject() {
	tests := []struct {
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
	}

	for _, tt := range tests {
		s.Run(tt.key, func() {
			Require := s.Require()
			Require.False(validateMetaKey(tt.key), "'%v' is should be rejected", tt.key)
		})
	}
}

func (s *MetaSuite) TestIndexOfFirstRightBracket() {
	tests := []struct {
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
	}

	for _, tt := range tests {
		s.Run(tt.key, func() {
			s.Assert().Equal(tt.expected, indexOfFirstRightBracket(tt.key))
		})
	}
}

func (s *MetaSuite) TestMetaIndexFromKey() {
	tests := []struct {
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
	}

	for _, tt := range tests {
		s.Run(tt.key, func() {
			s.Assert().Equal(tt.expected, metaIndexFromKey(tt.key))
		})
	}
}

func (s *MetaSuite) TestSymmetry_json_object() {
	nonJSONMetaSpec := s.MetaSpec
	nonJSONMetaSpec.JSONValue = false
	s.MetaSpec.JSONValue = true

	tests := []struct {
		name                   string
		key                    string
		expectJSONEqualNonJSON bool
	}{
		{
			name:                   `string value`,
			key:                    "str",
			expectJSONEqualNonJSON: false,
		},
		{
			name:                   `object value`,
			key:                    "foo",
			expectJSONEqualNonJSON: true,
		},
		{
			name:                   `array value`,
			key:                    "ary",
			expectJSONEqualNonJSON: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			Require := s.Require()
			Require.NoError(s.CopyMockFile(testFile))

			// Get the non-json value from mock
			nonJSONValue, err := nonJSONMetaSpec.Get(tt.key)
			Require.NoError(err)

			// Get the json value from mock
			jsonValue, err := s.MetaSpec.Get(tt.key)
			Require.NoError(err)

			Assert := s.Assert()
			// Compare starting condition
			if tt.expectJSONEqualNonJSON {
				Assert.Equal(jsonValue, nonJSONValue)
			} else {
				Assert.NotEqual(jsonValue, nonJSONValue)
			}

			// Reset the output for writing
			_, err = s.MetaSpec.SetupDir()
			Require.NoError(err)
			Require.NoError(os.Remove(testFilePath))

			// Set and get the jsonValue to/from writable file with jsonValue true
			Require.NoError(s.MetaSpec.Set(tt.key, jsonValue))
			newJSONValue, err := s.MetaSpec.Get(tt.key)
			Require.NoError(err)
			Assert.Equal(jsonValue, newJSONValue)
		})
	}
}

func (s *MetaSuite) TestMetaSpec_IsExternal() {
	tests := []struct {
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
	}

	for _, tt := range tests {
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
	tests := []struct {
		name     string
		external string
		expected string
		wantErr  bool
	}{
		{
			name:     "sd@1016708:job1",
			external: "sd@1016708:job1",
			expected: s.LastSuccessfulMetaJSON,
		},
		{
			name:     "sd@123:missing",
			external: "sd@123:missing",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var mockHandler MockHandler
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/pipelines/1016708/jobs"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), s.JobsJSON)
				})
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/jobs/392525/lastSuccessfulMeta"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), s.LastSuccessfulMetaJSON)
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
					SdAPIURL:  testServer.URL + "/v4/",
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

			// Ensure that the caching behavior works too
			_, err = os.Stat(metaSpec.MetaFilePath())
			s.Assert().True(os.IsNotExist(err), "%s should not exist", metaSpec.MetaFilePath())
			defaultMeta := metaSpec.CloneDefaultMeta()
			sdVal, err := defaultMeta.Get("sd")
			s.Assert().NoError(err)
			s.Assert().NotEqual("null", sdVal, "sd should have cached values but was %s", sdVal)
		})
	}
}

func (s *MetaSuite) TestMetaSpec_SkipFetchDoesntSave() {
	metaSpec := MetaSpec{
		MetaFile:                     "sd@1016708:job1",
		MetaSpace:                    testDir,
		SkipFetchNonexistentExternal: true,
	}
	got, err := metaSpec.GetExternalData()
	s.Require().NoError(err)
	s.Assert().Equal("{}", string(got))
	_, err = os.Stat(metaSpec.MetaFilePath())
	s.Assert().Error(err, "File not expected to exist %s", metaSpec.MetaFilePath())
	s.Assert().True(os.IsNotExist(err),
		"File not expected to exist %s; err %v", metaSpec.MetaFilePath(), err)

	defaultMeta := metaSpec.CloneDefaultMeta()
	sdVal, err := defaultMeta.Get("sd")
	s.Require().NoError(err, `Should be able to get missing "sd" key without err`)
	s.Assert().Equal("null", sdVal, "sd should not have any cached values, but had %s", sdVal)
}

func (s *MetaSuite) TestMetaSpec_CachedGet() {
	tests := []struct {
		name     string
		spec     MetaSpec
		key      string
		want     string
		validate func(spec *MetaSpec, got string)
		wantErr  bool
	}{
		{
			name: "str",
			spec: MetaSpec{
				MetaSpace:  testDir,
				MetaFile:   externalFile,
				JSONValue:  false,
				CacheLocal: true,
			},
			key:  "str",
			want: "meow",
		},
		{
			name: "obj.abc",
			spec: MetaSpec{
				MetaSpace:  testDir,
				MetaFile:   externalFile,
				JSONValue:  false,
				CacheLocal: true,
			},
			key:  "obj.abc",
			want: "def",
		},
		{
			name: "obj",
			spec: MetaSpec{
				MetaSpace:  testDir,
				MetaFile:   externalFile,
				JSONValue:  false,
				CacheLocal: true,
			},
			key:  "obj",
			want: `{"abc":"def"}`,
			validate: func(spec *MetaSpec, got string) {
				s2, err := spec.Get("obj.abc")
				s.Require().NoError(err)
				s.Assert().Equal("def", s2)
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			_ = os.RemoveAll(testDir)
			s.SetupTest()
			s.Require().NoError(s.CopyMockFile(externalFile))
			got, err := tt.spec.Get(tt.key)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			// Assert that it equals on the first pass.
			s.Require().NoError(err)
			s.Assert().Equal(tt.want, got)

			// Get local clone and disable cachedGet then assert again.
			localClone := tt.spec.CloneDefaultMeta()
			localClone.CacheLocal = false
			got, err = tt.spec.Get(tt.key)
			s.Require().NoError(err)
			s.Assert().Equal(tt.want, got)

			if tt.validate != nil {
				tt.validate(&tt.spec, got)
			}
		})
	}
}

func (s *MetaSuite) TestMetaSpec_Lua() {
	l := LuaSpec{
		MetaSpec:     &s.MetaSpec,
		EvaluateFile: "test.lua",
	}
	s.Assert().NoError(l.Do())
}
