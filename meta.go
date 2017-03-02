package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"runtime/debug"
	"strconv"

	"github.com/urfave/cli"
)

// VERSION gets set by the build script via the LDFLAGS
var VERSION string

var mkdirAll = os.MkdirAll
var stat = os.Stat
var writeFile = ioutil.WriteFile
var readFile = ioutil.ReadFile
var fprintf = fmt.Fprintf

const metaFile = "meta.json"
const metaKeyValidator = `^\w+(((\[\]|\[(0|[1-9]\d*)\]))?(\.\w+)*)*$`

// Get meta from file based on key
func getMeta(key string, metaSpace string, output io.Writer) error {
	metaFilePath := metaSpace + "/" + metaFile
	metaJson, err := readFile(metaFilePath)
	if err != nil {
		return err
	}
	var metaInterface map[string]interface{}
	err = json.Unmarshal(metaJson, &metaInterface)
	if err != nil {
		return err
	}

	result := metaInterface[key]
	switch result.(type) {
	case map[string]interface{}, []interface{}:
		resultJson, _ := json.Marshal(result)
		fprintf(output, "%v", string(resultJson))
	case nil:
		fprintf(output, "null")
	default:
		fprintf(output, "%v", result)
	}

	return nil
}

// Store meta to file with key and value
func setMeta(key string, value string, metaSpace string) error {
	metaFilePath := metaSpace + "/" + metaFile
	var previousMeta map[string]interface{}

	_, err := stat(metaFilePath)
	// Not exist directory
	if err != nil {
		err = setupDir(metaSpace)
		if err != nil {
			return err
		}
		// Initialize interface if first setting meta
		previousMeta = make(map[string]interface{})
	} else {
		metaJson, _ := readFile(metaFilePath)
		// Exist meta.json
		if len(metaJson) != 0 {
			err = json.Unmarshal(metaJson, &previousMeta)
			if err != nil {
				return err
			}
		} else {
			// Exist meta.json but it is empty
			previousMeta = make(map[string]interface{})
		}
	}

	key, parsedValue := parseMetaValue(key, value, previousMeta)
	previousMeta[key] = parsedValue

	resultJson, err := json.Marshal(previousMeta)

	err = writeFile(metaFilePath, resultJson, 0666)
	if err != nil {
		return err
	}
	return nil
}

// Parse arguments of meta-cli to JSON
func parseMetaValue(key string, value string, previousMeta interface{}) (string, interface{}) {
	for position, char := range key {
		if string([]rune{char}) == "[" {
			nextChar := key[position+1]
			if nextChar == []byte("]")[0] {
				// Value is array
				var metaValue [1]interface{}
				key = key[0:position] + key[position+2:] // Remove bracket[] from key
				key, metaValue[0] = parseMetaValue(key, value, previousMeta)
				return key, metaValue
			} else {
				// Value is array with index
				var i int
				for i = position + 1; ; i++ {
					_, err := strconv.Atoi(string(key[i])) // Check the next char is integer
					if err != nil {
						break
					}
				}
				metaIndex, _ := strconv.Atoi(key[position+1 : i]) // e.g. if array[10], get "10"
				key = key[0:position] + key[i+1:]                 // Remove bracket[num] from key

				// Convert previousMeta Interface to Map
				previousMetaValue := reflect.ValueOf(previousMeta)
				var previousMetaMap map[string]interface{}
				previousMetaMap = make(map[string]interface{})
				var previousKey string
				if previousMetaValue.Kind() == reflect.Map {
					for _, k := range previousMetaValue.MapKeys() {
						previousKey, _ = k.Interface().(string)
						previousMetaMap[previousKey] = previousMetaValue.MapIndex(k).Interface()
					}
				}

				var metaValue []interface{}
				// previousMetaMap[previousKey] is empty or string, create array with null except "value"
				if previousMetaMap[previousKey] == nil || reflect.ValueOf(previousMetaMap[previousKey]).Kind() == reflect.String {
					metaValue = make([]interface{}, metaIndex+1)
					key, metaValue[metaIndex] = parseMetaValue(key, value, previousMetaMap[previousKey])
					return key, metaValue
				} else {
					previousObject := reflect.ValueOf(previousMetaMap[previousKey])
					if metaIndex+1 > previousObject.Len() {
						metaValue = make([]interface{}, metaIndex+1)
						key, metaValue[metaIndex] = parseMetaValue(key, value, nil)
					} else {
						metaValue = make([]interface{}, previousObject.Len())
						key, metaValue[metaIndex] = parseMetaValue(key, value, previousObject.Index(metaIndex).Interface())
					}
					// Insert previousValues to metaVelue[] when previousObject is Array
					if previousObject.Kind() == reflect.Slice {
						for i := 0; i < previousObject.Len(); i++ {
							if i != metaIndex {
								metaValue[i] = previousObject.Index(i).Interface()
							}
						}
					}
					return key, metaValue
				}
			}
		} else if string([]rune{char}) == "." {
			// Value is object
			childKey := key[position+1:]
			key = key[0:position]
			var obj map[string]interface{}
			obj = make(map[string]interface{})
			childKey, tmpValue := parseMetaValue(childKey, value, previousMeta)
			obj[childKey] = tmpValue
			return key, obj
		}
	}
	// Value is int
	i, err := strconv.Atoi(value)
	if err == nil {
		return key, i
	}
	// Value is float
	f, err := strconv.ParseFloat(value, 64)
	if err == nil {
		return key, f
	}
	// Value is bool
	b, err := strconv.ParseBool(value)
	if err == nil {
		return key, b
	}
	// Value is string
	return key, value
}

