//go:generate go run github.com/golang/mock/mockgen -source container.go -package builder -destination container_mock.go DockerEngineClient

package builder

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	networktypes "github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// DockerEngineClient ...
type DockerEngineClient interface {
	ContainerCreate(ctx context.Context, config *containertypes.Config, hostConfig *containertypes.HostConfig, networkingConfig *networktypes.NetworkingConfig, platform *specs.Platform, containerName string) (containertypes.ContainerCreateCreatedBody, error)
	ContainerStart(ctx context.Context, containerName string, options types.ContainerStartOptions) error
	ContainerLogs(ctx context.Context, containerName string, options types.ContainerLogsOptions) (io.ReadCloser, error)
	ContainerWait(ctx context.Context, containerName string, condition containertypes.WaitCondition) (<-chan containertypes.ContainerWaitOKBody, <-chan error)
	ContainerRemove(ctx context.Context, container string, options types.ContainerRemoveOptions) error
	ContainerExecAttach(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error)
	ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error)
	ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error
	ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error)
	CopyToContainer(ctx context.Context, container, path string, content io.Reader, options types.CopyToContainerOptions) error
	ImageBuild(ctx context.Context, context io.Reader, options types.ImageBuildOptions) (types.ImageBuildResponse, error)
	ImageList(ctx context.Context, options types.ImageListOptions) ([]types.ImageSummary, error)
	ImagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error)
}
