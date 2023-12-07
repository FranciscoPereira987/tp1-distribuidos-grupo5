package dood

import (
	"context"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

type DooD struct {
	ctx    context.Context
	cancel context.CancelFunc
	cli    *client.Client
}

func NewDockerClient(host string) (*DooD, error) {
	dood := new(DooD)
	cli, err := client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
	if err == nil {
		ctx, cancel := context.WithCancel(context.Background())
		dood.cli = cli
		dood.ctx = ctx
		dood.cancel = cancel
	}

	return dood, err
}

func (dood *DooD) Shutdown() {
	dood.cancel()
	dood.cli.Close()
}

func NewDockerClientDefault() (*DooD, error) {
	return NewDockerClient("unix:///var/run/docker.sock")
}

/*
Returns a channel throught which container names can be sent to restart them
Once the channel is closed, the context for the DooD is canceled and resources freed
*/
func (dood *DooD) StartIncoming(group *sync.WaitGroup) (chan string, chan struct{}) {
	channel := make(chan string, 1)
	closeChan := make(chan struct{})
	group.Add(1)
	go func() {
	loop:
		for {
			select {
			case name := <-channel:
				if err := dood.StartContainer(name); err != nil {
					logrus.Errorf("action: starting container %s | status: failed | reason: %s", name, err)
				}
			case <-closeChan:
				break loop
			}
		}
		logrus.Info("action: docker server | status: ending")
		dood.cancel()
		group.Done()
	}()
	return channel, closeChan
}

/*
Kills the container
*/
func (dood *DooD) Kill(container string) error {
	return dood.cli.ContainerKill(dood.ctx, container, "SIGKILL")
}

func (dood *DooD) LogDockerImages() error {
	summary, err := dood.cli.ImageList(dood.ctx, types.ImageListOptions{})
	if err == nil {
		for _, image := range summary {
			logrus.Infof("Image ID: %s | Image size: %d", image.ID, image.Size)
		}
	}

	return err
}

func (dood *DooD) StartContainer(containerName string) error {
	return dood.cli.ContainerStart(dood.ctx, containerName, types.ContainerStartOptions{})
}
