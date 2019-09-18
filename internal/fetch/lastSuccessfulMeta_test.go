package fetch

import (
	"github.com/stretchr/testify/suite"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
)

const (
	kMockHttpDir            = "../../mockHttp"
	kJobsJsonFile           = "jobs.json"
	kLastSuccessfulMetaFile = "lastSuccessfulMeta.json"
)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

type LastSuccessfulMetaSuite struct {
	suite.Suite
	JobsJson               string
	LastSuccessfulMetaJson string
}

func (s *LastSuccessfulMetaSuite) SetupSuite() {
	data, err := ioutil.ReadFile(filepath.Join(kMockHttpDir, kJobsJsonFile))
	s.Require().NoError(err)
	s.JobsJson = string(data)

	data, err = ioutil.ReadFile(filepath.Join(kMockHttpDir, kLastSuccessfulMetaFile))
	s.Require().NoError(err)
	s.LastSuccessfulMetaJson = string(data)
}

func TestLastSuccessfulMetaSuite(t *testing.T) {
	suite.Run(t, new(LastSuccessfulMetaSuite))
}

func (s *LastSuccessfulMetaSuite) TestLastSuccessfulMetaRequest_LastSuccessfulMetaURL() {
	for _, tc := range []struct {
		name     string
		request  LastSuccessfulMetaRequest
		jobID    int64
		expected string
	}{
		{
			request: LastSuccessfulMetaRequest{
				SdApiUrl: "https://api.screwdriver.ouroath.com/v4/",
			},
			jobID:    123,
			expected: "https://api.screwdriver.ouroath.com/v4/jobs/123/lastSuccessfulMeta",
		},
	} {
		s.Run(tc.name, func() {
			got := tc.request.LastSuccessfulMetaURL(tc.jobID)
			s.Assert().Equal(tc.expected, got)
		})
	}
}

func (s *LastSuccessfulMetaSuite) TestLastSuccessfulMetaRequest_JobIdFromJsonByName() {
	for _, tc := range []struct {
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
	} {
		s.Run(tc.name, func() {
			got, err := tc.request.JobIdFromJsonByName(s.JobsJson, tc.jobName)
			if tc.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Assert().Equal(tc.expected, got)
		})
	}
}

func (s *LastSuccessfulMetaSuite) TestLastSuccessfulMetaRequest_GetOrFetchJobId() {
	for _, tc := range []struct {
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
	} {
		s.Run(tc.name, func() {
			var mockHandler MockHandler
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/pipelines/1016708/jobs" &&
					req.Header.Get("Authorization") == "Bearer test-token"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), s.JobsJson)
				})
			testServer := httptest.NewServer(&mockHandler)
			defer testServer.Close()

			tc.request.SdApiUrl = testServer.URL + "/v4/"
			tc.request.SdToken = "test-token"
			tc.request.Transport = testServer.Client().Transport
			got, err := tc.request.FetchJobId(&tc.jobDescription)
			if tc.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Assert().Equal(tc.expected, got)
			mockHandler.AssertExpectations(s.T())
		})
	}
}

func (s *LastSuccessfulMetaSuite) TestLastSuccessfulMetaRequest_GetLastSuccessfulMeta() {
	for _, tc := range []struct {
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
			expected: s.LastSuccessfulMetaJson,
		},
	} {
		s.Run(tc.name, func() {
			var mockHandler MockHandler
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/pipelines/1016708/jobs" &&
					req.Header.Get("Authorization") == "Bearer test-token"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), s.JobsJson)
				})
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/jobs/392525/lastSuccessfulMeta" &&
					req.Header.Get("Authorization") == "Bearer test-token"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), s.LastSuccessfulMetaJson)
				})
			testServer := httptest.NewServer(&mockHandler)
			defer testServer.Close()

			tc.request.SdApiUrl = testServer.URL + "/v4/"
			tc.request.SdToken = "test-token"
			tc.request.Transport = testServer.Client().Transport
			got, err := tc.request.FetchLastSuccessfulMeta(&tc.jobDescription)
			if tc.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Assert().Equal(tc.expected, string(got))
			mockHandler.AssertExpectations(s.T())
		})
	}
}
