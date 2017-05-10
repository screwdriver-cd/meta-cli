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
	"strings"

	"github.com/urfave/cli"
)

// VERSION gets set by the build script via the LDFLAGS
var VERSION string

var mkdirAll = os.MkdirAll
var stat = os.Stat
var writeFile = ioutil.WriteFile
var readFile = ioutil.ReadFile
var fprintf = fmt.Fprintf

var metaKeyValidator = regexp.MustCompile(`^\w+(((\[\]|\[(0|[1-9]\d*)\]))?(\.\w+)*)*$`)
var rightBracketRegExp = regexp.MustCompile(`\[(.*?)\]`)

const metaFile = "meta.json"

// getMeta prints meta value from file based on key
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

	_, result := fetchMetaValue(key, metaInterface)

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

// indexOfFirstRightBracket gets index of right bracket("]"). e.g. the key is foo[10].bar[4], return 6
func indexOfFirstRightBracket(key string) int {
	return (rightBracketRegExp.FindStringIndex(key)[1] - 1)
}

// metaIndexFromKey gets number in brackets. e.g. the key is foo[10].bar[4], return 10
func metaIndexFromKey(key string) int {
	indexString := rightBracketRegExp.FindStringSubmatch(key)[1]
	index, err := strconv.Atoi(indexString)
	if err != nil {
		return 0
	}
	return index
}

// convertInterfaceToMap converts interface{} to map[string]interface{} via Value
func convertInterfaceToMap(metaInterface interface{}) map[string]interface{} {
	metaValue := reflect.ValueOf(metaInterface)
	metaMap := make(map[string]interface{})
	if metaValue.Kind() == reflect.Map {
		for _, keyValue := range metaValue.MapKeys() {
			keyString, _ := keyValue.Interface().(string)
			metaMap[keyString] = metaValue.MapIndex(keyValue).Interface()
		}
	} else {
		return nil
	}
	return metaMap
}

// convertInterfaceToSlice converts interface{} to []interface{} via Value
func convertInterfaceToSlice(metaInterface interface{}) []interface{} {
	metaValue := reflect.ValueOf(metaInterface)
	metaSlice := make([]interface{}, metaValue.Len())
	if metaValue.Kind() == reflect.Slice {
		for i := 0; i < metaValue.Len(); i++ {
			metaSlice[i] = metaValue.Index(i).Interface()
		}
	} else {
		return nil
	}
	return metaSlice
}

// fetchMetaValue fetches value from meta by using key
func fetchMetaValue(key string, meta interface{}) (string, interface{}) {
	var result interface{}
	for current, char := range key {
		if string([]rune{char}) == "[" {
			// Value is array with index
			rightBracket := indexOfFirstRightBracket(key)
			metaIndex := metaIndexFromKey(key) // e.g. if key is foo[10], get "10"
			shortenKey := key[rightBracket+1:] // e.g. foo[10].bar -> .bar
			metaMap := convertInterfaceToMap(meta)
			if metaMap == nil {
				return "", nil
			}

			childMeta := metaMap[key[0:current]]
			childMetaSlice := convertInterfaceToSlice(childMeta)
			if childMetaSlice == nil {
				return "", nil
			}
			return fetchMetaValue(shortenKey, childMetaSlice[metaIndex])
		} else if string([]rune{char}) == "." {
			// Value is object
			childKey := strings.Split(key, ".")[0]                       // e.g. foo.bar.baz -> foo
			shortenKey := strings.Join(strings.Split(key, ".")[1:], ".") // e.g. foo.bar.baz -> bar.baz
			metaMap := convertInterfaceToMap(meta)
			if metaMap == nil {
				return "", nil
			}
			if len(childKey) != 0 {
				return fetchMetaValue(shortenKey, metaMap[childKey])
			} else {
				return fetchMetaValue(shortenKey, metaMap)
			}
		}
	}
	if len(key) != 0 {
		// convert type interface -> Value -> map[string]interface{}
		var metaMap map[string]interface{}
		metaMap = convertInterfaceToMap(meta)
		result = metaMap[key]
	} else {
		result = meta
	}

	return key, result
}

