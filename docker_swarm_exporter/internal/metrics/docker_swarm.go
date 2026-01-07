package metrics

import (
	"context"
	"docker_swarm_exporter/internal/exporter"
	"docker_swarm_exporter/internal/mysql"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	cli *client.Client
	err error
)

func init() {
	exporter.Register(
		NewScrapeReplicaHost())
}

func ConnectSwarm(path string) error {
	// "tcp://10.7.33.78:2375"
	cli, err = client.NewClientWithOpts(
		client.WithHost(path),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return err
	}
	ctx := context.Background()
	_, err = cli.Ping(ctx)
	if err != nil {
		return err
	}
	return nil
}

func DisconnectSwarm() error {
	return cli.Close()
}

func listServices() ([]swarm.Service, error) {
	ctx := context.Background()
	return cli.ServiceList(ctx, types.ServiceListOptions{})
}

type ScrapeReplicaHost struct {
	instance mysql.Instance
	replicaCount
	imageVersion
}

func NewScrapeReplicaHost() *ScrapeReplicaHost {
	return &ScrapeReplicaHost{
		replicaCount: *NewReplicaCount(),
		imageVersion: *NewImageVersion(),
	}
}

func (qd ScrapeReplicaHost) Collect(ch chan<- prometheus.Metric) {
	services, err := listServices()
	if err != nil {
		logrus.Errorf("list services error: %s", err)
		return
	}
	for _, service := range services {
		serviceName := service.Spec.Name
		image := service.Spec.TaskTemplate.ContainerSpec.Image
		qd.imageVersion.Collect(ch,
			1,
			[]string{
				serviceName,
				image,
			})
		if service.Spec.Mode.Replicated != nil {
			replicas := *service.Spec.Mode.Replicated.Replicas
			qd.replicaCount.Collect(ch,
				float64(replicas),
				[]string{
					serviceName,
				})
		}

	}
}

type replicaCount struct {
	*baseMetrics
}

func NewReplicaCount() *replicaCount {
	return &replicaCount{
		NewMetrics(
			"swarm_service_desired_replicas",
			"Number of replicas requested for this service",
			[]string{
				"service_name",
			})}
}

func (qd *replicaCount) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type imageVersion struct {
	*baseMetrics
}

func NewImageVersion() *imageVersion {
	return &imageVersion{
		NewMetrics(
			"swarm_service_info",
			"Information about each service",
			[]string{
				"service_name",
				"image",
			})}
}

func (qd *imageVersion) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}
