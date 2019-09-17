package main

import (
	"bytes"
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
	"time"

	"github.com/screwdriver-cd/meta-cli/internal/fetch"
	"github.com/sirupsen/logrus"

	"gopkg.in/urfave/cli.v1"
)

// These variables get set by the build script via the LDFLAGS
// Detail about these variables are here: https://goreleaser.com/#builds
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var mkdirAll = os.MkdirAll
var stat = os.Stat
var writeFile = ioutil.WriteFile
var readFile = ioutil.ReadFile
var fprintf = fmt.Fprintf

var metaKeyValidator = regexp.MustCompile(`^(\w+(-*\w+)*)+(((\[\]|\[(0|[1-9]\d*)\]))?(\.(\w+(-*\w+)*)+)*)*$`)
var rightBracketRegExp = regexp.MustCompile(`\[(.*?)\]`)

// getMeta prints meta value from file based on key
func getMeta(key string, metaSpace string, metaFile string, output io.Writer, jsonValue bool, lastSuccessfulMetaRequest *fetch.LastSuccessfulMetaRequest) error {
	metaFilePath := metaSpace + "/" + metaFile + ".json"

	_, err := stat(metaFilePath)
	// Setup directory if it does not exist
	if err != nil {
		logrus.Debugf("Err statting %v: %v", metaFilePath, err)
		err = setupDir(metaSpace, metaFile)
		if err != nil {
			return err
		}
		if lastSuccessfulMetaRequest == nil || metaFile == "meta" {
			_, err = io.WriteString(output, "null")
			return err
		}
		jobDescription, err := fetch.ParseJobDescription(lastSuccessfulMetaRequest.DefaultSdPipelineId, metaFile)
		if err != nil {
			return err
		}
		data, err := lastSuccessfulMetaRequest.FetchLastSuccessfulMeta(jobDescription)
		if err != nil {
			return err
		}
		err = writeFile(metaFilePath, data, 0666)
		if err != nil {
			return err
		}
	}

	logrus.Tracef("Reading file %v", metaFilePath)
	metaJSON, err := readFile(metaFilePath)
	if err != nil {
		return err
	}

	var metaInterface map[string]interface{}
	// for Unmarshal integer as integer, not float64
	decoder := json.NewDecoder(bytes.NewReader(metaJSON))
	decoder.UseNumber()
	err = decoder.Decode(&metaInterface)
	if err != nil {
		return err
	}

	_, result := fetchMetaValue(key, metaInterface)

	switch result.(type) {
	case map[string]interface{}, []interface{}:
		resultJSON, _ := json.Marshal(result)
		_, err = fprintf(output, "%v", string(resultJSON))
	case nil:
		_, err = fprintf(output, "null")
	default:
		if jsonValue {
			resultJSON, _ := json.Marshal(result)
			_, err = fprintf(output, "%v", string(resultJSON))
		} else {
			_, err = fprintf(output, "%v", result)
		}
	}

	return err
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
			}
			return fetchMetaValue(shortenKey, metaMap)
		}
	}
	if len(key) != 0 {
		// convert type interface -> Value -> map[string]interface{}
		var metaMap map[string]interface{} = convertInterfaceToMap(meta)
		result = metaMap[key]
	} else {
		result = meta
	}

	return key, result
}

// setMeta stores meta to file with key and value
func setMeta(key string, value string, metaSpace string, metaFile string, jsonValue bool) error {
	metaFilePath := metaSpace + "/" + metaFile + ".json"
	var previousMeta map[string]interface{}

	if metaFile != "meta" {
		return errors.New("can only meta set current build meta")
	}

	_, err := stat(metaFilePath)
	// Not exist directory
	if err != nil {
		err = setupDir(metaSpace, metaFile)
		if err != nil {
			return err
		}
		// Initialize interface if first setting meta
		previousMeta = make(map[string]interface{})
	} else {
		metaJSON, _ := readFile(metaFilePath)
		// Exist meta.json
		if len(metaJSON) != 0 {
			err = json.Unmarshal(metaJSON, &previousMeta)
			if err != nil {
				return err
			}
		} else {
			// Exist meta.json but it is empty
			previousMeta = make(map[string]interface{})
		}
	}

	key, parsedValue := setMetaValueRecursive(key, value, previousMeta, jsonValue)
	previousMeta[key] = parsedValue

	resultJSON, err := json.Marshal(previousMeta)
	if err != nil {
		return err
	}
	err = writeFile(metaFilePath, resultJSON, 0666)
	if err != nil {
		return err
	}
	return nil
}

