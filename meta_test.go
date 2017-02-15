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
	//readFile = func(filename string) ([]byte, error) { return ioutil.ReadFile("./test/test.json") }
	writeFile = func(filename string, data []byte, perm os.FileMode) error {
		return ioutil.WriteFile(testFilePath, data, 0666)
	}
	/*
		printf = func(format string, a ...interface{}) (n int, err error) {
			stdout := new(bytes.Buffer)
			fmt.Printf("%v", a)
			fmt.Printf("\n%v", format)
			fmt.Printf("\n%v", stdout)
			return fmt.Fprintf(stdout, format, a)
		}
	*/
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
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(stdout.Bytes()[:]))
	}

	stdout = new(bytes.Buffer)
	getMeta("bool", mockDir, stdout)
	expected = []byte("true")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(stdout.Bytes()[:]))
	}

	stdout = new(bytes.Buffer)
	getMeta("int", mockDir, stdout)
	expected = []byte("1")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(stdout.Bytes()[:]))
	}

	stdout = new(bytes.Buffer)
	getMeta("float", mockDir, stdout)
	expected = []byte("1.5")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(stdout.Bytes()[:]))
	}

	stdout = new(bytes.Buffer)
	getMeta("obj", mockDir, stdout)
	expected = []byte("{\"ccc\":\"ddd\",\"momo\":{\"toke\":\"toke\"}}")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(stdout.Bytes()[:]))
	}

	stdout = new(bytes.Buffer)
	getMeta("ary", mockDir, stdout)
	expected = []byte("[\"aaa\",\"bbb\"]")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(stdout.Bytes()[:]))
	}

	stdout = new(bytes.Buffer)
	getMeta("nu", mockDir, stdout)
	expected = []byte("null")
	if bytes.Compare(expected, stdout.Bytes()) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(stdout.Bytes()[:]))
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
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(out))
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
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(out))
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
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(out))
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
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(out))
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
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(out))
	}

	setMeta("array[2]", "argarg", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"array\":[null,\"arg\",\"argarg\"]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(out))
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
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(out))
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
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(out))
	}

	setMeta("foo.bar.baz", "piyo", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":{\"bar\":{\"baz\":\"piyo\"}}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(out))
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
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(out))
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
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(out))
	}

	setMeta("foo.bar[1]", "baz", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":{\"bar\":[null,\"baz\"]}}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(out))
	}

	setMeta("foo[1].bar[1]", "baz", testDir)
	out, err = exec.Command("cat", testFilePath).Output()
	if err != nil {
		t.Fatal("Meta file did not create.")
	}
	expected = []byte("{\"foo\":[null,{\"bar\":[null,\"baz\"]}]}")
	if bytes.Compare(expected, out) != 0 {
		t.Fatalf("not matched. expected '%v', actual '%v'", string(expected[:]), string(out))
	}

}
