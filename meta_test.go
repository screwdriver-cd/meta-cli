package main

import (
	"bytes"
	"github.com/screwdriver-cd/meta-cli/internal/fetch"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/termie/go-shutil"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testFile = "meta"
const testDir = "./_test"
const testFilePath = testDir + "/" + testFile + ".json"
const mockDir = "./mock"
const externalFile = "sd@123:component"
const externalFilePath = testDir + "/" + externalFile + ".json"
const doesNotExistFile = "woof"

type MetaSuite struct {
	suite.Suite
	MetaSpec MetaSpec
}

func TestMetaSuite(t *testing.T) {
	suite.Run(t, new(MetaSuite))
}

func (s *MetaSuite) SetupTest() {
	s.MetaSpec.MetaSpace = testDir
	s.MetaSpec.MetaFile = testFile
	_, _ = s.MetaSpec.SetupDir()
}

func (s *MetaSuite) TeardownTest() {
	_ = os.RemoveAll(s.MetaSpec.MetaSpace)
}

func (s *MetaSuite) CopyMockFile(metaFile string) error {
	metaFile = metaFile + ".json"
	return shutil.CopyFile(filepath.Join(mockDir, metaFile), filepath.Join(testDir, metaFile), false)
}

func (s *MetaSuite) TestSetupDir() {
	var err error
	var data []byte

	_ = os.RemoveAll(testDir)
	_, _ = s.MetaSpec.SetupDir()

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
	err := setMeta("str", "val", testDir, externalFile, false)
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
	_, _ = s.MetaSpec.SetupDir()

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
	_ = os.RemoveAll(testDir)
	_, _ = s.MetaSpec.SetupDir()
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
			s.T().Parallel()
			Require := s.Require()
			got, err := s.MetaSpec.Get(tt.key)
			if tt.wantErr {
				Require.Error(err)
				return
			}
			Require.NoError(err)
			Assert := s.Assert()
			Assert.Equal(tt.expected, got)
		})
	}
}

func TestGetMeta_json_object(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
			stdout := new(bytes.Buffer)
			err := getMeta(tc.key, mockDir, testFile, stdout, true, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, stdout.String())
		})
	}
}

