package volume

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 模拟的 JSON 输出
const mockVolumeListJSON = `[
  {
    "Name": "b51e14cdc77e26b3ad0c0da318950bf9abcfe9996b66f66aa0377ea8e43a1c7e",
    "Driver": "local",
    "Mountpoint": "/var/lib/containers/storage/volumes/b51e14cdc77e26b3ad0c0da318950bf9abcfe9996b66f66aa0377ea8e43a1c7e/_data",
    "CreatedAt": "2025-06-06T09:33:02.285512319+08:00",
    "Labels": {},
    "Scope": "local",
    "Options": {},
    "Anonymous": true,
    "MountCount": 0,
    "NeedsCopyUp": true,
    "LockNumber": 5
  },
  {
    "Name": "mydata",
    "Driver": "local",
    "Mountpoint": "/var/lib/containers/storage/volumes/mydata/_data",
    "CreatedAt": "2025-06-06T11:15:10.464592939+08:00",
    "Labels": {},
    "Scope": "local",
    "Options": {},
    "MountCount": 0,
    "NeedsCopyUp": true,
    "NeedsChown": true,
    "LockNumber": 6
  }
]`

func TestNewParser(t *testing.T) {
	parser := NewParser()
	assert.NotNil(t, parser)
}

func TestParseJSONOutput(t *testing.T) {
	parser := NewParser()
	status, err := parser.Parse([]byte(mockVolumeListJSON))

	require.NoError(t, err)
	require.NotNil(t, status)
	require.Len(t, status.Volumes, 2)

	// 测试第一个卷
	volume1 := status.Volumes[0]
	assert.Equal(t, "b51e14cdc77e26b3ad0c0da318950bf9abcfe9996b66f66aa0377ea8e43a1c7e", volume1.Name)
	assert.Equal(t, "local", volume1.Driver)
	assert.Equal(t, "/var/lib/containers/storage/volumes/b51e14cdc77e26b3ad0c0da318950bf9abcfe9996b66f66aa0377ea8e43a1c7e/_data", volume1.MountPoint)
	assert.Equal(t, "local", volume1.Scope)
	assert.True(t, volume1.Anonymous)
	assert.Equal(t, 0, volume1.MountCount)
	assert.True(t, volume1.NeedsCopyUp)
	assert.Equal(t, 5, volume1.LockNumber)
	assert.False(t, volume1.Created.IsZero())

	// 测试第二个卷
	volume2 := status.Volumes[1]
	assert.Equal(t, "mydata", volume2.Name)
	assert.Equal(t, "local", volume2.Driver)
	assert.Equal(t, "/var/lib/containers/storage/volumes/mydata/_data", volume2.MountPoint)
	assert.Equal(t, 6, volume2.LockNumber)
	assert.False(t, volume2.Created.IsZero())
}

func TestParseTextOutput(t *testing.T) {
	parser := NewParser()
	textOutput := `DRIVER     VOLUME NAME
local      b51e14cdc77e26b3ad0c0da318950bf9abcfe9996b66f66aa0377ea8e43a1c7e
local      mydata`

	status, err := parser.Parse([]byte(textOutput))
	require.NoError(t, err)
	require.Len(t, status.Volumes, 2)

	// 测试第一个卷
	volume1 := status.Volumes[0]
	assert.Equal(t, "local", volume1.Driver)
	assert.Equal(t, "b51e14cdc77e26b3ad0c0da318950bf9abcfe9996b66f66aa0377ea8e43a1c7e", volume1.Name)

	// 测试第二个卷
	volume2 := status.Volumes[1]
	assert.Equal(t, "local", volume2.Driver)
	assert.Equal(t, "mydata", volume2.Name)
}

func TestNewCollector(t *testing.T) {
	logger := logrus.New()
	timeout := 30 * time.Second

	collector := NewCollector(logger, timeout)

	assert.NotNil(t, collector)
	assert.Equal(t, logger, collector.logger)
	assert.Equal(t, timeout, collector.timeout)
	assert.NotNil(t, collector.parser)
	assert.NotNil(t, collector.infoDesc)
	assert.NotNil(t, collector.createdDesc)
}

func TestCollectorDescribe(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	ch := make(chan *prometheus.Desc, 10)
	collector.Describe(ch)
	close(ch)

	var descs []*prometheus.Desc
	for desc := range ch {
		descs = append(descs, desc)
	}

	assert.Len(t, descs, 2) // 2个指标描述符
}

func TestCollectorName(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	name := collector.Name()
	assert.Equal(t, "podman_volume", name)
}

