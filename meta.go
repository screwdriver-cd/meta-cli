package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/screwdriver-cd/meta-cli/internal/fetch"
	"github.com/sirupsen/logrus"

	"gopkg.in/urfave/cli.v1"
)

const (
	defaultMetaFile  = "meta"
	defaultMetaSpace = "/sd/meta"
)

// These variables get set by the build script via the LDFLAGS
// Detail about these variables are here: https://goreleaser.com/#builds
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var metaKeyValidator = regexp.MustCompile(`^(\w+(-*\w+)*)+(((\[\]|\[(0|[1-9]\d*)\]))?(\.(\w+(-*\w+)*)+)*)*$`)
var rightBracketRegExp = regexp.MustCompile(`\[(.*?)\]`)

// MetaSpec encapsulates the parameters usually from CLI so they are more readable and shareable than positional params.
type MetaSpec struct {
	// The directory for metadata
	MetaSpace string
	// When true, do not fetch last successful external meta from external sources, which don't aren't local
	SkipFetchNonexistentExternal bool
	// The base name of the meta file (without .json extension)
	MetaFile string
	// When true, treat values (for get and set) as json objects, otherwise set is string, get is value-dependent
	JSONValue bool
	// The object describing information required to fetch metadata from external sources
	LastSuccessfulMetaRequest fetch.LastSuccessfulMetaRequest
}

// MetaFilePath returns the absolute path to the meta file.
func (m *MetaSpec) MetaFilePath() string {
	return filepath.Join(m.MetaSpace, m.MetaFile+".json")
}

// IsExternal determines whether the meta is the default or externally provided.
func (m *MetaSpec) IsExternal() bool {
	return m.MetaFile != defaultMetaFile
}

// CloneDefaultMeta returns a copy of |m| with the default meta.
func (m *MetaSpec) CloneDefaultMeta() *MetaSpec {
	ret := *m
	ret.MetaFile = defaultMetaFile
	return &ret
}

// GetExternalData gets external data from meta key, external file, or fetching from lastSuccessfulMeta
func (m *MetaSpec) GetExternalData() ([]byte, error) {
	// Get the job description of the external job for looking up or fetching
	jobDescription, err := fetch.ParseJobDescription(m.LastSuccessfulMetaRequest.DefaultSdPipelineID, m.MetaFile)
	if err != nil {
		return nil, err
	}
	logrus.Tracef("jobDescription: %#v", jobDescription)

	// First try looking up in the the local (default) external meta key
	defaultMetaSpec := m.CloneDefaultMeta()
	externalMetaKey := jobDescription.MetaKey()
	externalMeta, err := defaultMetaSpec.Get(externalMetaKey)
	if err == nil && externalMeta != "null" {
		logrus.Debugf("Found data in external meta key %s", externalMetaKey)
		return []byte(externalMeta), nil
	}

	// Get from file or fetch lastSuccessfulMeta if possible, needed and store the result in the meta key
	metaFilePath := m.MetaFilePath()
	metaData, err := ioutil.ReadFile(metaFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// If we shouldn't fetch, then return without caching in the default meta.
		if m.SkipFetchNonexistentExternal {
			logrus.Debugf("%s doesn't exist; skipping fetch", metaFilePath)
			return []byte("{}"), nil
		}
		logrus.Debugf("%s doesn't exist; setting up", metaFilePath)
		_, err = m.SetupDir()
		if err != nil {
			return nil, err
		}
		logrus.Debugf("Fetching metadata from %s", jobDescription.External())
		if metaData, err = m.LastSuccessfulMetaRequest.FetchLastSuccessfulMeta(jobDescription); err != nil {
			return nil, err
		}
	}

	// Store the result in the external meta key
	err = defaultMetaSpec.Set(externalMetaKey, string(metaData))
	if err != nil {
		return nil, err
	}
	return metaData, nil
}