func TestSetMeta_bool(t *testing.T) {
	setupDir(testDir, testFile)
	os.Remove(testFilePath)

	setMeta("bool", "true", testDir, testFile, false)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"bool\":true}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_number(t *testing.T) {
	setupDir(testDir, testFile)
	os.Remove(testFilePath)

	setMeta("int", "10", testDir, testFile, false)
	setMeta("float", "15.5", testDir, testFile, false)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"float\":15.5,\"int\":10}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_string(t *testing.T) {
	setupDir(testDir, testFile)
	os.Remove(testFilePath)

	setMeta("str", "val", testDir, testFile, false)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"str\":\"val\"}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_flexibleKey(t *testing.T) {
	setupDir(testDir, testFile)
	os.Remove(testFilePath)

	setMeta("foo-bar", "val", testDir, testFile, false)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"foo-bar\":\"val\"}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_array(t *testing.T) {
	setupDir(testDir, testFile)
	os.Remove(testFilePath)

	setMeta("array[]", "arg", testDir, testFile, false)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"array\":[\"arg\"]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_array_with_index(t *testing.T) {
	setupDir(testDir, testFile)
	os.Remove(testFilePath)

	setMeta("array[1]", "arg", testDir, testFile, false)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"array\":[null,\"arg\"]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("array[2]", "argarg", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"array\":[null,\"arg\",\"argarg\"]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_array_with_index_to_string(t *testing.T) {
	setupDir(testDir, testFile)
	os.Remove(testFilePath)

	setMeta("array[1]", "arg", testDir, testFile, false)
	setMeta("array", "str", testDir, testFile, false)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"array\":\"str\"}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_object(t *testing.T) {
	setupDir(testDir, testFile)
	os.Remove(testFilePath)

	setMeta("foo.bar", "baz", testDir, testFile, false)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"foo\":{\"bar\":\"baz\"}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.barbar", "bazbaz", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":{\"bar\":\"baz\",\"barbar\":\"bazbaz\"}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.bar.baz", "piyo", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":{\"bar\":{\"baz\":\"piyo\"},\"barbar\":\"bazbaz\"}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.bar-baz", "dashed-key", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":{\"bar\":{\"baz\":\"piyo\"},\"bar-baz\":\"dashed-key\",\"barbar\":\"bazbaz\"}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_object_to_string(t *testing.T) {
	setupDir(testDir, testFile)
	os.Remove(testFilePath)

	setMeta("foo.bar", "baz", testDir, testFile, false)
	setMeta("foo", "baz", testDir, testFile, false)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"foo\":\"baz\"}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_array_with_object(t *testing.T) {
	setupDir(testDir, testFile)
	os.Remove(testFilePath)

	setMeta("foo[1].bar", "baz", testDir, testFile, false)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"foo\":[null,{\"bar\":\"baz\"}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.bar[1]", "baz", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":{\"bar\":[null,\"baz\"]}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[1]", "baz", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":[null,{\"bar\":[null,\"baz\"]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[0].bar[1]", "baz", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[null,\"baz\"]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[0]", "ba", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[\"ba\",\"baz\"]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[2]", "bazbaz", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[\"ba\",\"baz\",\"bazbaz\"]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[3].baz[1]", "qux", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[\"ba\",\"baz\",\"bazbaz\",{\"baz\":[null,\"qux\"]}]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[3].baz[0]", "quxqux", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[\"ba\",\"baz\",\"bazbaz\",{\"baz\":[\"quxqux\",\"qux\"]}]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[0].bar[3].baz[1]", "qux", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\",null,{\"baz\":[null,\"qux\"]}]},{\"bar\":[\"ba\",\"baz\",\"bazbaz\",{\"baz\":[\"quxqux\",\"qux\"]}]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_object_with_array(t *testing.T) {
	setupDir(testDir, testFile)
	os.Remove(testFilePath)

	setMeta("foo.bar[1]", "baz", testDir, testFile, false)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"foo\":{\"bar\":[null,\"baz\"]}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.bar[0]", "baz0", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":{\"bar\":[\"baz0\",\"baz\"]}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.barbar[2]", "bazbaz", testDir, testFile, false)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":{\"bar\":[\"baz0\",\"baz\"],\"barbar\":[null,null,\"bazbaz\"]}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_json_object(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				r := recover()
				assert.Equal(t, tc.wantErr, r != nil, "wantErr %v; err %v", tc.wantErr, r)
			}()

			require.NoError(t, setupDir(testDir, testFile))
			require.NoError(t, os.Remove(testFilePath))

			require.NoError(t, setMeta(tc.key, tc.value, testDir, testFile, true))
			out, err := ioutil.ReadFile(testFilePath)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, string(out))
		})
	}
}

func TestValidateMetaKeyWithAccept(t *testing.T) {
	testKey := "foo"
	r := validateMetaKey(testKey)
	if r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "f-o-o"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo[]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "f-o-o[]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo[0]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo[1]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo[10]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "a[10][20]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "a-b[10][20]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo.bar"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo[].bar"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo[1].bar"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo.bar[]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo.bar[1]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo[].bar[]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo[1].bar[1]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo.bar.baz"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo.bar-baz"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "f-o-o.bar--baz"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo[1].bar[2].baz[3]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "foo[1].bar-baz[2]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "f-o-o[1].bar--baz[2]"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}

	testKey = "1.2.3"
	if r = validateMetaKey(testKey); r == false {
		t.Fatalf("'%v' is should be accepted", testKey)
	}
}

