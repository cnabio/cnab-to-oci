package e2e

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"gotest.tools/icmd"
)

func startRegistry(t *testing.T) *Container {
	c := &Container{image: "registry:2", privatePort: 5000}
	c.Start(t)
	return c
}

// Container represents a docker container
type Container struct {
	image       string
	privatePort int
	container   string
}

// Start starts a new docker container on a random port
func (c *Container) Start(t *testing.T) {
	result := icmd.RunCommand("docker", "run", "--rm", "-d", "-P", c.image).Assert(t, icmd.Success)
	c.container = strings.Trim(result.Stdout(), " \r\n")
	time.Sleep(time.Second * 3)
}

// Stop terminates this container
func (c *Container) Stop(t *testing.T) {
	icmd.RunCommand("docker", "stop", c.container).Assert(t, icmd.Success)
}

// GetAddress returns the host:port this container listens on
func (c *Container) GetAddress(t *testing.T) string {
	result := icmd.RunCommand("docker", "port", c.container, strconv.Itoa(c.privatePort)).Assert(t, icmd.Success)
	return fmt.Sprintf("127.0.0.1:%v", strings.Trim(strings.Split(result.Stdout(), ":")[1], " \r\n"))
}
