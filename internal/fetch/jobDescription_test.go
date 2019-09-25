package fetch

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type JobDescriptionSuite struct {
	suite.Suite
}

func TestJobDescriptionSuite(t *testing.T) {
	suite.Run(t, new(JobDescriptionSuite))
}

func (s *JobDescriptionSuite) Test_parseJobDescription() {
	tests := []struct {
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
	}

	for _, tt := range tests {
		s.Run(tt.jobDescription, func() {
			got, err := ParseJobDescription(tt.defaultPipelineID, tt.jobDescription)
			if tt.wantErr {
				s.Require().Error(err, tt.jobDescription)
				return
			}
			s.Require().NoError(err, tt.jobDescription)
			s.Assert().Equal(tt.want, got, tt.jobDescription)
		})
	}
}

func (s *JobDescriptionSuite) TestJobDescription_External() {
	tests := []struct {
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
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := tt.jobDescription.External()
			s.Assert().Equal(tt.name, got)
		})
	}
}

func (s *JobDescriptionSuite) TestJobDescription_MetaKey() {
	tests := []struct {
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
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			got := tt.jobDescription.MetaKey()
			s.Assert().Equal(tt.name, got)
		})
	}
}