func TestValidateMetaKeyWithReject(t *testing.T) {
	testKey := "foo[["
	r := validateMetaKey(testKey)
	if r == true {
		t.Fatalf("'%v' is should be rejected", testKey)
	}

	testKey = "foo[]]"
	if r = validateMetaKey(testKey); r == true {
		t.Fatalf("'%v' is should be rejected", testKey)
	}

	testKey = "foo[[]]"
	if r = validateMetaKey(testKey); r == true {
		t.Fatalf("'%v' is should be rejected", testKey)
	}

	testKey = "foo[1e]"
	if r = validateMetaKey(testKey); r == true {
		t.Fatalf("'%v' is should be rejected", testKey)
	}

	testKey = "foo[01]"
	if r = validateMetaKey(testKey); r == true {
		t.Fatalf("'%v' is should be rejected", testKey)
	}

	testKey = "foo."
	if r = validateMetaKey(testKey); r == true {
		t.Fatalf("'%v' is should be rejected", testKey)
	}

	testKey = "foo.[]"
	if r = validateMetaKey(testKey); r == true {
		t.Fatalf("'%v' is should be rejected", testKey)
	}

	testKey = "-foo"
	if r = validateMetaKey(testKey); r == true {
		t.Fatalf("'%v' is should be rejected", testKey)
	}

	testKey = "foo-[]"
	if r = validateMetaKey(testKey); r == true {
		t.Fatalf("'%v' is should be rejected", testKey)
	}

	testKey = "foo.-bar"
	if r = validateMetaKey(testKey); r == true {
		t.Fatalf("'%v' is should be rejected", testKey)
	}

	testKey = "foo.bar-[]"
	if r = validateMetaKey(testKey); r == true {
		t.Fatalf("'%v' is should be rejected", testKey)
	}
}

func TestIndexOfFirstRightBracket(t *testing.T) {
	key := "foo[1]"
	i := indexOfFirstRightBracket(key)
	expected := 5

	if i != expected {
		t.Fatalf("Expected '%d' but '%d'", expected, i)
	}

	key = "foo[10]"
	i = indexOfFirstRightBracket(key)
	expected = 6

	if i != expected {
		t.Fatalf("Expected '%d' but '%d'", expected, i)
	}
}

func TestMetaIndexFromKey(t *testing.T) {
	key := "foo[1]"
	i := metaIndexFromKey(key)
	expected := 1

	if i != expected {
		t.Fatalf("Expected '%d' but '%d'", expected, i)
	}

	key = "foo[10]"
	i = metaIndexFromKey(key)
	expected = 10

	if i != expected {
		t.Fatalf("Expected '%d' but '%d'", expected, i)
	}

	key = "foo[10].bar[4].baz"
	i = metaIndexFromKey(key)
	expected = 10

	if i != expected {
		t.Fatalf("Expected '%d' but '%d'", expected, i)
	}

	key = "foo[]"
	i = metaIndexFromKey(key)
	expected = 0

	if i != expected {
		t.Fatalf("Expected '%d' but '%d'", expected, i)
	}
}

func TestSymmetry_json_object(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
			// Get the non-json value from mock
			stdout := new(bytes.Buffer)
			require.NoError(t, getMeta(tc.key, mockDir, testFile, stdout, false, nil))
			nonJsonValue := stdout.String()

			// Get the json value from mock
			stdout = new(bytes.Buffer)
			require.NoError(t, getMeta(tc.key, mockDir, testFile, stdout, true, nil))
			jsonValue := stdout.String()

			// Compare starting condition
			if tc.expectJsonEqualNonJson {
				assert.Equal(t, jsonValue, nonJsonValue)
			} else {
				assert.NotEqual(t, jsonValue, nonJsonValue)
			}

			// Reset the output for writing
			require.NoError(t, setupDir(testDir, testFile))
			require.NoError(t, os.Remove(testFilePath))

			// Set and get the jsonValue to/from writable file with jsonValue true
			require.NoError(t, setMeta(tc.key, jsonValue, testDir, testFile, true))
			stdout = new(bytes.Buffer)
			require.NoError(t, getMeta(tc.key, testDir, testFile, stdout, true, nil))
			newJsonValue := stdout.String()
			assert.Equal(t, jsonValue, newJsonValue)
		})
	}
}

