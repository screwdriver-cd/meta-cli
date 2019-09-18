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
		s.Run(tc.jobDescription, func() {
			got, err := ParseJobDescription(tc.defaultPipelineID, tc.jobDescription)
			if tc.wantErr {
				s.Require().Error(err, tc.jobDescription)
				return
			}
			s.Require().NoError(err, tc.jobDescription)
			s.Assert().Equal(tc.want, got, tc.jobDescription)
		})
	}
}

func (s *JobDescriptionSuite) TestJobDescription_External() {
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
		s.Run(tc.name, func() {
			got := tc.jobDescription.External()
			s.Assert().Equal(tc.name, got)
		})
	}
}

func (s *JobDescriptionSuite) TestJobDescription_MetaKey() {
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
		s.Run(tc.name, func() {
			got := tc.jobDescription.MetaKey()
			s.Assert().Equal(tc.name, got)
		})
	}
}
