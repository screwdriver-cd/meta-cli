package fetch

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	mockHTTPDir            = "../../mockHttp"
	jobsJSONFile           = "jobs.json"
	lastSuccessfulMetaFile = "lastSuccessfulMeta.json"
)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

type LastSuccessfulMetaSuite struct {
	suite.Suite
	JobsJSON               string
	LastSuccessfulMetaJSON string
}

func (s *LastSuccessfulMetaSuite) SetupSuite() {
	data, err := ioutil.ReadFile(filepath.Join(mockHTTPDir, jobsJSONFile))
	s.Require().NoError(err)
	s.JobsJSON = string(data)

	data, err = ioutil.ReadFile(filepath.Join(mockHTTPDir, lastSuccessfulMetaFile))
	s.Require().NoError(err)
	s.LastSuccessfulMetaJSON = string(data)
}

func TestLastSuccessfulMetaSuite(t *testing.T) {
	suite.Run(t, new(LastSuccessfulMetaSuite))
}

func (s *LastSuccessfulMetaSuite) TestLastSuccessfulMetaRequest_LastSuccessfulMetaURL() {
	tests := []struct {
		name     string
		request  LastSuccessfulMetaRequest
		jobID    int64
		expected string
	}{
		{
			request: LastSuccessfulMetaRequest{
				SdAPIURL: "https://api.screwdriver.ouroath.com/v4/",
			},
			jobID:    123,
			expected: "https://api.screwdriver.ouroath.com/v4/jobs/123/lastSuccessfulMeta",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := tt.request.LastSuccessfulMetaURL(tt.jobID)
			s.Assert().Equal(tt.expected, got)
		})
	}
}

func (s *LastSuccessfulMetaSuite) TestLastSuccessfulMetaRequest_JobIdFromJsonByName() {
	tests := []struct {
		name     string
		request  LastSuccessfulMetaRequest
		jobName  string
		expected int64
		wantErr  bool
	}{
		{
			name:     "job1",
			jobName:  "job1",
			expected: 392525,
		},
		{
			name:     "competing-meta-2",
			jobName:  "competing-meta-2",
			expected: 392544,
		},
		{
			name:    "missing-job",
			jobName: "missing-job",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got, err := tt.request.JobIDFromJSONByName(s.JobsJSON, tt.jobName)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Assert().Equal(tt.expected, got)
		})
	}
}

func (s *LastSuccessfulMetaSuite) TestLastSuccessfulMetaRequest_GetOrFetchJobId() {
	tests := []struct {
		name           string
		request        LastSuccessfulMetaRequest
		jobDescription JobDescription
		expected       int64
		wantErr        bool
	}{
		{
			name: "job1",
			jobDescription: JobDescription{
				PipelineID: 1016708,
				JobName:    "job1",
			},
			expected: 392525,
		},
		{
			name: "competing-meta-2",
			jobDescription: JobDescription{
				PipelineID: 1016708,
				JobName:    "competing-meta-2",
			},
			expected: 392544,
		},
		{
			name: "missing-job",
			jobDescription: JobDescription{
				PipelineID: 1016708,
				JobName:    "missing-job",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var mockHandler MockHandler
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/pipelines/1016708/jobs" &&
					req.Header.Get("Authorization") == "Bearer test-token"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), s.JobsJSON)
				})
			testServer := httptest.NewServer(&mockHandler)
			defer testServer.Close()

			tt.request.SdAPIURL = testServer.URL + "/v4/"
			tt.request.SdToken = "test-token"
			tt.request.Transport = testServer.Client().Transport
			got, err := tt.request.FetchJobID(&tt.jobDescription)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Assert().Equal(tt.expected, got)
			mockHandler.AssertExpectations(s.T())
		})
	}
}

func (s *LastSuccessfulMetaSuite) TestLastSuccessfulMetaRequest_GetLastSuccessfulMeta() {
	tests := []struct {
		name           string
		request        LastSuccessfulMetaRequest
		jobDescription JobDescription
		expected       string
		wantErr        bool
	}{
		{
			name: "job1",
			jobDescription: JobDescription{
				PipelineID: 1016708,
				JobName:    "job1",
			},
			expected: s.LastSuccessfulMetaJSON,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var mockHandler MockHandler
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/pipelines/1016708/jobs" &&
					req.Header.Get("Authorization") == "Bearer test-token"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), s.JobsJSON)
				})
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/jobs/392525/lastSuccessfulMeta" &&
					req.Header.Get("Authorization") == "Bearer test-token"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), s.LastSuccessfulMetaJSON)
				})
			testServer := httptest.NewServer(&mockHandler)
			defer testServer.Close()

			tt.request.SdAPIURL = testServer.URL + "/v4/"
			tt.request.SdToken = "test-token"
			tt.request.Transport = testServer.Client().Transport
			got, err := tt.request.FetchLastSuccessfulMeta(&tt.jobDescription)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Assert().Equal(tt.expected, string(got))
			mockHandler.AssertExpectations(s.T())
		})
	}
}
