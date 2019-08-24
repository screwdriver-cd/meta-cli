package fetch

import (
	"regexp"
	"strconv"
)

var jobDescriptionSDRegExp = regexp.MustCompile(`^sd@(\d+):(\w+)$`)

type JobDescription struct {
	External   string
	PipelineID int64
	JobName    string
}

func ParseJobDescription(defaultPipelineID int64, external string) (*JobDescription, error) {
	ret := &JobDescription{
		External:   external,
		PipelineID: defaultPipelineID,
	}
	matches := jobDescriptionSDRegExp.FindStringSubmatch(external)
	if len(matches) == 0 {
		ret.JobName = external
		return ret, nil
	}
	var err error
	ret.PipelineID, err = strconv.ParseInt(matches[1], 10, 0)
	if err != nil {
		return nil, err
	}
	ret.JobName = matches[2]
	return ret, nil
}