// SetupDir creates the metaspace directory and writes a file with empty object.
func (m *MetaSpec) SetupDir() ([]byte, error) {
	err := os.MkdirAll(m.MetaSpace, 0777)
	if err != nil {
		return nil, err
	}
	data := []byte("{}")
	err = ioutil.WriteFile(m.MetaFilePath(), data, 0666)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// GetFileData gets the data from file, setting up file with empty json object if empty.
func (m *MetaSpec) GetFileData() ([]byte, error) {
	metaFilePath := m.MetaFilePath()
	logrus.Tracef("Reading file %v", metaFilePath)
	data, err := ioutil.ReadFile(metaFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		return m.SetupDir()
	}
	return data, nil
}

// GetData gets either external or default meta data
func (m *MetaSpec) GetData() ([]byte, error) {
	if m.IsExternal() {
		return m.GetExternalData()
	}
	return m.GetFileData()
}

// Get gets metadata for the given key
func (m *MetaSpec) Get(key string) (string, error) {
	metaJSON, err := m.GetData()
	if err != nil {
		return "", err
	}

	var metaInterface map[string]interface{}
	// for Unmarshal integer as integer, not float64
	decoder := json.NewDecoder(bytes.NewReader(metaJSON))
	decoder.UseNumber()
	err = decoder.Decode(&metaInterface)
	if err != nil {
		return "", err
	}

	_, result := fetchMetaValue(key, metaInterface)

	switch result.(type) {
	case map[string]interface{}, []interface{}:
		resultJSON, _ := json.Marshal(result)
		return fmt.Sprintf("%v", string(resultJSON)), nil
	case nil:
		return "null", nil
	default:
		if m.JSONValue {
			resultJSON, _ := json.Marshal(result)
			return fmt.Sprintf("%v", string(resultJSON)), nil
		}
		return fmt.Sprintf("%v", result), nil
	}
}

// Set sets metadata for the given key to the given value
func (m *MetaSpec) Set(key string, value string) error {
	if m.IsExternal() {
		return errors.New("can only meta set current build meta")
	}
	metaFilePath := m.MetaFilePath()
	var previousMeta map[string]interface{}

	metaJSON, err := ioutil.ReadFile(metaFilePath)
	// Not exist directory
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		_, err := m.SetupDir()
		if err != nil {
			return err
		}
		// Initialize interface if first setting meta
		previousMeta = make(map[string]interface{})
	} else {
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

	key, parsedValue := setMetaValueRecursive(key, value, previousMeta, m.JSONValue)
	previousMeta[key] = parsedValue

	resultJSON, err := json.Marshal(previousMeta)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(metaFilePath, resultJSON, 0666)
	if err != nil {
		return err
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
	metaSpec := MetaSpec{
		MetaSpace:                    defaultMetaSpace,
		SkipFetchNonexistentExternal: false,
		MetaFile:                     defaultMetaFile,
		JSONValue:                    false,
	}
	loglevel := logrus.GetLevel().String()
	var lockfile string

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
		Destination: &metaSpec.MetaSpace,
	}
	externalFlag := cli.StringFlag{
		Name:        "external, e",
		Usage:       "MetaFile pipeline meta",
		Value:       defaultMetaFile,
		Destination: &metaSpec.MetaFile,
	}
	fetchNonexistentExternalFlag := cli.BoolFlag{
		Name:        "skip-fetch, F",
		Usage:       `Used with --external to skip fetching from lastSuccessfulMeta when not triggered by external job`,
		Destination: &metaSpec.SkipFetchNonexistentExternal,
	}
	jsonValueFlag := cli.BoolFlag{
		Name: "json-value, j",
		Usage: "Treat value as json. When false, set values are treated as string; get is value-dependent" +
			"and strings are not json-escaped",
		Destination: &metaSpec.JSONValue,
	}
	sdTokenFlag := cli.StringFlag{
		Name:        "sd-token, t",
		Usage:       "Set the SD_TOKEN to use in SD API calls",
		EnvVar:      "SD_TOKEN",
		Destination: &metaSpec.LastSuccessfulMetaRequest.SdToken,
	}
	sdAPIURLFlag := cli.StringFlag{
		Name:        "sd-api-url, u",
		Usage:       "Set the SD_API_URL to use in SD API calls",
		EnvVar:      "SD_API_URL",
		Value:       "https://api.screwdriver.cd/v4/",
		Destination: &metaSpec.LastSuccessfulMetaRequest.SdAPIURL,
	}
	sdPipelineIDFlag := cli.Int64Flag{
		Name:        "sd-pipeline-id, p",
		Usage:       "Set the SD_PIPELINE_ID of the job for fetching last successful meta",
		EnvVar:      "SD_PIPELINE_ID",
		Destination: &metaSpec.LastSuccessfulMetaRequest.DefaultSdPipelineID,
	}
	sdLoglevelFlag := cli.StringFlag{
		Name:        "loglevel, l",
		Usage:       "Set the loglevel",
		Value:       logrus.GetLevel().String(),
		Destination: &loglevel,
	}
	sdLockfileFlag := cli.StringFlag{
		Name:        "lockfile",
		Usage:       "Set the lockfile location",
		Value:       "/var/run/meta.lock",
		Destination: &lockfile,
	}

	app.Flags = []cli.Flag{metaSpaceFlag, sdLoglevelFlag, sdLockfileFlag}
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
				flocker := flock.New(lockfile)
				if err := flocker.RLock(); err != nil {
					failureExit(err)
				}
				defer func() { _ = flocker.Unlock() }()

				if c.NArg() != 1 {
					logrus.Error("meta get expects exactly one argument (key)")
					return cli.ShowCommandHelp(c, "get")
				}
				key := c.Args().Get(0)
				if valid := validateMetaKey(key); !valid {
					failureExit(errors.New("meta key validation error"))
				}
				if _, err := fetch.ParseJobDescription(metaSpec.LastSuccessfulMetaRequest.DefaultSdPipelineID, metaSpec.MetaFile); metaSpec.IsExternal() && err != nil {
					failureExit(err)
				}
				value, err := metaSpec.Get(key)
				if err != nil {
					failureExit(err)
				}
				_, err = io.WriteString(os.Stdout, value)
				if err != nil {
					failureExit(err)
				}
				successExit()
				return nil
			},
			Flags: []cli.Flag{
				externalFlag, fetchNonexistentExternalFlag, jsonValueFlag, sdTokenFlag, sdAPIURLFlag, sdPipelineIDFlag,
			},
		},
		{
			Name:  "set",
			Usage: "Set a metadata with key and value",
			Action: func(c *cli.Context) error {
				flocker := flock.New(lockfile)
				if err := flocker.Lock(); err != nil {
					failureExit(err)
				}
				defer func() { _ = flocker.Unlock() }()

				if c.NArg() != 2 {
					logrus.Error("meta set expects exactly two arguments (key, value)")
					return cli.ShowCommandHelp(c, "set")
				}
				key := c.Args().Get(0)
				val := c.Args().Get(1)
				if valid := validateMetaKey(key); !valid {
					failureExit(errors.New("meta key validation error"))
				}
				err := metaSpec.Set(key, val)
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
