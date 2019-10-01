package fetch

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var jobDescriptionSDRegExp = regexp.MustCompile(`^sd@(\d+):([\w-]+)$`)

type JobDescription struct {
	// The base name of the meta file (without the .json extension)
	MetaFile string
	// The pipeline ID for this job
	PipelineID int64
	// The name of the job
	JobName string
}

// ParseJobDescription parses the string of the form sd@123:jobName or jobName to create a new JobDescription object
func ParseJobDescription(defaultPipelineID int64, external string) (*JobDescription, error) {
	if strings.HasPrefix(external, "-") {
		return nil, fmt.Errorf(`--external "%s" appears to be a flag; not an external description`, external)
	}
	ret := &JobDescription{
		MetaFile:   external,
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

// External returns a representation of the JobDescription appropriate for use in meta get's --external flag
func (jd *JobDescription) External() string {
	return fmt.Sprintf("sd@%d:%s", jd.PipelineID, jd.JobName)
}

// MetaKey returns a valid meta key unique to this job-description, which may be used for storing metadata
func (jd *JobDescription) MetaKey() string {
	return fmt.Sprintf("sd.%d.%s", jd.PipelineID, jd.JobName)
}
