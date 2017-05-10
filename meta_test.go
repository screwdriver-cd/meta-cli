package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

const testDir = "./_test"
const testFilePath = testDir + "/" + metaFile
const mockDir = "./mock"

func TestMain(m *testing.M) {
	// setup functions
	setupDir(testDir)
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		return ioutil.WriteFile(testFilePath, data, 0666)
	}
	// run test
	retCode := m.Run()
	// teardown functions
	os.RemoveAll(testDir)
	os.Exit(retCode)
}

func TestSetupDir(t *testing.T) {
	os.RemoveAll(testDir)

	setupDir(testDir)
	_, err := os.Stat(testFilePath)
	if err != nil {
		t.Errorf("could not create %s in %s", metaFile, testDir)
	}
}

func TestGetMeta(t *testing.T) {
	stdout := new(bytes.Buffer)
	getMeta("str", mockDir, stdout)
	expected := []byte("fuga")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("bool", mockDir, stdout)
	expected = []byte("true")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("int", mockDir, stdout)
	expected = []byte("1")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("float", mockDir, stdout)
	expected = []byte("1.5")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("obj", mockDir, stdout)
	expected = []byte("{\"ccc\":\"ddd\",\"momo\":{\"toke\":\"toke\"}}")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("obj.ccc", mockDir, stdout)
	expected = []byte("ddd")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("obj.momo", mockDir, stdout)
	expected = []byte("{\"toke\":\"toke\"}")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("ary", mockDir, stdout)
	expected = []byte("[\"aaa\",\"bbb\",{\"ccc\":{\"ddd\":[1,2,3]}}]")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("ary[0]", mockDir, stdout)
	expected = []byte("aaa")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("ary[2]", mockDir, stdout)
	expected = []byte("{\"ccc\":{\"ddd\":[1,2,3]}}")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("ary[2].ccc", mockDir, stdout)
	expected = []byte("{\"ddd\":[1,2,3]}")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("ary[2].ccc.ddd", mockDir, stdout)
	expected = []byte("[1,2,3]")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("ary[2].ccc.ddd[1]", mockDir, stdout)
	expected = []byte("2")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	stdout = new(bytes.Buffer)
	getMeta("nu", mockDir, stdout)
	expected = []byte("null")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	// The key does not exist in meta.json
	stdout = new(bytes.Buffer)
	getMeta("notexist", mockDir, stdout)
	expected = []byte("null")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	// It makes golang zero-value
	stdout = new(bytes.Buffer)
	getMeta("ary[]", mockDir, stdout)
	expected = []byte("aaa")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}

	// The key does not exist in meta.json
	stdout = new(bytes.Buffer)
	getMeta("ary.aaa.bbb.ccc.ddd[10]", mockDir, stdout)
	expected = []byte("null")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(stdout.Bytes()))
	}
}

func TestGetMetaWithFailuer(t *testing.T) {
	// meta.json does not exist
	stdout := new(bytes.Buffer)
	err := getMeta("str", "not_exist", stdout)
	if err == nil {
		t.Fatalf("error should be occured")
	}
}

func TestSetMeta_bool(t *testing.T) {
	setupDir(testDir)
	os.Remove(testFilePath)

	setMeta("bool", "true", testDir)
	out, err := exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected := []byte("{\"bool\":true}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}
func TestSetMeta_number(t *testing.T) {
	setupDir(testDir)
	os.Remove(testFilePath)

	setMeta("int", "10", testDir)
	setMeta("float", "15.5", testDir)
	out, err := exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected := []byte("{\"float\":15.5,\"int\":10}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_string(t *testing.T) {
	setupDir(testDir)
	os.Remove(testFilePath)

	setMeta("str", "val", testDir)
	out, err := exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected := []byte("{\"str\":\"val\"}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_array(t *testing.T) {
	setupDir(testDir)
	os.Remove(testFilePath)

	setMeta("array[]", "arg", testDir)
	out, err := exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected := []byte("{\"array\":[\"arg\"]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_array_with_index(t *testing.T) {
	setupDir(testDir)
	os.Remove(testFilePath)

	setMeta("array[1]", "arg", testDir)
	out, err := exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected := []byte("{\"array\":[null,\"arg\"]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("array[2]", "argarg", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"array\":[null,\"arg\",\"argarg\"]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_array_with_index_to_string(t *testing.T) {
	setupDir(testDir)
	os.Remove(testFilePath)

	setMeta("array[1]", "arg", testDir)
	setMeta("array", "str", testDir)
	out, err := exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected := []byte("{\"array\":\"str\"}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_object(t *testing.T) {
	setupDir(testDir)
	os.Remove(testFilePath)

	setMeta("foo.bar", "baz", testDir)
	out, err := exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected := []byte("{\"foo\":{\"bar\":\"baz\"}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.barbar", "bazbaz", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":{\"bar\":\"baz\",\"barbar\":\"bazbaz\"}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.bar.baz", "piyo", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":{\"bar\":{\"baz\":\"piyo\"},\"barbar\":\"bazbaz\"}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_object_to_string(t *testing.T) {
	setupDir(testDir)
	os.Remove(testFilePath)

	setMeta("foo.bar", "baz", testDir)
	setMeta("foo", "baz", testDir)
	out, err := exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected := []byte("{\"foo\":\"baz\"}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_array_with_object(t *testing.T) {
	setupDir(testDir)
	os.Remove(testFilePath)

	setMeta("foo[1].bar", "baz", testDir)
	out, err := exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected := []byte("{\"foo\":[null,{\"bar\":\"baz\"}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.bar[1]", "baz", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":{\"bar\":[null,\"baz\"]}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[1]", "baz", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":[null,{\"bar\":[null,\"baz\"]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[0].bar[1]", "baz", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[null,\"baz\"]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[0]", "ba", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[\"ba\",\"baz\"]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[2]", "bazbaz", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[\"ba\",\"baz\",\"bazbaz\"]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[3].baz[1]", "qux", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[\"ba\",\"baz\",\"bazbaz\",{\"baz\":[null,\"qux\"]}]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[1].bar[3].baz[0]", "quxqux", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\"]},{\"bar\":[\"ba\",\"baz\",\"bazbaz\",{\"baz\":[\"quxqux\",\"qux\"]}]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo[0].bar[3].baz[1]", "qux", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":[{\"bar\":[null,\"baz\",null,{\"baz\":[null,\"qux\"]}]},{\"bar\":[\"ba\",\"baz\",\"bazbaz\",{\"baz\":[\"quxqux\",\"qux\"]}]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}
}

func TestSetMeta_object_with_array(t *testing.T) {
	setupDir(testDir)
	os.Remove(testFilePath)

	setMeta("foo.bar[1]", "baz", testDir)
	out, err := exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected := []byte("{\"foo\":{\"bar\":[null,\"baz\"]}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.bar[0]", "baz0", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":{\"bar\":[\"baz0\",\"baz\"]}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected), string(out))
	}

	setMeta("foo.barbar[2]", "bazbaz", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
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

	testKey = "foo[]"
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

	testKey = "foo[1].bar[2].baz[3]"
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

	testKey = "a-b"
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