func TestCollectInfoMetrics(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	status := &Status{
		Volumes: []Volume{
			{
				Name:       "mydata",
				Driver:     "local",
				MountPoint: "/var/lib/containers/storage/volumes/mydata/_data",
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectInfoMetrics(ch, status)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 1)
}

func TestCollectCreatedMetrics(t *testing.T) {
	logger := logrus.New()
	collector := NewCollector(logger, 30*time.Second)

	created, _ := time.Parse(time.RFC3339, "2025-06-06T11:15:10.464592939+08:00")

	status := &Status{
		Volumes: []Volume{
			{
				Name:    "mydata",
				Created: created,
			},
		},
	}

	ch := make(chan prometheus.Metric, 10)
	collector.collectCreatedMetrics(ch, status)
	close(ch)

	var metrics []prometheus.Metric
	for metric := range ch {
		metrics = append(metrics, metric)
	}

	assert.Len(t, metrics, 1)
}

func TestParseInvalidJSON(t *testing.T) {
	parser := NewParser()

	// 测试无效JSON - parser会fallback到文本模式，对于单个词会跳过（字段不足）
	status, err := parser.Parse([]byte(`invalidjson`))
	require.NoError(t, err)          // 不会返回错误，因为会fallback到文本解析
	assert.Len(t, status.Volumes, 0) // 单个词不满足至少2个字段的要求，会被跳过

	// 测试空JSON
	status, err = parser.Parse([]byte(`[]`))
	require.NoError(t, err)
	assert.Len(t, status.Volumes, 0)
}

func TestParseVolumeLine(t *testing.T) {
	parser := &defaultParser{}

	// 测试正常行
	volume, err := parser.parseVolumeLine("local      mydata")
	require.NoError(t, err)
	assert.Equal(t, "local", volume.Driver)
	assert.Equal(t, "mydata", volume.Name)

	// 测试字段不足的行
	_, err = parser.parseVolumeLine("incomplete")
	assert.Error(t, err)

	// 测试空行
	_, err = parser.parseVolumeLine("")
	assert.Error(t, err)
}

func TestParseCreatedTime(t *testing.T) {
	parser := NewParser()

	jsonWithTime := `[
		{
			"Name": "test-volume",
			"Driver": "local",
			"Mountpoint": "/var/lib/containers/storage/volumes/test-volume/_data",
			"CreatedAt": "2025-06-06T11:15:10.464592939+08:00",
			"Labels": {},
			"Scope": "local",
			"Options": {},
			"MountCount": 0,
			"NeedsCopyUp": true,
			"LockNumber": 1
		}
	]`

	status, err := parser.Parse([]byte(jsonWithTime))
	require.NoError(t, err)
	require.Len(t, status.Volumes, 1)

	volume := status.Volumes[0]
	assert.False(t, volume.Created.IsZero())
	assert.Equal(t, "test-volume", volume.Name)

	expected, _ := time.Parse(time.RFC3339, "2025-06-06T11:15:10.464592939+08:00")
	assert.Equal(t, expected, volume.Created)
}

func TestAllFieldsParsing(t *testing.T) {
	parser := NewParser()

	completeJSON := `[
		{
			"Name": "complete-volume",
			"Driver": "local",
			"Mountpoint": "/var/lib/containers/storage/volumes/complete-volume/_data",
			"CreatedAt": "2025-06-06T11:15:10.464592939+08:00",
			"Labels": {},
			"Scope": "local",
			"Options": {},
			"Anonymous": false,
			"MountCount": 2,
			"NeedsCopyUp": false,
			"LockNumber": 10
		}
	]`

	status, err := parser.Parse([]byte(completeJSON))
	require.NoError(t, err)
	require.Len(t, status.Volumes, 1)

	volume := status.Volumes[0]
	assert.Equal(t, "complete-volume", volume.Name)
	assert.Equal(t, "local", volume.Driver)
	assert.Equal(t, "/var/lib/containers/storage/volumes/complete-volume/_data", volume.MountPoint)
	assert.Equal(t, "local", volume.Scope)
	assert.False(t, volume.Anonymous)
	assert.Equal(t, 2, volume.MountCount)
	assert.False(t, volume.NeedsCopyUp)
	assert.Equal(t, 10, volume.LockNumber)
}

func TestEmptyFieldsHandling(t *testing.T) {
	parser := NewParser()

	// 测试缺少某些字段的JSON
	minimalJSON := `[
		{
			"Name": "minimal-volume",
			"Driver": "local"
		}
	]`

	status, err := parser.Parse([]byte(minimalJSON))
	require.NoError(t, err)
	require.Len(t, status.Volumes, 1)

	volume := status.Volumes[0]
	assert.Equal(t, "minimal-volume", volume.Name)
	assert.Equal(t, "local", volume.Driver)
	assert.Equal(t, "", volume.MountPoint)  // 默认为空
	assert.Equal(t, 0, volume.MountCount)   // 默认为0
	assert.True(t, volume.Created.IsZero()) // 默认为零时间
}
