package stats

import (
	"context"
	"errors"
	"sync"

	"github.com/docker/docker/api/types/container"
)

type ContainerStats struct {
	Running   int `json:"running"`
	Stopped   int `json:"stopped"`
	Healthy   int `json:"healthy"`
	Unhealthy int `json:"unhealthy"`
	Total     int `json:"total"`
}

type DockerClient interface {
	ContainerInspect(ctx context.Context, containerID string) (container.InspectResponse, error)
}

func CalculateContainerStats(ctx context.Context, cli DockerClient, containers []container.Summary) (ContainerStats, error) {
	var running, stopped, healthy, unhealthy int

	var mu sync.Mutex
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)

	var aggErr error
	var aggMu sync.Mutex

	for i := range containers {
		id := containers[i].ID
		semaphore <- struct{}{}
		wg.Go(func() {
			defer func() { <-semaphore }()

			containerInspection, err := cli.ContainerInspect(ctx, id)
			stat := ContainerStats{}
			if err != nil {
				aggMu.Lock()
				aggErr = errors.Join(aggErr, err)
				aggMu.Unlock()
				return
			}
			stat = getContainerStatus(containerInspection.State)

			mu.Lock()
			running += stat.Running
			stopped += stat.Stopped
			healthy += stat.Healthy
			unhealthy += stat.Unhealthy
			mu.Unlock()
		})
	}

	wg.Wait()

	return ContainerStats{
		Running:   running,
		Stopped:   stopped,
		Healthy:   healthy,
		Unhealthy: unhealthy,
		Total:     len(containers),
	}, aggErr
}

func getContainerStatus(state *container.State) ContainerStats {
	stat := ContainerStats{}
	if state == nil {
		return stat
	}

	switch state.Status {
	case container.StateRunning:
		stat.Running++
	case container.StateExited, container.StateDead:
		stat.Stopped++
	}

	if state.Health != nil {
		switch state.Health.Status {
		case container.Healthy:
			stat.Healthy++
		case container.Unhealthy:
			stat.Unhealthy++
		}
	}

	return stat
}
