package clean

import (
	"docker-clean/signal"
	"fmt"
	"os"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	// DockerUnixSock socket file for communicate with docker daemon
	DockerUnixSock = "/var/run/docker.sock"
)

var rg = regexp.MustCompile(`^([0-1]{1}\d|2[0-3]):([0-5]\d)$`)

// Clean clean the garbage in docker
func Clean(cmd *cobra.Command, args []string) (err error) {
	dockerHost, err := cmd.Flags().GetString("docker-host")
	if err != nil {
		return
	}

	excludeImages, err := cmd.Flags().GetStringSlice("exclude-images")
	if err != nil {
		return
	}

	isExist, err := CheckFileExist(DockerUnixSock)
	if err != nil {
		return
	}

	var (
		startTimeStr string
		stopTimeStr  string
	)
	if cmd.Flags().Changed("start-time") {
		startTimeStr, err = cmd.Flags().GetString("start-time")
		if err != nil {
			return err
		}
		if !rg.MatchString(startTimeStr) {
			return fmt.Errorf("Error format for the start-time: %s", startTimeStr)
		}
	}

	if cmd.Flags().Changed("stop-time") {
		stopTimeStr, err = cmd.Flags().GetString("stop-time")
		if err != nil {
			return err
		}
		if !rg.MatchString(stopTimeStr) {
			return fmt.Errorf("Error format for the stop-time: %s", startTimeStr)
		}
	}

	if isExist {
		dockerHost = "unix:///var/run/docker.sock"
	} else if len(strings.TrimSpace(dockerHost)) == 0 {
		return fmt.Errorf("Must be set one of The http endpoint for docker and /var/run/docker.sock")
	}

	schedule := &Schedule{
		DockerHost:    dockerHost,
		ExcludeImages: excludeImages,
		Tasks:         map[TaskType]Job{},
	}

	//
	// schedule.Tasks[TaskCleanNoneImage] = &Task{
	// 	s:         schedule,
	// 	StartTime: &startTime,
	// 	StopTime:  &stopTime,
	// 	Status:    StatusNotActived,
	// }

	schedule.Tasks[TaskCleanNoneImage] = &NoneImageJob{
		s:         schedule,
		StartTime: startTimeStr,
		StopTime:  stopTimeStr,
		Status:    StatusNotActived,
	}

	if err = schedule.Init(); err != nil {
		return
	}

	log.Infoln(dockerHost, isExist, schedule)
	schedule.Start()

	signal.HandleSignal(func() {
		log.Infoln("Received The close signal, The program will close")
		schedule.Stop()
	})

	return
}

// CheckFileExist returns whether the given file or directory exists or not
func CheckFileExist(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
