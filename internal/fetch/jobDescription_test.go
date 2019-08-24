package fetch

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
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
				External:   `sd@123:myName`,
				PipelineID: 123,
				JobName:    "myName",
			},
		},
		{
			jobDescription:    `myName`,
			defaultPipelineID: 123,
			want: &JobDescription{
				External:   `myName`,
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
