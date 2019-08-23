package fetch

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_parseJobDescription(t *testing.T) {
	for _, tc := range []struct {
		jobDescription string
		want           *JobDescription
		wantErr        bool
	}{
		{
			jobDescription: `sd@123:myName`,
			want: &JobDescription{
				PipelineID: 123,
				JobName:    "myName",
			},
		},
		{
			jobDescription: `sd@123`,
			want: &JobDescription{
				JobID: 123,
			},
		},
		{
			jobDescription: `myName`,
			want: &JobDescription{
				JobName: "myName",
			},
		},
	} {
		t.Run(tc.jobDescription, func(t *testing.T) {
			got, err := ParseJobDescription(tc.jobDescription)
			if tc.wantErr {
				require.Error(t, err, tc.jobDescription)
				return
			}
			require.NoError(t, err, tc.jobDescription)
			assert.Equal(t, tc.want, got, tc.jobDescription)
		})
	}
}

func Test_ExternalString(t *testing.T) {
	for _, tc := range []struct {
		name           string
		jobDescription JobDescription
		expected       string
		wantErr        bool
	}{
		{
			name: "missing PipelineID",
			jobDescription: JobDescription{
				JobName: "foo",
			},
			wantErr: true,
		},
		{
			name: "missing JobName",
			jobDescription: JobDescription{
				PipelineID: 123,
			},
			wantErr: true,
		},
		{
			name: "legal external",
			jobDescription: JobDescription{
				PipelineID: 123,
				JobName:    "myjob",
			},
			expected: "sd@123:myjob",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.jobDescription.ExternalString()
			if tc.wantErr {
				require.Error(t, err, tc.jobDescription)
				return
			}
			require.NoError(t, err, tc.jobDescription)
			assert.Equal(t, tc.expected, got)
		})
	}
}
