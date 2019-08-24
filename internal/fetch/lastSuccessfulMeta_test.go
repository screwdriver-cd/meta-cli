package fetch

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	jobsJson = `
[
  {
    "id": 392545,
    "name": "competing-meta-join",
    "permutations": [
      {
        "annotations": {},
        "commands": [
          {
            "name": "nop",
            "command": "meta get meta"
          },
          {
            "name": "teardown-gather-meta",
            "command": "cp -r \"$(dirname \"$SD_META_PATH\")\" \"${SD_ARTIFACTS_DIR}/\""
          }
        ],
        "environment": {
          "SD_TEMPLATE_FULLNAME": "sieve/nop",
          "SD_TEMPLATE_NAME": "nop",
          "SD_TEMPLATE_NAMESPACE": "sieve",
          "SD_TEMPLATE_VERSION": "1.0.0"
        },
        "image": "docker.ouroath.com:4443/astracloud/no-op-base:latest",
        "secrets": [],
        "settings": {},
        "requires": [
          "competing-meta-1",
          "competing-meta-2"
        ]
      }
    ],
    "pipelineId": 1016709,
    "state": "ENABLED",
    "archived": false
  },
  {
    "id": 392544,
    "name": "competing-meta-2",
    "permutations": [
      {
        "annotations": {},
        "commands": [
          {
            "name": "nop",
            "command": "meta set meta.foo bar\nmeta set meta.competing-meta-2 abc\n"
          },
          {
            "name": "teardown-gather-meta",
            "command": "cp -r \"$(dirname \"$SD_META_PATH\")\" \"${SD_ARTIFACTS_DIR}/\""
          }
        ],
        "environment": {
          "SD_TEMPLATE_FULLNAME": "sieve/nop",
          "SD_TEMPLATE_NAME": "nop",
          "SD_TEMPLATE_NAMESPACE": "sieve",
          "SD_TEMPLATE_VERSION": "1.0.0"
        },
        "image": "docker.ouroath.com:4443/astracloud/no-op-base:latest",
        "secrets": [],
        "settings": {},
        "requires": [
          "~pr",
          "~commit"
        ]
      }
    ],
    "pipelineId": 1016709,
    "state": "ENABLED",
    "archived": false
  },
  {
    "id": 392543,
    "name": "competing-meta-1",
    "permutations": [
      {
        "annotations": {},
        "commands": [
          {
            "name": "nop",
            "command": "meta set meta.foo bar\nmeta set meta.competing-meta-1 abc\n"
          },
          {
            "name": "teardown-gather-meta",
            "command": "cp -r \"$(dirname \"$SD_META_PATH\")\" \"${SD_ARTIFACTS_DIR}/\""
          }
        ],
        "environment": {
          "SD_TEMPLATE_FULLNAME": "sieve/nop",
          "SD_TEMPLATE_NAME": "nop",
          "SD_TEMPLATE_NAMESPACE": "sieve",
          "SD_TEMPLATE_VERSION": "1.0.0"
        },
        "image": "docker.ouroath.com:4443/astracloud/no-op-base:latest",
        "secrets": [],
        "settings": {},
        "requires": [
          "~pr",
          "~commit"
        ]
      }
    ],
    "pipelineId": 1016709,
    "state": "ENABLED",
    "archived": false
  },
  {
    "id": 392535,
    "name": "see-if-external-propagates",
    "permutations": [
      {
        "annotations": {},
        "commands": [
          {
            "name": "nop",
            "command": "meta set meta.foo bar"
          },
          {
            "name": "teardown-gather-meta",
            "command": "cp -r \"$(dirname \"$SD_META_PATH\")\" \"${SD_ARTIFACTS_DIR}/\""
          }
        ],
        "environment": {
          "SD_TEMPLATE_FULLNAME": "sieve/nop",
          "SD_TEMPLATE_NAME": "nop",
          "SD_TEMPLATE_NAMESPACE": "sieve",
          "SD_TEMPLATE_VERSION": "1.0.0"
        },
        "image": "docker.ouroath.com:4443/astracloud/no-op-base:latest",
        "secrets": [],
        "settings": {},
        "requires": [
          "~fetch-from-pipeline1"
        ]
      }
    ],
    "pipelineId": 1016709,
    "state": "ENABLED",
    "archived": false
  },
  {
    "id": 392534,
    "name": "fetch-from-pipeline1",
    "permutations": [
      {
        "annotations": {},
        "commands": [
          {
            "name": "nop",
            "command": "curl -fs https://api.screwdriver.ouroath.com/v4/jobs/392524/lastSuccessfulMeta -H \"Authorization: Bearer ${SD_TOKEN}\" -o \"$(dirname \"$SD_META_PATH\")/sd@1016708:job1.json\"\nmeta get --external sd@1016708:job1 meta\n"
          },
          {
            "name": "teardown-gather-meta",
            "command": "cp -r \"$(dirname \"$SD_META_PATH\")\" \"${SD_ARTIFACTS_DIR}/\""
          }
        ],
        "environment": {
          "SD_TEMPLATE_FULLNAME": "sieve/nop",
          "SD_TEMPLATE_NAME": "nop",
          "SD_TEMPLATE_NAMESPACE": "sieve",
          "SD_TEMPLATE_VERSION": "1.0.0"
        },
        "image": "docker.ouroath.com:4443/astracloud/no-op-base:latest",
        "secrets": [],
        "settings": {},
        "requires": [
          "~pr",
          "~commit"
        ]
      }
    ],
    "pipelineId": 1016709,
    "state": "ENABLED",
    "archived": false
  },
  {
    "id": 392525,
    "name": "job1",
    "permutations": [
      {
        "annotations": {},
        "commands": [
          {
            "name": "nop",
            "command": "meta set meta.foo bar"
          },
          {
            "name": "teardown-gather-meta",
            "command": "cp -r \"$(dirname \"$SD_META_PATH\")\" \"${SD_ARTIFACTS_DIR}/\""
          }
        ],
        "environment": {
          "SD_TEMPLATE_FULLNAME": "sieve/nop",
          "SD_TEMPLATE_NAME": "nop",
          "SD_TEMPLATE_NAMESPACE": "sieve",
          "SD_TEMPLATE_VERSION": "1.0.0"
        },
        "image": "docker.ouroath.com:4443/astracloud/no-op-base:latest",
        "secrets": [],
        "settings": {},
        "requires": [
          "~pr",
          "~commit"
        ]
      }
    ],
    "pipelineId": 1016709,
    "state": "ENABLED",
    "archived": false
  }
]
`
	metaJson = `{"foo","bar","arr":[1,2,3]}`
)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.Called(w, r)
}

