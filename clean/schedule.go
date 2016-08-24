package clean

import (
	"fmt"
	"strings"

	"github.com/docker/engine-api/client"
)

type (
	// Schedule for clean docker
	Schedule struct {
		DockerHost    string
		ExcludeImages []string
		dockerClient  *client.Client
		Tasks         map[TaskType]Job
	}
)

// Init init the Schedule
func (s *Schedule) Init() (err error) {
	if len(strings.TrimSpace(s.DockerHost)) == 0 {
		err = fmt.Errorf("The docker host must be set")
		return
	}

	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	s.dockerClient, err = client.NewClient(s.DockerHost, "v1.22", nil, defaultHeaders)
	if err != nil {
		return
	}

	return
}

// Start schedule
func (s *Schedule) Start() {
	for typ, task := range s.Tasks {
		switch typ {
		case TaskCleanNoneImage:
			task.Start()
		}
	}
}

// Stop schedule
func (s *Schedule) Stop() {
	for _, task := range s.Tasks {
		task.Stop()
	}
}
