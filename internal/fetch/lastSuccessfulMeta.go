package fetch

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// LastSuccessfulMetaRequest describes a request for the SD lastSuccessfulMeta API call.
type LastSuccessfulMetaRequest struct {
	// SdToken is the screwdriver OAuth2 token
	SdToken string
	// SdAPIURL is the base url to the screwdriver rest API.
	SdAPIURL string

	// DefaultSdPipelineID is the default pipeline id to handle external names with just the jobName (no sd@pipelineId)
	DefaultSdPipelineID int64

	// Is the transport to use in calling the screwdriver REST apis (when nil, uses http.DefaultTransport)
	Transport http.RoundTripper
}

// GetTransport returns a non-nil transport, assigning the default when nil
func (r *LastSuccessfulMetaRequest) GetTransport() http.RoundTripper {
	if r.Transport == nil {
		r.Transport = http.DefaultTransport
	}
	return r.Transport
}

// LastSuccessfulMetaURL returns the URL for using the screwdriver lastSuccessfulMeta REST API
func (r *LastSuccessfulMetaRequest) LastSuccessfulMetaURL(jobID int64) string {
	return fmt.Sprintf("%sjobs/%d/lastSuccessfulMeta", r.SdAPIURL, jobID)
}

// JobsForPipelineURL returns the URL for using the screwdriver jobs REST API for a given pipelineID
func (r *LastSuccessfulMetaRequest) JobsForPipelineURL(piplineID int64) string {
	return fmt.Sprintf("%spipelines/%d/jobs", r.SdAPIURL, piplineID)
}

// JobForPipelineURL returns the URL for using the screwdriver jobs REST API for a given pipelineID and jobName
func (r *LastSuccessfulMetaRequest) JobForPipelineURL(piplineID int64, jobName string) string {
	return fmt.Sprintf("%spipelines/%d/jobs?jobName=%s", r.SdAPIURL, piplineID, jobName)
}

// JobIDFromJSONByName extracts the ID of the given jobName from the json string
func (r *LastSuccessfulMetaRequest) JobIDFromJSONByName(json, jobName string) (int64, error) {
	result := gjson.Get(json, fmt.Sprintf("#(name==%#v).id", jobName))
	if !result.Exists() {
		return 0, fmt.Errorf("jobName %v not found in json", jobName)
	}
	return result.Int(), nil
}

// FetchJobID fetches the job information from the given jobDescription, parses and returns the job id
func (r *LastSuccessfulMetaRequest) FetchJobID(jobDescription *JobDescription) (int64, error) {
	if jobDescription.PipelineID == 0 {
		logrus.Debugf("Defaulting pipelineId to %d", r.DefaultSdPipelineID)
		jobDescription.PipelineID = r.DefaultSdPipelineID
	}
	if jobDescription.PipelineID == 0 {
		return 0, fmt.Errorf("jobDescription does not have pipelineID %#v", jobDescription)
	}
	jobForPipelineURL := r.JobForPipelineURL(jobDescription.PipelineID, jobDescription.JobName)
	logrus.Tracef("jobForPipelineURL=%s", jobForPipelineURL)
	request, err := http.NewRequest("GET", jobForPipelineURL, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Add("Authorization", "Bearer "+r.SdToken)
	response, err := r.GetTransport().RoundTrip(request)
	if err != nil {
		return 0, err
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}
	return r.JobIDFromJSONByName(string(data), jobDescription.JobName)
}

// FetchLastSuccessfulMeta fetches the last successful meta from the given jobDescription and returns raw data
func (r *LastSuccessfulMetaRequest) FetchLastSuccessfulMeta(jobDescription *JobDescription) ([]byte, error) {
	jobID, err := r.FetchJobID(jobDescription)
	if err != nil {
		return nil, err
	}
	logrus.Tracef("jobID=%d", jobID)
	lastSuccessfulMetaURL := r.LastSuccessfulMetaURL(jobID)
	logrus.Tracef("lastSuccessfulMetaURL=%s", lastSuccessfulMetaURL)
	request, err := http.NewRequest("GET", lastSuccessfulMetaURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Authorization", "Bearer "+r.SdToken)
	response, err := r.GetTransport().RoundTrip(request)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}
