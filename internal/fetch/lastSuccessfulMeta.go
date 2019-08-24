package fetch

import (
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/http"
)

type LastSuccessfulMetaRequest struct {
	SdToken  string
	SdApiUrl string

	DefaultSdPipelineId int64
}

func (r *LastSuccessfulMetaRequest) LastSuccessfulMetaURL(jobID int64) string {
	return fmt.Sprintf("%sjobs/%d/lastSuccessfulMeta", r.SdApiUrl, jobID)
}

func (r *LastSuccessfulMetaRequest) JobsForPipelineURL(piplineID int64) string {
	return fmt.Sprintf("%spipelines/%d/jobs", r.SdApiUrl, piplineID)
}

func (r *LastSuccessfulMetaRequest) JobIdFromJsonByName(json string, jobName string) (int64, error) {
	result := gjson.Get(json, fmt.Sprintf("#(name==%#v).id", jobName))
	if !result.Exists() {
		return 0, fmt.Errorf("jobName %v not found in json", jobName)
	}
	return result.Int(), nil
}

func (r *LastSuccessfulMetaRequest) FetchJobId(roundTripper http.RoundTripper, jobDescription *JobDescription) (int64, error) {
	if jobDescription.PipelineID == 0 {
		jobDescription.PipelineID = r.DefaultSdPipelineId
	}
	if jobDescription.PipelineID == 0 {
		return 0, fmt.Errorf("jobDescription does not have pipelineID %#v", jobDescription)
	}
	jobsForPipelineURL := r.JobsForPipelineURL(jobDescription.PipelineID)
	request, err := http.NewRequest("GET", jobsForPipelineURL, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Add("Authorization", "Bearer "+r.SdToken)
	response, err := roundTripper.RoundTrip(request)
	if err != nil {
		return 0, err
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}
	return r.JobIdFromJsonByName(string(data), jobDescription.JobName)
}

func (r *LastSuccessfulMetaRequest) FetchLastSuccessfulMeta(roundTripper http.RoundTripper, jobDescription *JobDescription) ([]byte, error) {
	jobId, err := r.FetchJobId(http.DefaultTransport, jobDescription)
	if err != nil {
		return nil, err
	}
	lastSuccessfulMetaURL := r.LastSuccessfulMetaURL(jobId)
	request, err := http.NewRequest("GET", lastSuccessfulMetaURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Authorization", "Bearer "+r.SdToken)
	response, err := roundTripper.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}
