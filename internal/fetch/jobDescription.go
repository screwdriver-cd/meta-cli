package fetch

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var jobDescriptionSDRegExp = regexp.MustCompile(`^sd@(\d+)(?::(\w+))?$`)

type JobDescription struct {
	PipelineID int64
	JobID      int64
	JobName    string
}

func ParseJobDescription(jobDescription string) (*JobDescription, error) {
	matches := jobDescriptionSDRegExp.FindStringSubmatch(jobDescription)
	if len(matches) == 0 {
		return &JobDescription{
			JobName: jobDescription,
		}, nil
	}
	id, err := strconv.ParseInt(matches[1], 10, 0)
	if err != nil {
		return nil, err
	}
	ret := &JobDescription{
		JobName: matches[2],
	}
	if ret.JobName == "" {
		ret.JobID = id
	} else {
		ret.PipelineID = id
	}
	return ret, nil
}

func (jd *JobDescription) ExternalString() (string, error) {
	var stringBuilder strings.Builder
	if jd.PipelineID == 0 {
		stringBuilder.WriteString("Missing PipelineID")
	}
	if jd.JobName == "" {
		stringBuilder.WriteString("Missing JobName")
	}
	if stringBuilder.Len() == 0 {
		return fmt.Sprintf("sd@%d:%s", jd.PipelineID, jd.JobName), nil
	}
	return "", errors.New(stringBuilder.String())
}