func TestLastSuccessfulMetaRequest_LastSuccessfulMetaURL(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
			got := tc.request.LastSuccessfulMetaURL(tc.jobID)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestLastSuccessfulMetaRequest_JobIdFromJsonByName(t *testing.T) {
	for _, tc := range []struct {
		name     string
		request  LastSuccessfulMetaRequest
		json     string
		jobName  string
		expected int64
		wantErr  bool
	}{
		{
			name:     "job1",
			json:     jobsJson,
			jobName:  "job1",
			expected: 392525,
		},
		{
			name:     "competing-meta-2",
			json:     jobsJson,
			jobName:  "competing-meta-2",
			expected: 392544,
		},
		{
			name:    "missing-job",
			json:    jobsJson,
			jobName: "missing-job",
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.request.JobIdFromJsonByName(tc.json, tc.jobName)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestLastSuccessfulMetaRequest_GetOrFetchJobId(t *testing.T) {
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
		t.Run(tc.name, func(t *testing.T) {
			var mockHandler MockHandler
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/pipelines/1016708/jobs" &&
					req.Header.Get("Authorization") == "Bearer test-token"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), jobsJson)
				})
			testServer := httptest.NewServer(&mockHandler)
			defer testServer.Close()

			tc.request.SdApiUrl = testServer.URL + "/v4/"
			tc.request.SdToken = "test-token"
			got, err := tc.request.FetchJobId(testServer.Client().Transport, &tc.jobDescription)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)
			mockHandler.AssertExpectations(t)
		})
	}
}

func TestLastSuccessfulMetaRequest_GetLastSuccessfulMeta(t *testing.T) {
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
			expected: metaJson,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mockHandler MockHandler
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/pipelines/1016708/jobs" &&
					req.Header.Get("Authorization") == "Bearer test-token"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), jobsJson)
				})
			mockHandler.On("ServeHTTP", mock.Anything, mock.MatchedBy(func(req *http.Request) bool {
				return req.URL.Path == "/v4/jobs/392525/lastSuccessfulMeta" &&
					req.Header.Get("Authorization") == "Bearer test-token"
			})).
				Once().
				Run(func(args mock.Arguments) {
					_, _ = io.WriteString(args.Get(0).(http.ResponseWriter), metaJson)
				})
			testServer := httptest.NewServer(&mockHandler)
			defer testServer.Close()

			tc.request.SdApiUrl = testServer.URL + "/v4/"
			tc.request.SdToken = "test-token"
			got, err := tc.request.FetchLastSuccessfulMeta(testServer.Client().Transport, &tc.jobDescription)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, string(got))
			mockHandler.AssertExpectations(t)
		})
	}
}