// setMeta stores meta to file with key and value
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

	key, parsedValue := setMetaValueRecursive(key, value, previousMeta)
	previousMeta[key] = parsedValue

	resultJson, err := json.Marshal(previousMeta)

	err = writeFile(metaFilePath, resultJson, 0666)
	if err != nil {
		return err
	}
	return nil
}

// setMetaValueRecursive updates meta
func setMetaValueRecursive(key string, value string, previousMeta interface{}) (string, interface{}) {
	for current, char := range key {
		if string([]rune{char}) == "[" {
			nextChar := key[current+1]
			if nextChar == []byte("]")[0] {
				// Value is array
				var metaValue [1]interface{}
				key = key[0:current] + key[current+2:] // Remove bracket[] from key
				key, metaValue[0] = setMetaValueRecursive(key, value, previousMeta)
				return key, metaValue
			} else {
				// Value is array with index
				rightBracket := indexOfFirstRightBracket(key)
				metaIndex := metaIndexFromKey(key)   // e.g. if key is foo[10], get "10"
				keyHead := key[0:current]            // e.g. foo[10].bar -> foo
				key = keyHead + key[rightBracket+1:] // Remove bracket and number from key. e.g. foo[10].bar -> foo.bar

				previousMetaMap := convertInterfaceToMap(previousMeta)
				previousMetaValue := reflect.ValueOf(previousMetaMap[keyHead])
				var metaValue []interface{}

				// previousMetaMap[keyHead] is empty or string, create array with null except value of argument
				if previousMetaMap[keyHead] == nil || reflect.ValueOf(previousMetaMap[keyHead]).Kind() == reflect.String {
					metaValue = make([]interface{}, metaIndex+1)
					key, metaValue[metaIndex] = setMetaValueRecursive(key, value, previousMetaMap[keyHead])
				} else {
					if metaIndex+1 > previousMetaValue.Len() {
						metaValue = make([]interface{}, metaIndex+1)
						key, metaValue[metaIndex] = setMetaValueRecursive(key, value, nil)
					} else {
						metaValue = make([]interface{}, previousMetaValue.Len())
						key, metaValue[metaIndex] = setMetaValueRecursive(key, value, previousMetaValue.Index(metaIndex).Interface())
					}
				}
				// Insert previous values to metaVelue[] when previousMetaValue type is slice except new value
				if previousMetaValue.Kind() == reflect.Slice {
					for i := 0; i < previousMetaValue.Len(); i++ {
						if i != metaIndex {
							metaValue[i] = previousMetaValue.Index(i).Interface()
						}
					}
				}
				return key, metaValue
			}
		} else if string([]rune{char}) == "." {
			// Value is object
			keyHead := key[0:current]   // e.g. aaa.bbb -> aaa
			childKey := key[current+1:] // e.g. aaa.bbb -> bbb
			obj := make(map[string]interface{})
			var tmpValue interface{}
			previousMetaMap := convertInterfaceToMap(previousMeta)
			if previousMetaMap[keyHead] == nil {
				childKey, tmpValue = setMetaValueRecursive(childKey, value, previousMetaMap)
			} else {
				// copy previous object only if it is map
				previousObj := convertInterfaceToMap(previousMetaMap[keyHead])
				if len(previousObj) != 0 {
					obj = previousObj
				}
				childKey, tmpValue = setMetaValueRecursive(childKey, value, previousMetaMap[keyHead])
			}
			obj[childKey] = tmpValue
			return keyHead, obj
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

// validateMetaKey validates the key of argument
func validateMetaKey(key string) bool {
	return metaKeyValidator.MatchString(key)
}

// successExit exits process with 0
func successExit() {
	os.Exit(0)
}

// failureExit exits process with 1
func failureExit(err error) {
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
				key := c.Args().Get(0)
				if valid := validateMetaKey(key); valid == false {
					failureExit(errors.New("Meta key validation error"))
				}
				err := getMeta(key, metaSpace, os.Stdout)
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