func TestMetaSpec_IsExternal(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			got := tt.metaSpec.IsExternal()
			assert.Equal(t, tt.want, got)
		})
	}
}

const (
	jobsJson = `
[
  {
    "id": 392545,
    "name": "competing-meta-join",
    "permutations": [
      {
        "annotations": {},
        "commands": [
          {
            "name": "nop",
            "command": "meta get meta"
          },
          {
            "name": "teardown-gather-meta",
            "command": "cp -r \"$(dirname \"$SD_META_PATH\")\" \"${SD_ARTIFACTS_DIR}/\""
          }
        ],
        "environment": {
          "SD_TEMPLATE_FULLNAME": "sieve/nop",
          "SD_TEMPLATE_NAME": "nop",
          "SD_TEMPLATE_NAMESPACE": "sieve",
          "SD_TEMPLATE_VERSION": "1.0.0"
        },
        "image": "docker.ouroath.com:4443/astracloud/no-op-base:latest",
        "secrets": [],
        "settings": {},
        "requires": [
          "competing-meta-1",
          "competing-meta-2"
        ]
      }
    ],
    "pipelineId": 1016709,
    "state": "ENABLED",
    "archived": false
  },
  {
    "id": 392544,
    "name": "competing-meta-2",
    "permutations": [
      {
        "annotations": {},
        "commands": [
          {
            "name": "nop",
            "command": "meta set meta.foo bar\nmeta set meta.competing-meta-2 abc\n"
          },
          {
            "name": "teardown-gather-meta",
            "command": "cp -r \"$(dirname \"$SD_META_PATH\")\" \"${SD_ARTIFACTS_DIR}/\""
          }
        ],
        "environment": {
          "SD_TEMPLATE_FULLNAME": "sieve/nop",
          "SD_TEMPLATE_NAME": "nop",
          "SD_TEMPLATE_NAMESPACE": "sieve",
          "SD_TEMPLATE_VERSION": "1.0.0"
        },
        "image": "docker.ouroath.com:4443/astracloud/no-op-base:latest",
        "secrets": [],
        "settings": {},
        "requires": [
          "~pr",
          "~commit"
        ]
      }
    ],
    "pipelineId": 1016709,
    "state": "ENABLED",
    "archived": false
  },
  {
    "id": 392543,
    "name": "competing-meta-1",
    "permutations": [
      {
        "annotations": {},
        "commands": [
          {
            "name": "nop",
            "command": "meta set meta.foo bar\nmeta set meta.competing-meta-1 abc\n"
          },
          {
            "name": "teardown-gather-meta",
            "command": "cp -r \"$(dirname \"$SD_META_PATH\")\" \"${SD_ARTIFACTS_DIR}/\""
          }
        ],
        "environment": {
          "SD_TEMPLATE_FULLNAME": "sieve/nop",
          "SD_TEMPLATE_NAME": "nop",
          "SD_TEMPLATE_NAMESPACE": "sieve",
          "SD_TEMPLATE_VERSION": "1.0.0"
        },
        "image": "docker.ouroath.com:4443/astracloud/no-op-base:latest",
        "secrets": [],
        "settings": {},
        "requires": [
          "~pr",
          "~commit"
        ]
      }
    ],
    "pipelineId": 1016709,
    "state": "ENABLED",
    "archived": false
  },
  {
    "id": 392535,
    "name": "see-if-external-propagates",
    "permutations": [
      {
        "annotations": {},
        "commands": [
          {
            "name": "nop",
            "command": "meta set meta.foo bar"
          },
          {
            "name": "teardown-gather-meta",
            "command": "cp -r \"$(dirname \"$SD_META_PATH\")\" \"${SD_ARTIFACTS_DIR}/\""
          }
        ],
        "environment": {
          "SD_TEMPLATE_FULLNAME": "sieve/nop",
          "SD_TEMPLATE_NAME": "nop",
          "SD_TEMPLATE_NAMESPACE": "sieve",
          "SD_TEMPLATE_VERSION": "1.0.0"
        },
        "image": "docker.ouroath.com:4443/astracloud/no-op-base:latest",
        "secrets": [],
        "settings": {},
        "requires": [
          "~fetch-from-pipeline1"
        ]
      }
    ],
    "pipelineId": 1016709,
    "state": "ENABLED",
    "archived": false
  },
  {
    "id": 392534,
    "name": "fetch-from-pipeline1",
    "permutations": [
      {
        "annotations": {},
        "commands": [
          {
            "name": "nop",
            "command": "curl -fs https://api.screwdriver.ouroath.com/v4/jobs/392524/lastSuccessfulMeta -H \"Authorization: Bearer ${SD_TOKEN}\" -o \"$(dirname \"$SD_META_PATH\")/sd@1016708:job1.json\"\nmeta get --external sd@1016708:job1 meta\n"
          },
          {
            "name": "teardown-gather-meta",
            "command": "cp -r \"$(dirname \"$SD_META_PATH\")\" \"${SD_ARTIFACTS_DIR}/\""
          }
        ],
        "environment": {
          "SD_TEMPLATE_FULLNAME": "sieve/nop",
          "SD_TEMPLATE_NAME": "nop",
          "SD_TEMPLATE_NAMESPACE": "sieve",
          "SD_TEMPLATE_VERSION": "1.0.0"
        },
        "image": "docker.ouroath.com:4443/astracloud/no-op-base:latest",
        "secrets": [],
        "settings": {},
        "requires": [
          "~pr",
          "~commit"
        ]
      }
    ],
    "pipelineId": 1016709,
    "state": "ENABLED",
    "archived": false
  },
  {
    "id": 392525,
    "name": "job1",
    "permutations": [
      {
        "annotations": {},
        "commands": [
          {
            "name": "nop",
            "command": "meta set meta.foo bar"
          },
          {
            "name": "teardown-gather-meta",
            "command": "cp -r \"$(dirname \"$SD_META_PATH\")\" \"${SD_ARTIFACTS_DIR}/\""
          }
        ],
        "environment": {
          "SD_TEMPLATE_FULLNAME": "sieve/nop",
          "SD_TEMPLATE_NAME": "nop",
          "SD_TEMPLATE_NAMESPACE": "sieve",
          "SD_TEMPLATE_VERSION": "1.0.0"
        },
        "image": "docker.ouroath.com:4443/astracloud/no-op-base:latest",
        "secrets": [],
        "settings": {},
        "requires": [
          "~pr",
          "~commit"
        ]
      }
    ],
    "pipelineId": 1016709,
    "state": "ENABLED",
    "archived": false
  }
]
`
	metaJson = `{"foo","bar","arr":[1,2,3]}`
)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

func TestMetaSpec_GetExternalData(t *testing.T) {
	for _, tt := range []struct {
		name     string
		external string
		expected string
		wantErr  bool
	}{
		{
			name:     "sd@1016708:job1",
			external: "sd@1016708:job1",
			expected: metaJson,
		},
		{
			name:     "sd@123:missing",
			external: "sd@123:missing",
			wantErr:  true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var mockHandler MockHandler
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/pipelines/1016708/jobs"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), jobsJson)
				})
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/jobs/392525/lastSuccessfulMeta"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), metaJson)
				})
			testServer := httptest.NewServer(&mockHandler)
			defer testServer.Close()

			tempDir, err := ioutil.TempDir("", "test")
			require.NoError(t, err)
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
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(got))
			mockHandler.AssertExpectations(t)
		})
	}
}
