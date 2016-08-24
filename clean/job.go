package clean

import (
	"context"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/types"
	"github.com/jasonlvhit/gocron"
)

const Format = "3:04:05"

type Job interface {
	Start()
	Stop()
}

// NoneImageJob clean none image job
type NoneImageJob struct {
	s          *Schedule
	Status     Status
	stopSignal chan StopType
	StartTime  string
	StopTime   string
	force      bool // force force remove when running container with
}

// Start start job
func (j *NoneImageJob) Start() {
	log.Infoln(time.Now().Local().Format(Format), "==============>> NoneImageJob Start", j.StartTime)
	go func(j *NoneImageJob) {
		gocron.Every(1).Day().At(j.StartTime).Do(j.NoneImage)
		log.Infoln(time.Now().Local().Format(Format), "==============>>  NoneImageJob Start ", (len(strings.TrimSpace(j.StopTime)) > 0), "stopTime", j.StopTime)
		gocron.Start()

		if len(strings.TrimSpace(j.StopTime)) > 0 {
			s := gocron.NewScheduler()
			s.Every(1).Day().At(j.StopTime).Do(func() {
				log.Infoln(time.Now().Local().Format(Format), "==============>>  NoneImageJob Start stop")
				j.stop(StopByTimeEnd)
			})
			s.Start()
		}
	}(j)
	// gocron.Every(1).Day().At(j.StartTime).Do(func() {
	// 	log.Infoln("==========>>> test", time.Now().Local().Format("3:04:05PM"))
	// })
}

// func (j *NoneImageJob) stopByTime() {
// 	if len(strings.TrimSpace(j.StopTime)) > 0 {
// 		gocron.Every(1).Day().At(j.StopTime).Do(j.stop, StopByTimeEnd)
// 	}
// }

// Stop stop job
func (j *NoneImageJob) Stop() {
	log.Infoln(time.Now().Local().Format(Format), "================>>> NoneImageJob Stop ")
	// gocron.Remove(j.stop)
	j.stop(StopByInterrupt)
}

func (j *NoneImageJob) stop(typ StopType) {
	log.Infoln(time.Now().Local().Format(Format), "================>>> NoneImageJob stop ", j.Status)
	if j.Status == StatusInActived {
		j.stopSignal <- typ
		close(j.stopSignal)
	}
}

// NoneImage remove none tag image
func (j *NoneImageJob) NoneImage() {
	log.Infoln(time.Now().Local().Format(Format), "=============>> NoneImageJob NoneImage")
	j.Status = StatusInActived
	j.stopSignal = make(chan StopType, 0)

	// for i := 0; i < 30; i++ {
	// 	select {
	// 	case v := <-j.stopSignal:
	// 		log.Infoln("Received Stop clean none image command, Stop Task!", v)
	// 		j.Status = StatusNotActived
	// 		return
	// 	default:
	// 		log.Infoln("==========>>> test", time.Now().Local().Format("3:04:05PM"))
	// 		time.Sleep(3 * time.Second)
	// 	}
	// }

	// j.Status = StatusCompleted
	// return
	var (
		err           error
		docker        = j.s.dockerClient
		topImages     []types.Image
		allImages     []types.Image
		allContainers []types.Container
		excludeImages = map[string]struct{}{}
		used          = map[string]string{}
	)

	if j.s.ExcludeImages != nil && len(j.s.ExcludeImages) > 0 {
		for _, imageTag := range j.s.ExcludeImages {
			excludeImages[imageTag] = struct{}{}
		}
	}

	if topImages, err = docker.ImageList(context.Background(), types.ImageListOptions{}); err != nil {
		log.WithFields(log.Fields{"NoneImageJob": "Clean none image"}).Errorln(time.Now().Local().Format(Format), "List top images error:", err)
		return
	}

	if allImages, err = docker.ImageList(context.Background(), types.ImageListOptions{All: true}); err != nil {
		log.WithFields(log.Fields{"NoneImageJob": "Clean none image"}).Errorln(time.Now().Local().Format(Format), "List all images error:", err)
		return
	}

	imageTree := make(map[string]types.Image, len(allImages))
	for _, image := range allImages {
		imageTree[image.ID] = image
	}

	if !j.force {
		if allContainers, err = docker.ContainerList(context.Background(), types.ContainerListOptions{All: true}); err != nil {
			log.WithFields(log.Fields{"NoneImageJob": "Clean none image"}).Errorln(time.Now().Local().Format(Format), "List all container error:", err)
			return
		}

		for _, container := range allContainers {
			inspected, err := docker.ContainerInspect(context.Background(), container.ID)
			if err != nil {
				log.WithFields(log.Fields{"NoneImageJob": "Clean none image"}).Errorf("getting container info for %s: %s", container.ID, err)
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
		case v := <-j.stopSignal:
			log.Infoln("Received Stop clean none image command, Stop Task!", v)
			j.Status = StatusNotActived
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
					log.Infoln(time.Now().Local().Format(Format), "Start remove image ", image.ID)
					_, err := docker.ImageRemove(context.Background(), image.ID, types.ImageRemoveOptions{PruneChildren: true, Force: true})
					if err != nil {
						log.WithFields(log.Fields{"NoneImageJob": "Clean none image"}).Errorf("while removing %s (%s): %s", image.ID, strings.Join(image.RepoTags, ","), err)
					}
				} else {
					// several tags case, remove each by name
					for _, r := range image.RepoTags {
						_, err := docker.ImageRemove(context.Background(), r, types.ImageRemoveOptions{Force: true, PruneChildren: true})
						log.Infoln(time.Now().Local().Format(Format), "Start remove image ", r)
						if err != nil {
							log.WithFields(log.Fields{"NoneImageJob": "Clean none image"}).Errorf("while removing %s (%s): %s", r, strings.Join(image.RepoTags, ","), err)
							continue
						}
					}
				}
			}
		}
	}
	j.Status = StatusCompleted
	// close(j.stopSignal)
	return
}
