package fetch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseJobDescription(t *testing.T) {
	for _, tc := range []struct {
		jobDescription    string
		defaultPipelineID int64
		want              *JobDescription
		wantErr           bool
	}{
		{
			jobDescription:    `sd@123:myName`,
			defaultPipelineID: 999,
			want: &JobDescription{
				MetaFile:   `sd@123:myName`,
				PipelineID: 123,
				JobName:    "myName",
			},
		},
		{
			jobDescription:    `myName`,
			defaultPipelineID: 123,
			want: &JobDescription{
				MetaFile:   `myName`,
				PipelineID: 123,
				JobName:    "myName",
			},
		},
	} {
		t.Run(tc.jobDescription, func(t *testing.T) {
			got, err := ParseJobDescription(tc.defaultPipelineID, tc.jobDescription)
			if tc.wantErr {
				require.Error(t, err, tc.jobDescription)
				return
			}
			require.NoError(t, err, tc.jobDescription)
			assert.Equal(t, tc.want, got, tc.jobDescription)
		})
	}
}

func TestJobDescription_External(t *testing.T) {
	for _, tc := range []struct {
		name           string
		jobDescription JobDescription
	}{
		{
			name: "sd@123:fooBar",
			jobDescription: JobDescription{
				PipelineID: 123,
				JobName:    "fooBar",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.jobDescription.External()
			assert.Equal(t, tc.name, got)
		})
	}
}

func TestJobDescription_MetaKey(t *testing.T) {
	for _, tc := range []struct {
		name           string
		jobDescription JobDescription
	}{
		{
			name: "sd.123.fooBar",
			jobDescription: JobDescription{
				PipelineID: 123,
				JobName:    "fooBar",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.jobDescription.MetaKey()
			assert.Equal(t, tc.name, got)
		})
	}
}
