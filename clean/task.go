package clean

import (
	"context"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
)

type (
	StopType string
	// TaskType 任务类型
	TaskType string
	// Status 任务状态
	Status string

	// Task 具体任务
	Task struct {
		s          *Schedule
		StartTime  *time.Time
		StopTime   *time.Time
		TaskType   TaskType
		Status     Status
		StopSignal chan struct{}
	}
)

const (
	// StopTimeEnd 时间到
	StopByTimeEnd StopType = "time"
	// StopByInterrupt 手动停止
	StopByInterrupt StopType = "interrupt"
	// TaskCleanNoneImage 删除none镜像
	TaskCleanNoneImage TaskType = "none"
	// TaskCleanPeriodImage 删除存在某一段时间的镜像
	TaskCleanPeriodImage TaskType = "period"
	// StatusNotActived 任务未激活状态
	StatusNotActived Status = "NotActived"
	// StatusInActived 任务已激活
	StatusInActived Status = "InActived"
	// StatusCompleted 任务已完成
	StatusCompleted Status = "Completed"
)

// Stop stop task
func (t *Task) Stop() {
	log.Infoln("Stop task", t.Status)
	if t.Status == StatusInActived {
		t.StopSignal <- struct{}{}
	}
}

// Start start the task
func (t *Task) Start() {
	if t.Status == StatusInActived || t.Status == StatusCompleted {
		return
	}

	stopSingal := make(chan StopType, 0)
	t.StopSignal = make(chan struct{}, 0)
	startTime := time.Now()
	if t.StartTime != nil {
		startTime = *t.StartTime
	}
	log.Infoln("====>> task will start at: ", startTime.Format(time.Kitchen), " now:", time.Now().Format(time.Kitchen))
	hour, minute, second := startTime.Clock()
	startTicker := time.NewTicker(24 * time.Hour)

	go func(stop chan struct{}, stopSingal chan StopType, stopTime *time.Time) {
		if stopTime != nil {
			log.Infoln("====>> task will stop at: ", stopTime.Format(time.Kitchen))
			hour, minute, second = stopTime.Clock()
			timer := time.NewTimer(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute + time.Duration(second)*time.Second)
			for { // multiplex two stop channal to one channal
				select {
				case <-timer.C: // time to stop Task
					log.Infoln("====>> task will stop by time end")
					stopSingal <- StopByTimeEnd
					break
				case <-stop: // received signal to stop Task
					log.Infoln("====>> task will stop by Interrupt")
					stopSingal <- StopByInterrupt
					timer.Stop()
					break
				}
			}
		} else { // Only received stop signal
			log.Infoln("====>> Only received stop signal ")
			select {
			case <-stop: // received signal to stop Task
				stopSingal <- StopByInterrupt
			}
		}
		close(stopSingal)
	}(t.StopSignal, stopSingal, t.StopTime)

	go func(startChan <-chan time.Time, stopChan chan StopType) {
		s := make(chan StopType, 0)
		for {
			select {
			case <-startChan: // time to start Task
				log.Infoln("====>> start task ====>>  ")
				t.NoneImage(false, s)
			case v := <-stopChan: // time to stop Task or received stop signal for the Task
				log.Infoln("====>> stop task ====>>  v ", v)
				s <- v
				close(s)
				return
			}
		}
	}(startTicker.C, stopSingal)
}

// NoneImage remove none tag image
// force force remove when running container with
func (t *Task) NoneImage(force bool, stop chan StopType) (err error) {
	t.Status = StatusInActived
	i := 0
	for {
		i++
		log.Infoln("NoneImage ", t.StartTime, t.StopTime, t.Status, t.TaskType, i)
		time.Sleep(1 * time.Second)
	}
	var (
		docker        = t.s.dockerClient
		topImages     []types.Image
		allImages     []types.Image
		allContainers []types.Container
		excludeImages = map[string]struct{}{}
		used          = map[string]string{}
	)

	if t.s.ExcludeImages != nil && len(t.s.ExcludeImages) > 0 {
		for _, imageTag := range t.s.ExcludeImages {
			excludeImages[imageTag] = struct{}{}
		}
	}

	if topImages, err = docker.ImageList(context.Background(), types.ImageListOptions{}); err != nil {
		return
	}

	if allImages, err = docker.ImageList(context.Background(), types.ImageListOptions{All: true}); err != nil {
		return
	}

	imageTree := make(map[string]types.Image, len(allImages))
	for _, image := range allImages {
		imageTree[image.ID] = image
	}

	if !force {
		if allContainers, err = docker.ContainerList(context.Background(), types.ContainerListOptions{All: true}); err != nil {
			return
		}

		for _, container := range allContainers {
			inspected, err := docker.ContainerInspect(context.Background(), container.ID)
			if err != nil {
				log.Errorf("getting container info for %s: %s", container.ID, err)
				continue
			}
			used[inspected.Image] = container.ID
			parent := imageTree[inspected.Image].ParentID
			for {
				if parent == "" {
					break
				}
				used[parent] = container.ID
				parent = imageTree[parent].ParentID
			}
		}
	}

	for _, image := range topImages {
		select {
		case v := <-stop:
			log.Infoln("Received Stop clean none image command, Stop Task!", v)
			t.Status = StatusNotActived
			return
		default:
			if _, ok := used[image.ID]; !ok {
				skip := false
				for _, tag := range image.RepoTags {
					if _, ok := excludeImages[tag]; ok {
						skip = true
					}

					if skip {
						break
					}
				}
				if skip {
					log.Infof("Skipping %s: %s", image.ID, strings.Join(image.RepoTags, ","))
					continue
				}

				if len(image.RepoTags) < 2 {
					// <none>:<none> case, just remove by id
					_, err := docker.ImageRemove(context.Background(), image.ID, types.ImageRemoveOptions{PruneChildren: true, Force: true})
					if err != nil {
						log.Errorf("while removing %s (%s): %s", image.ID, strings.Join(image.RepoTags, ","), err)
					}
				} else {
					// several tags case, remove each by name
					for _, r := range image.RepoTags {
						_, err := docker.ImageRemove(context.Background(), r, types.ImageRemoveOptions{Force: true, PruneChildren: true})
						if err != nil {
							log.Errorf("while removing %s (%s): %s", r, strings.Join(image.RepoTags, ","), err)
							continue
						}
					}
				}
			}
		}
	}
	t.Status = StatusCompleted
	close(t.StopSignal)
	return
}

// PeriodImage 删除存在某一段时间的镜像
func (t *Task) PeriodImage(s *Schedule) (err error) {
	return
}
