package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

const testFile = "meta"
const testDir = "./_test"
const testFilePath = testDir + "/" + testFile + ".json"
const mockDir = "./mock"
const externalFile = "sd@123:component"
const externalFilePath = testDir + "/" + externalFile + ".json"
const doesNotExistFile = "woof"

func TestMain(m *testing.M) {
	// setup functions
	setupDir(testDir, testFile)
	// run test
	retCode := m.Run()
	// teardown functions
	os.RemoveAll(testDir)
	os.Exit(retCode)
}

func TestSetupDir(t *testing.T) {
	var err error
	var data []byte

	os.RemoveAll(testDir)

	setupDir(testDir, testFile)
	_, err = os.Stat(testFilePath)
	if err != nil {
		t.Errorf("could not create %s in %s", testFilePath, testDir)
	}

	data, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Errorf("could not read %s in %s", testFilePath, testDir)
	}
	if string(data[:]) != "{}" {
		t.Errorf("%s does not have an empty JSON object: %v", testFilePath, string(data[:]))
	}
}

func TestExternalMetaFile(t *testing.T) {
	setupDir(testDir, externalFile)
	os.Remove(externalFilePath)

	// Test set (meta file is not meta.json, should fail)
	err := setMeta("str", "val", testDir, externalFile)
	if err == nil {
		t.Fatalf("error should be occured")
	}

	// Test get
	stdout := new(bytes.Buffer)
	getMeta("str", mockDir, externalFile, stdout)
	expected := []byte("meow")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}
}

func TestGetMetaNoFile(t *testing.T) {
	os.RemoveAll(testDir)

	stdout := new(bytes.Buffer)
	getMeta("woof", testDir, doesNotExistFile, stdout)
	expected := []byte("null")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}
}

func TestGetMeta(t *testing.T) {
	stdout := new(bytes.Buffer)
	getMeta("str", mockDir, testFile, stdout)
	expected := []byte("fuga")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("bool", mockDir, testFile, stdout)
	expected = []byte("true")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("int", mockDir, testFile, stdout)
	expected = []byte("1234567")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("float", mockDir, testFile, stdout)
	expected = []byte("1.5")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("foo.bar-baz", mockDir, testFile, stdout)
	expected = []byte("dashed-key")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("obj", mockDir, testFile, stdout)
	expected = []byte("{\"ccc\":\"ddd\",\"momo\":{\"toke\":\"toke\"}}")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("obj.ccc", mockDir, testFile, stdout)
	expected = []byte("ddd")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("obj.momo", mockDir, testFile, stdout)
	expected = []byte("{\"toke\":\"toke\"}")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("ary", mockDir, testFile, stdout)
	expected = []byte("[\"aaa\",\"bbb\",{\"ccc\":{\"ddd\":[1234567,2,3]}}]")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("ary[0]", mockDir, testFile, stdout)
	expected = []byte("aaa")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("ary[2]", mockDir, testFile, stdout)
	expected = []byte("{\"ccc\":{\"ddd\":[1234567,2,3]}}")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("ary[2].ccc", mockDir, testFile, stdout)
	expected = []byte("{\"ddd\":[1234567,2,3]}")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("ary[2].ccc.ddd", mockDir, testFile, stdout)
	expected = []byte("[1234567,2,3]")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("ary[2].ccc.ddd[1]", mockDir, testFile, stdout)
	expected = []byte("2")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("nu", mockDir, testFile, stdout)
	expected = []byte("null")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	// The key does not exist in meta.json
	stdout = new(bytes.Buffer)
	getMeta("notexist", mockDir, testFile, stdout)
	expected = []byte("null")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	// It makes golang zero-value
	stdout = new(bytes.Buffer)
	getMeta("ary[]", mockDir, testFile, stdout)
	expected = []byte("aaa")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	// The key does not exist in meta.json
	stdout = new(bytes.Buffer)
	getMeta("ary.aaa.bbb.ccc.ddd[10]", mockDir, testFile, stdout)
	expected = []byte("null")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}
}

func TestSetMeta_bool(t *testing.T) {
	setupDir(testDir, testFile)
	os.Remove(testFilePath)

	setMeta("bool", "true", testDir, testFile)
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

	setMeta("int", "10", testDir, testFile)
	setMeta("float", "15.5", testDir, testFile)
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

	setMeta("str", "val", testDir, testFile)
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

	setMeta("foo-bar", "val", testDir, testFile)
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

	setMeta("array[]", "arg", testDir, testFile)
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

	setMeta("array[1]", "arg", testDir, testFile)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"array\":[null,\"arg\"]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("array[2]", "argarg", testDir, testFile)
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

	setMeta("array[1]", "arg", testDir, testFile)
	setMeta("array", "str", testDir, testFile)
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

	setMeta("foo.bar", "baz", testDir, testFile)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"foo\":{\"bar\":\"baz\"}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.barbar", "bazbaz", testDir, testFile)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":{\"bar\":\"baz\",\"barbar\":\"bazbaz\"}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.bar.baz", "piyo", testDir, testFile)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":{\"bar\":{\"baz\":\"piyo\"},\"barbar\":\"bazbaz\"}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.bar-baz", "dashed-key", testDir, testFile)
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

	setMeta("foo.bar", "baz", testDir, testFile)
	setMeta("foo", "baz", testDir, testFile)
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

	setMeta("foo[1].bar", "baz", testDir, testFile)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"foo\":[null,{\"bar\":\"baz\"}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.bar[1]", "baz", testDir, testFile)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":{\"bar\":[null,\"baz\"]}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[1]", "baz", testDir, testFile)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":[null,{\"bar\":[null,\"baz\"]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[0].bar[1]", "baz", testDir, testFile)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[null,\"baz\"]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[0]", "ba", testDir, testFile)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[\"ba\",\"baz\"]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[2]", "bazbaz", testDir, testFile)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[\"ba\",\"baz\",\"bazbaz\"]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[3].baz[1]", "qux", testDir, testFile)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[\"ba\",\"baz\",\"bazbaz\",{\"baz\":[null,\"qux\"]}]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[3].baz[0]", "quxqux", testDir, testFile)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[\"ba\",\"baz\",\"bazbaz\",{\"baz\":[\"quxqux\",\"qux\"]}]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[0].bar[3].baz[1]", "qux", testDir, testFile)
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

	setMeta("foo.bar[1]", "baz", testDir, testFile)
	out, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected := []byte("{\"foo\":{\"bar\":[null,\"baz\"]}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.bar[0]", "baz0", testDir, testFile)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":{\"bar\":[\"baz0\",\"baz\"]}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.barbar[2]", "bazbaz", testDir, testFile)
	out, err = ioutil.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Meta file did not create. error: %v", err)
	}
	expected = []byte("{\"foo\":{\"bar\":[\"baz0\",\"baz\"],\"barbar\":[null,null,\"bazbaz\"]}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
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
