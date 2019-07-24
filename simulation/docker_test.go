package simulation

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func TestDockerAdapterBuild(t *testing.T) {
	if !IsDockerAvailable(client.DefaultDockerHost) {
		t.Skip("could not connect to the docker daemon")
	}

	// Create a docker client
	c, err := dockerClient()
	if err != nil {
		t.Fatalf("could not create docker client: %v", err)
	}
	defer c.Close()

	imageTag := "test-docker-adapter-build:latest"

	config := DefaultDockerAdapterConfig()

	// Build based on a Dockerfile
	config.BuildContext = &DockerBuildContext{
		Directory:  "../",
		Dockerfile: "Dockerfile",
		Tag:        imageTag,
	}

	// Create docker adapter: This will build the image
	_, err = NewDockerAdapter(config)
	if err != nil {
		t.Fatalf("could not create docker adapter: %v", err)
	}

	// Cleanup image
	_, err = c.ImageRemove(context.Background(), imageTag, types.ImageRemoveOptions{})
	if err != nil {
		t.Fatalf("could not delete docker image: %v", err)
	}
}

// Create docker client
func dockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(
		client.WithHost(client.DefaultDockerHost),
		client.WithAPIVersionNegotiation(),
	)
}