// setupDir makes directory and json file for meta
func setupDir(metaSpace string) error {
	err := mkdirAll(metaSpace, 0777)
	if err != nil {
		return err
	}
	err = writeFile(metaSpace+"/"+metaFile, []byte(""), 0666)
	if err != nil {
		return err
	}
	return nil
}

// validate key of metaSet
func validateMetaKey(key string) bool {
	return regexp.MustCompile(metaKeyValidator).MatchString(key)
}

var successExit = func() {
	os.Exit(0)
}

var failureExit = func(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	}
	os.Exit(1)
}

// finalRecover makes one last attempt to recover from a panic.
// This should only happen if the previous recovery caused a panic.
func finalRecover() {
	if p := recover(); p != nil {
		fmt.Fprintln(os.Stderr, "ERROR: Something terrible has happened. Please file a ticket with this info:")
		fmt.Fprintf(os.Stderr, "ERROR: %v\n%v\n", p, string(debug.Stack()))
		failureExit(nil)
	}
	successExit()
}

func main() {
	defer finalRecover()

	var metaSpace string

	app := cli.NewApp()
	app.Name = "meta-cli"
	app.Usage = "get or set metadata for Screwdriver build"
	app.UsageText = "meta command arguments [options]"
	app.Copyright = "(c) 2017 Yahoo Inc."

	if VERSION == "" {
		VERSION = "0.0.0"
	}
	app.Version = VERSION

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "meta-space",
			Usage:       "Location of meta temporarily",
			Value:       "/sd/meta",
			Destination: &metaSpace,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "get",
			Usage: "Get a metadata with key",
			Action: func(c *cli.Context) error {
				if len(c.Args()) == 0 {
					return cli.ShowAppHelp(c)
				}
				err := getMeta(c.Args().First(), metaSpace, os.Stdout)
				if err != nil {
					failureExit(err)
				}
				successExit()
				return nil
			},
			Flags: app.Flags,
		},
		{
			Name:  "set",
			Usage: "Set a metadata with key and value",
			Action: func(c *cli.Context) error {
				if len(c.Args()) <= 1 {
					return cli.ShowAppHelp(c)
				}
				key := c.Args().Get(0)
				val := c.Args().Get(1)
				if valid := validateMetaKey(key); valid == false {
					failureExit(errors.New("Meta key validation error"))
				}
				err := setMeta(key, val, metaSpace)
				if err != nil {
					failureExit(err)
				}
				successExit()
				return nil
			},
			Flags: app.Flags,
		},
	}

	app.Run(os.Args)
}