// setMetaValueRecursive updates meta
func setMetaValueRecursive(key string, value string, previousMeta interface{}, jsonValue bool) (string, interface{}) {
	for current, char := range key {
		if string([]rune{char}) == "[" {
			nextChar := key[current+1]
			if nextChar == []byte("]")[0] {
				// Value is array
				var metaValue [1]interface{}
				key = key[0:current] + key[current+2:] // Remove bracket[] from key
				key, metaValue[0] = setMetaValueRecursive(key, value, previousMeta, jsonValue)
				return key, metaValue
			}

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
				key, metaValue[metaIndex] = setMetaValueRecursive(key, value, previousMetaMap[keyHead], jsonValue)
			} else {
				if metaIndex+1 > previousMetaValue.Len() {
					metaValue = make([]interface{}, metaIndex+1)
					key, metaValue[metaIndex] = setMetaValueRecursive(key, value, nil, jsonValue)
				} else {
					metaValue = make([]interface{}, previousMetaValue.Len())
					key, metaValue[metaIndex] = setMetaValueRecursive(key, value, previousMetaValue.Index(metaIndex).Interface(), jsonValue)
				}
			}
			// Insert previous values to metaValue[] when previousMetaValue type is slice except new value
			if previousMetaValue.Kind() == reflect.Slice {
				for i := 0; i < previousMetaValue.Len(); i++ {
					if i != metaIndex {
						metaValue[i] = previousMetaValue.Index(i).Interface()
					}
				}
			}
			return key, metaValue
		} else if string([]rune{char}) == "." {
			// Value is object
			keyHead := key[0:current]   // e.g. aaa.bbb -> aaa
			childKey := key[current+1:] // e.g. aaa.bbb -> bbb
			obj := make(map[string]interface{})
			var tmpValue interface{}
			previousMetaMap := convertInterfaceToMap(previousMeta)
			if previousMetaMap[keyHead] == nil {
				childKey, tmpValue = setMetaValueRecursive(childKey, value, previousMetaMap, jsonValue)
			} else {
				// copy previous object only if it is map
				previousObj := convertInterfaceToMap(previousMetaMap[keyHead])
				if len(previousObj) != 0 {
					obj = previousObj
				}
				childKey, tmpValue = setMetaValueRecursive(childKey, value, previousMetaMap[keyHead], jsonValue)
			}
			obj[childKey] = tmpValue
			return keyHead, obj
		}
	}
	if jsonValue {
		var objectValue interface{}
		err := json.Unmarshal([]byte(value), &objectValue)
		if err != nil {
			logrus.Panic(err)
		}
		return key, objectValue
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
func setupDir(metaSpace string, metaFile string) error {
	err := mkdirAll(metaSpace, 0777)
	if err != nil {
		return err
	}
	err = writeFile(metaSpace+"/"+metaFile+".json", []byte("{}"), 0666)
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

	// Set to defaults in case not all commands alter these variables with flags.
	var metaSpace string = "/sd/meta"
	var metaFile string = "meta"
	var jsonValue bool = false
	var lastSuccessfulMetaRequest fetch.LastSuccessfulMetaRequest
	var loglevel string = logrus.GetLevel().String()

	app := cli.NewApp()
	app.Name = "meta-cli"
	app.Usage = "get or set metadata for Screwdriver build"
	app.UsageText = "meta command arguments [options]"
	app.Version = fmt.Sprintf("%v, commit %v, built at %v", version, commit, date)

	if date != "unknown" {
		// date is passed in from GoReleaser which uses RFC3339 format
		t, _ := time.Parse(time.RFC3339, date)
		date = t.Format("2006")
	}
	app.Copyright = "(c) 2017-" + date + " Yahoo Inc."

	metaSpaceFlag := cli.StringFlag{
		Name:        "meta-space",
		Usage:       "Location of meta temporarily",
		Value:       "/sd/meta",
		Destination: &metaSpace,
	}
	externalFlag := cli.StringFlag{
		Name:        "external, last-successful, e",
		Usage:       "MetaFile pipeline meta",
		Value:       "meta",
		Destination: &metaFile,
	}
	jsonValueFlag := cli.BoolFlag{
		Name:        "json-value, j",
		Usage:       "Treat value as json",
		Destination: &jsonValue,
	}
	sdTokenFlag := cli.StringFlag{
		Name:        "sd-token, t",
		Usage:       "Set the SD_TOKEN to use in SD API calls",
		EnvVar:      "SD_TOKEN",
		Destination: &lastSuccessfulMetaRequest.SdToken,
	}
	sdApiUrlFlag := cli.StringFlag{
		Name:        "sd-api-url, u",
		Usage:       "Set the SD_API_URL to use in SD API calls",
		EnvVar:      "SD_API_URL",
		Value:       "https://api.screwdriver.cd/v4/",
		Destination: &lastSuccessfulMetaRequest.SdApiUrl,
	}
	sdPipelineIdFlag := cli.Int64Flag{
		Name:        "sd-pipeline-id, p",
		Usage:       "Set the SD_PIPELINE_ID for job description",
		EnvVar:      "SD_PIPELINE_ID",
		Destination: &lastSuccessfulMetaRequest.DefaultSdPipelineId,
	}
	sdLoglevelFlag := cli.StringFlag{
		Name:        "loglevel, l",
		Usage:       "Set the loglevel",
		Value:       logrus.GetLevel().String(),
		Destination: &loglevel,
	}

	app.Flags = []cli.Flag{metaSpaceFlag, sdLoglevelFlag}
	app.Before = func(context *cli.Context) error {
		level, err := logrus.ParseLevel(loglevel)
		if err != nil {
			return err
		}
		logrus.SetLevel(level)
		return nil
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
				if valid := validateMetaKey(key); !valid {
					failureExit(errors.New("meta key validation error"))
				}
				err := getMeta(key, metaSpace, metaFile, os.Stdout, jsonValue, &lastSuccessfulMetaRequest)
				if err != nil {
					failureExit(err)
				}
				successExit()
				return nil
			},
			Flags: []cli.Flag{externalFlag, jsonValueFlag, sdTokenFlag, sdApiUrlFlag, sdPipelineIdFlag},
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
				if valid := validateMetaKey(key); !valid {
					failureExit(errors.New("meta key validation error"))
				}
				err := setMeta(key, val, metaSpace, metaFile, jsonValue)
				if err != nil {
					failureExit(err)
				}
				successExit()
				return nil
			},
			Flags: []cli.Flag{jsonValueFlag},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
