package metrics

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"kafka_exporter/internal/exporter"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/IBM/sarama"
	"github.com/aws/aws-msk-iam-sasl-signer-go/signer"
	"github.com/krallistic/kazoo-go"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	exporter.Register(
		NewKafkaExporter())
}

const (
	namespace = "kafka"
	clientID  = "kafka_exporter"
)

var (
	topicFilter  = ".*"
	topicExclude = "^$"
	groupFilter  = ".*"
	groupExclude = "^$"
	logSarama    = false
	opts         = kafkaOpts{}
	wg           = sync.WaitGroup{}
)

var (
	clusterBrokers                     *prometheus.Desc
	clusterBrokerInfo                  *prometheus.Desc
	topicPartitions                    *prometheus.Desc
	topicCurrentOffset                 *prometheus.Desc
	topicOldestOffset                  *prometheus.Desc
	topicPartitionLeader               *prometheus.Desc
	topicPartitionReplicas             *prometheus.Desc
	topicPartitionInSyncReplicas       *prometheus.Desc
	topicPartitionUsesPreferredReplica *prometheus.Desc
	topicUnderReplicatedPartition      *prometheus.Desc
	consumergroupCurrentOffset         *prometheus.Desc
	consumergroupCurrentOffsetSum      *prometheus.Desc
	consumergroupLag                   *prometheus.Desc
	consumergroupLagSum                *prometheus.Desc
	consumergroupLagZookeeper          *prometheus.Desc
	consumergroupMembers               *prometheus.Desc
)

// Exporter collects Kafka stats from the given server and exports them using
// the prometheus metrics package.
type KafkaExporter struct {
	client                  sarama.Client
	topicFilter             *regexp.Regexp
	topicExclude            *regexp.Regexp
	groupFilter             *regexp.Regexp
	groupExclude            *regexp.Regexp
	mu                      sync.Mutex
	useZooKeeperLag         bool
	zookeeperClient         *kazoo.Kazoo
	nextMetadataRefresh     time.Time
	metadataRefreshInterval time.Duration
	offsetShowAll           bool
	topicWorkers            int
	allowConcurrent         bool
	sgMutex                 sync.Mutex
	sgWaitCh                chan struct{}
	sgChans                 []chan<- prometheus.Metric
	consumerGroupFetchAll   bool
}

type kafkaOpts struct {
	uri                      []string
	useSASL                  bool
	useSASLHandshake         bool
	saslUsername             string
	saslPassword             string
	saslMechanism            string
	saslDisablePAFXFast      bool
	saslAwsRegion            string
	useTLS                   bool
	tlsServerName            string
	tlsCAFile                string
	tlsCertFile              string
	tlsKeyFile               string
	tlsInsecureSkipTLSVerify bool
	kafkaVersion             string
	useZooKeeperLag          bool
	uriZookeeper             []string
	labels                   string
	metadataRefreshInterval  string
	serviceName              string
	kerberosConfigPath       string
	realm                    string
	keyTabPath               string
	kerberosAuthType         string
	offsetShowAll            bool
	topicWorkers             int
	allowConcurrent          bool
	allowAutoTopicCreation   bool
	verbosityLogLevel        int
}

type MSKAccessTokenProvider struct {
	region string
}

func (m *MSKAccessTokenProvider) Token() (*sarama.AccessToken, error) {
	token, _, err := signer.GenerateAuthToken(context.TODO(), m.region)
	return &sarama.AccessToken{Token: token}, err
}

// CanReadCertAndKey returns true if the certificate and key files already exists,
// otherwise returns false. If lost one of cert and key, returns error.
func CanReadCertAndKey(certPath, keyPath string) (bool, error) {
	certReadable := canReadFile(certPath)
	keyReadable := canReadFile(keyPath)

	if !certReadable && !keyReadable {
		return false, nil
	}

	if !certReadable {
		return false, fmt.Errorf("error reading %s, certificate and key must be supplied as a pair", certPath)
	}

	if !keyReadable {
		return false, fmt.Errorf("error reading %s, certificate and key must be supplied as a pair", keyPath)
	}

	return true, nil
}

// If the file represented by path exists and
// readable, returns true otherwise returns false.
func canReadFile(path string) bool {
	cleanPath := filepath.Clean(path)
	f, err := os.Open(cleanPath)
	if err != nil {
		return false
	}

	defer f.Close()

	return true
}

func NewKafkaExporter() *KafkaExporter {
	init_kafka_opts()
	var zookeeperClient *kazoo.Kazoo
	config := sarama.NewConfig()
	config.ClientID = clientID
	kafkaVersion, err := sarama.ParseKafkaVersion(opts.kafkaVersion)
	if err != nil {
		log.Print("failed:", err)
		return nil
	}
	config.Version = kafkaVersion

	if opts.useSASL {
		// Convert to lowercase so that SHA512 and SHA256 is still valid
		opts.saslMechanism = strings.ToLower(opts.saslMechanism)

		saslPassword := opts.saslPassword
		if saslPassword == "" {
			saslPassword = os.Getenv("SASL_USER_PASSWORD")
		}

		switch opts.saslMechanism {
		case "scram-sha512":
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
			config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA512)
		case "scram-sha256":
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA256} }
			config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA256)
		case "gssapi":
			config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeGSSAPI)
			config.Net.SASL.GSSAPI.ServiceName = opts.serviceName
			config.Net.SASL.GSSAPI.KerberosConfigPath = opts.kerberosConfigPath
			config.Net.SASL.GSSAPI.Realm = opts.realm
			config.Net.SASL.GSSAPI.Username = opts.saslUsername
			if opts.kerberosAuthType == "keytabAuth" {
				config.Net.SASL.GSSAPI.AuthType = sarama.KRB5_KEYTAB_AUTH
				config.Net.SASL.GSSAPI.KeyTabPath = opts.keyTabPath
			} else {
				config.Net.SASL.GSSAPI.AuthType = sarama.KRB5_USER_AUTH
				config.Net.SASL.GSSAPI.Password = saslPassword
			}
			if opts.saslDisablePAFXFast {
				config.Net.SASL.GSSAPI.DisablePAFXFAST = true
			}
		case "awsiam":
			config.Net.SASL.Mechanism = sarama.SASLMechanism(sarama.SASLTypeOAuth)
			config.Net.SASL.TokenProvider = &MSKAccessTokenProvider{region: opts.saslAwsRegion}
		case "plain":
		default:
			log.Printf(`invalid sasl mechanism "%s": can only be "scram-sha256", "scram-sha512", "gssapi", "awsiam" or "plain"`,
				opts.saslMechanism)
			return nil
		}

		config.Net.SASL.Enable = true
		config.Net.SASL.Handshake = opts.useSASLHandshake

		if opts.saslUsername != "" {
			config.Net.SASL.User = opts.saslUsername
		}

		if saslPassword != "" {
			config.Net.SASL.Password = saslPassword
		}
	}

	if opts.useTLS {
		config.Net.TLS.Enable = true

		config.Net.TLS.Config = &tls.Config{
			ServerName: opts.tlsServerName,
			// #nosec G402
			InsecureSkipVerify: opts.tlsInsecureSkipTLSVerify,
		}

		if opts.tlsCAFile != "" {
			if ca, err := os.ReadFile(opts.tlsCAFile); err == nil {
				config.Net.TLS.Config.RootCAs = x509.NewCertPool()
				config.Net.TLS.Config.RootCAs.AppendCertsFromPEM(ca)
			} else {
				log.Print("failed:", err)
				return nil
			}
		}

		canReadCertAndKey, err := CanReadCertAndKey(opts.tlsCertFile, opts.tlsKeyFile)
		if err != nil {
			log.Print("error reading cert and key", err)
			return nil
		}
		if canReadCertAndKey {
			cert, err := tls.LoadX509KeyPair(opts.tlsCertFile, opts.tlsKeyFile)
			if err == nil {
				config.Net.TLS.Config.Certificates = []tls.Certificate{cert}
			} else {
				log.Print("failed:", err)
				return nil
			}
		}
	}

	if opts.useZooKeeperLag {
		zookeeperClient, err = kazoo.NewKazoo(opts.uriZookeeper, nil)
		if err != nil {
			log.Print("error connecting to zookeeper", err)
			return nil
		}
	}

	interval, err := time.ParseDuration(opts.metadataRefreshInterval)
	if err != nil {
		log.Print("Cannot parse metadata refresh interval", err)
		return nil
	}

	config.Metadata.RefreshFrequency = interval

	config.Metadata.AllowAutoTopicCreation = opts.allowAutoTopicCreation

	client, err := sarama.NewClient(opts.uri, config)

	if err != nil {
		log.Print("Error Init Kafka Client", err)
		return nil
	}

	// Init our exporter.
	return &KafkaExporter{
		client:                  client,
		topicFilter:             regexp.MustCompile(topicFilter),
		topicExclude:            regexp.MustCompile(topicExclude),
		groupFilter:             regexp.MustCompile(groupFilter),
		groupExclude:            regexp.MustCompile(groupExclude),
		useZooKeeperLag:         opts.useZooKeeperLag,
		zookeeperClient:         zookeeperClient,
		nextMetadataRefresh:     time.Now(),
		metadataRefreshInterval: interval,
		offsetShowAll:           opts.offsetShowAll,
		topicWorkers:            opts.topicWorkers,
		allowConcurrent:         opts.allowConcurrent,
		sgMutex:                 sync.Mutex{},
		sgWaitCh:                nil,
		sgChans:                 []chan<- prometheus.Metric{},
		consumerGroupFetchAll:   config.Version.IsAtLeast(sarama.V2_0_0_0),
	}
}

func (e *KafkaExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- clusterBrokers
	ch <- clusterBrokerInfo
	ch <- topicCurrentOffset
	ch <- topicOldestOffset
	ch <- topicPartitions
	ch <- topicPartitionLeader
	ch <- topicPartitionReplicas
	ch <- topicPartitionInSyncReplicas
	ch <- topicPartitionUsesPreferredReplica
	ch <- topicUnderReplicatedPartition
	ch <- consumergroupCurrentOffset
	ch <- consumergroupCurrentOffsetSum
	ch <- consumergroupLag
	ch <- consumergroupLagZookeeper
	ch <- consumergroupLagSum
	ch <- consumergroupMembers
}

func (e *KafkaExporter) Collect(ch chan<- prometheus.Metric) {
	// log.Println("run here Collect")
	if e.allowConcurrent {
		e.collect(ch)
		return
	}
	// Locking to avoid race add
	e.sgMutex.Lock()
	e.sgChans = append(e.sgChans, ch)
	// Safe to compare length since we own the Lock
	if len(e.sgChans) == 1 {
		e.sgWaitCh = make(chan struct{})
		go e.collectChans(e.sgWaitCh)
	} else {
		log.Print("concurrent calls detected, waiting for first to finish")
	}
	// Put in another variable to ensure not overwriting it in another Collect once we wait
	waiter := e.sgWaitCh
	e.sgMutex.Unlock()
	// Released lock, we have insurance that our chan will be part of the collectChan slice
	<-waiter
	// collectChan finished
	// e.collect(ch)
}

func (e *KafkaExporter) collectChans(quit chan struct{}) {
	original := make(chan prometheus.Metric)
	container := make([]prometheus.Metric, 0, 100)
	go func() {
		for metric := range original {
			container = append(container, metric)
		}
	}()
	e.collect(original)
	close(original)
	// Lock to avoid modification on the channel slice
	e.sgMutex.Lock()
	for _, ch := range e.sgChans {
		for _, metric := range container {
			ch <- metric
		}
	}
	// Reset the slice
	e.sgChans = e.sgChans[:0]
	// Notify remaining waiting Collect they can return
	close(quit)
	// Release the lock so Collect can append to the slice again
	e.sgMutex.Unlock()
}

func (e *KafkaExporter) fetchOffsetVersion() int16 {
	version := e.client.Config().Version
	if e.client.Config().Version.IsAtLeast(sarama.V2_0_0_0) {
		return 4
	} else if version.IsAtLeast(sarama.V0_10_2_0) {
		return 2
	} else if version.IsAtLeast(sarama.V0_8_2_2) {
		return 1
	}
	return 0
}

func (e *KafkaExporter) collect(ch chan<- prometheus.Metric) {
	// var wg = sync.WaitGroup{}
	ch <- prometheus.MustNewConstMetric(
		clusterBrokers, prometheus.GaugeValue, float64(len(e.client.Brokers())),
	)
	for _, b := range e.client.Brokers() {
		ch <- prometheus.MustNewConstMetric(
			clusterBrokerInfo, prometheus.GaugeValue, 1, strconv.Itoa(int(b.ID())), b.Addr(),
		)
	}

	offset := make(map[string]map[int32]int64)

	now := time.Now()

	if now.After(e.nextMetadataRefresh) {
		log.Print("Refreshing client metadata")

		if err := e.client.RefreshMetadata(); err != nil {
			log.Printf("Cannot refresh topics, using cached data: %v", err)
		}

		e.nextMetadataRefresh = now.Add(e.metadataRefreshInterval)
	}

	topics, err := e.client.Topics()
	if err != nil {
		log.Printf("Cannot get topics: %v", err)
		return
	}

	topicChannel := make(chan string)

	loopTopics := func() {
		ok := true
		for ok {
			topic, open := <-topicChannel
			ok = open
			if open {
				e.GetTopicMetrics(topic, ch, offset)
			}
		}
	}

	minx := func(x int, y int) int {
		if x < y {
			return x
		} else {
			return y
		}
	}

	N := len(topics)
	if N > 1 {
		N = minx(N/2, e.topicWorkers)
	}

	for w := 1; w <= N; w++ {
		go loopTopics()
	}

	for _, topic := range topics {
		if e.topicFilter.MatchString(topic) && !e.topicExclude.MatchString(topic) {
			wg.Add(1)
			topicChannel <- topic
		}
	}
	close(topicChannel)

	wg.Wait()

	log.Printf("Fetching consumer group metrics")
	if len(e.client.Brokers()) > 0 {
		uniqueBrokerAddresses := make(map[string]bool)
		var servers []string
		for _, broker := range e.client.Brokers() {
			normalizedAddress := strings.ToLower(broker.Addr())
			if !uniqueBrokerAddresses[normalizedAddress] {
				uniqueBrokerAddresses[normalizedAddress] = true
				servers = append(servers, broker.Addr())
			}
		}
		log.Print("", servers)
		for _, broker := range e.client.Brokers() {
			for _, server := range servers {
				if server == broker.Addr() {
					wg.Add(1)
					go e.GetConsumerGroupMetrics(broker, ch, offset)
				}
			}
		}
		wg.Wait()
	} else {
		log.Printf("No valid broker, cannot get consumer group metrics")
	}
}

func (e *KafkaExporter) GetTopicMetrics(topic string, ch chan<- prometheus.Metric, offset map[string]map[int32]int64) {
	defer wg.Done()

	if !e.topicFilter.MatchString(topic) || e.topicExclude.MatchString(topic) {
		return
	}

	partitions, err := e.client.Partitions(topic)
	if err != nil {
		log.Printf("Cannot get partitions of topic %s: %v", topic, err)
		return
	}
	ch <- prometheus.MustNewConstMetric(
		topicPartitions, prometheus.GaugeValue, float64(len(partitions)), topic,
	)
	e.mu.Lock()
	offset[topic] = make(map[int32]int64, len(partitions))
	e.mu.Unlock()
	for _, partition := range partitions {
		broker, err := e.client.Leader(topic, partition)
		if err != nil {
			log.Printf("Cannot get leader of topic %s partition %d: %v", topic, partition, err)
		} else {
			ch <- prometheus.MustNewConstMetric(
				topicPartitionLeader, prometheus.GaugeValue, float64(broker.ID()), topic, strconv.FormatInt(int64(partition), 10),
			)
		}

		currentOffset, err := e.client.GetOffset(topic, partition, sarama.OffsetNewest)
		if err != nil {
			log.Printf("Cannot get current offset of topic %s partition %d: %v", topic, partition, err)
		} else {
			e.mu.Lock()
			offset[topic][partition] = currentOffset
			e.mu.Unlock()
			ch <- prometheus.MustNewConstMetric(
				topicCurrentOffset, prometheus.GaugeValue, float64(currentOffset), topic, strconv.FormatInt(int64(partition), 10),
			)
		}

		oldestOffset, err := e.client.GetOffset(topic, partition, sarama.OffsetOldest)
		if err != nil {
			log.Printf("Cannot get oldest offset of topic %s partition %d: %v", topic, partition, err)
		} else {
			ch <- prometheus.MustNewConstMetric(
				topicOldestOffset, prometheus.GaugeValue, float64(oldestOffset), topic, strconv.FormatInt(int64(partition), 10),
			)
		}

		replicas, err := e.client.Replicas(topic, partition)
		if err != nil {
			log.Printf("Cannot get replicas of topic %s partition %d: %v", topic, partition, err)
		} else {
			ch <- prometheus.MustNewConstMetric(
				topicPartitionReplicas, prometheus.GaugeValue, float64(len(replicas)), topic, strconv.FormatInt(int64(partition), 10),
			)
		}

		inSyncReplicas, err := e.client.InSyncReplicas(topic, partition)
		if err != nil {
			log.Printf("Cannot get in-sync replicas of topic %s partition %d: %v", topic, partition, err)
		} else {
			ch <- prometheus.MustNewConstMetric(
				topicPartitionInSyncReplicas, prometheus.GaugeValue, float64(len(inSyncReplicas)), topic, strconv.FormatInt(int64(partition), 10),
			)
		}

		if broker != nil && replicas != nil && len(replicas) > 0 && broker.ID() == replicas[0] {
			ch <- prometheus.MustNewConstMetric(
				topicPartitionUsesPreferredReplica, prometheus.GaugeValue, float64(1), topic, strconv.FormatInt(int64(partition), 10),
			)
		} else {
			ch <- prometheus.MustNewConstMetric(
				topicPartitionUsesPreferredReplica, prometheus.GaugeValue, float64(0), topic, strconv.FormatInt(int64(partition), 10),
			)
		}

		if replicas != nil && inSyncReplicas != nil && len(inSyncReplicas) < len(replicas) {
			ch <- prometheus.MustNewConstMetric(
				topicUnderReplicatedPartition, prometheus.GaugeValue, float64(1), topic, strconv.FormatInt(int64(partition), 10),
			)
		} else {
			ch <- prometheus.MustNewConstMetric(
				topicUnderReplicatedPartition, prometheus.GaugeValue, float64(0), topic, strconv.FormatInt(int64(partition), 10),
			)
		}

		if e.useZooKeeperLag {
			ConsumerGroups, err := e.zookeeperClient.Consumergroups()

			if err != nil {
				log.Printf("Cannot get consumer group %v", err)
			}

			for _, group := range ConsumerGroups {
				offset, _ := group.FetchOffset(topic, partition)
				if offset > 0 {

					consumerGroupLag := currentOffset - offset
					ch <- prometheus.MustNewConstMetric(
						consumergroupLagZookeeper, prometheus.GaugeValue, float64(consumerGroupLag), group.Name, topic, strconv.FormatInt(int64(partition), 10),
					)
				}
			}
		}
	}
}

func (e *KafkaExporter) GetConsumerGroupMetrics(broker *sarama.Broker, ch chan<- prometheus.Metric, offset map[string]map[int32]int64) {
	defer wg.Done()
	if err := broker.Open(e.client.Config()); err != nil && err != sarama.ErrAlreadyConnected {
		log.Printf("Cannot connect to broker %d: %v", broker.ID(), err)
		return
	}
	defer broker.Close()

	groups, err := broker.ListGroups(&sarama.ListGroupsRequest{})
	if err != nil {
		log.Printf("Cannot get consumer group: %v", err)
		return
	}
	groupIds := make([]string, 0)
	for groupId := range groups.Groups {
		if e.groupFilter.MatchString(groupId) && !e.groupExclude.MatchString(groupId) {
			groupIds = append(groupIds, groupId)
		}
	}

	describeGroups, err := broker.DescribeGroups(&sarama.DescribeGroupsRequest{Groups: groupIds})
	if err != nil {
		log.Printf("Cannot get describe groups: %v", err)
		return
	}
	for _, group := range describeGroups.Groups {
		if group.Err != 0 {
			log.Printf("Cannot describe for the group %s with error code %d", group.GroupId, group.Err)
			continue
		}
		offsetFetchRequest := sarama.OffsetFetchRequest{ConsumerGroup: group.GroupId, Version: e.fetchOffsetVersion()}
		if e.offsetShowAll {
			for topic, partitions := range offset {
				for partition := range partitions {
					offsetFetchRequest.AddPartition(topic, partition)
				}
			}
		} else {
			for _, member := range group.Members {
				if len(member.MemberAssignment) == 0 {
					log.Printf("MemberAssignment is empty for group member: %v in group: %v", member.MemberId, group.GroupId)
					continue
				}
				assignment, err := member.GetMemberAssignment()
				if err != nil {
					log.Printf("Cannot get GetMemberAssignment of group member %v : %v", member, err)
					continue
				}
				for topic, partions := range assignment.Topics {
					for _, partition := range partions {
						offsetFetchRequest.AddPartition(topic, partition)
					}
				}
			}
		}
		ch <- prometheus.MustNewConstMetric(
			consumergroupMembers, prometheus.GaugeValue, float64(len(group.Members)), group.GroupId,
		)
		offsetFetchResponse, err := broker.FetchOffset(&offsetFetchRequest)
		if err != nil {
			log.Printf("Cannot get offset of group %s: %v", group.GroupId, err)
			continue
		}

		for topic, partitions := range offsetFetchResponse.Blocks {
			// If the topic is not consumed by that consumer group, skip it
			topicConsumed := false
			for _, offsetFetchResponseBlock := range partitions {
				// Kafka will return -1 if there is no offset associated with a topic-partition under that consumer group
				if offsetFetchResponseBlock.Offset != -1 {
					topicConsumed = true
					break
				}
			}
			if !topicConsumed {
				continue
			}

			var currentOffsetSum int64
			var lagSum int64
			for partition, offsetFetchResponseBlock := range partitions {
				err := offsetFetchResponseBlock.Err
				if err != sarama.ErrNoError {
					log.Printf("Error for  partition %d :%v", partition, err.Error())
					continue
				}
				currentOffset := offsetFetchResponseBlock.Offset
				currentOffsetSum += currentOffset
				ch <- prometheus.MustNewConstMetric(
					consumergroupCurrentOffset, prometheus.GaugeValue, float64(currentOffset), group.GroupId, topic, strconv.FormatInt(int64(partition), 10),
				)
				e.mu.Lock()
				currentPartitionOffset, currentPartitionOffsetError := e.client.GetOffset(topic, partition, sarama.OffsetNewest)
				if currentPartitionOffsetError != nil {
					log.Printf("Cannot get current offset of topic %s partition %d: %v", topic, partition, currentPartitionOffsetError)
				} else {
					var lag int64
					if offsetFetchResponseBlock.Offset == -1 {
						lag = -1
					} else {
						if offset, ok := offset[topic][partition]; ok {
							if currentPartitionOffset == -1 {
								currentPartitionOffset = offset
							}
						}
						lag = currentPartitionOffset - offsetFetchResponseBlock.Offset
						lagSum += lag
					}

					ch <- prometheus.MustNewConstMetric(
						consumergroupLag, prometheus.GaugeValue, float64(lag), group.GroupId, topic, strconv.FormatInt(int64(partition), 10),
					)
				}
				e.mu.Unlock()
			}
			ch <- prometheus.MustNewConstMetric(
				consumergroupCurrentOffsetSum, prometheus.GaugeValue, float64(currentOffsetSum), group.GroupId, topic,
			)
			ch <- prometheus.MustNewConstMetric(
				consumergroupLagSum, prometheus.GaugeValue, float64(lagSum), group.GroupId, topic,
			)
		}
	}
}

type KafkaConfig struct {
	KafkaUri     string `yaml:"kafka_uri"`
	ZookeeperUri string `yaml:"zookeeper_uri"`
}

func LoadConfig(path string) (*KafkaConfig, error) {
	cleanPath := filepath.Clean(path)
	// 限制文件扩展名
	ext := filepath.Ext(cleanPath)
	if ext != ".yaml" && ext != ".yml" && ext != "" {
		return nil, fmt.Errorf("invalid file extension: only .yaml or .yml files are allowed")
	}
	configDir := "/etc/uos-exporter"
	if !strings.HasPrefix(cleanPath, configDir) {
		return nil, fmt.Errorf("config file must be located within %s", configDir)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, err
	}

	var config KafkaConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func init_kafka_opts() {
	kafkaURI := "localhost:9092"
	zookeeperURI := "localhost:2181"

	config, err := LoadConfig("/etc/uos-exporter/kafka-exporter.yaml")
	if err != nil {
		log.Printf("Error get BIND URI %v\n", err)
	} else {
		kafkaURI = config.KafkaUri
		zookeeperURI = config.ZookeeperUri
	}

	opts = kafkaOpts{
		uri:                      []string{kafkaURI},
		useSASL:                  false,
		useSASLHandshake:         true,
		saslUsername:             "",
		saslPassword:             "",
		saslAwsRegion:            os.Getenv("AWS_REGION"),
		saslMechanism:            "",
		serviceName:              "",
		kerberosConfigPath:       "",
		realm:                    "",
		kerberosAuthType:         "",
		keyTabPath:               "",
		saslDisablePAFXFast:      false,
		useTLS:                   false,
		tlsServerName:            "",
		tlsCAFile:                "",
		tlsCertFile:              "",
		tlsKeyFile:               "",
		tlsInsecureSkipTLSVerify: false,
		kafkaVersion:             sarama.V2_0_0_0.String(),
		useZooKeeperLag:          false,
		uriZookeeper:             []string{zookeeperURI},
		labels:                   "",
		metadataRefreshInterval:  "30s",
		offsetShowAll:            true,
		allowConcurrent:          false,
		topicWorkers:             100,
		allowAutoTopicCreation:   false,
		verbosityLogLevel:        0,
	}

	labels := make(map[string]string)

	// Protect against empty labels
	if opts.labels != "" {
		for _, label := range strings.Split(opts.labels, ",") {
			splitted := strings.Split(label, "=")
			if len(splitted) >= 2 {
				labels[splitted[0]] = splitted[1]
			}
		}
	}

	if logSarama {
		sarama.Logger = log.New(os.Stdout, "[sarama] ", log.LstdFlags)
	}
	// setup(topicFilter, topicExclude, groupFilter, groupExclude, logSarama, opts, labels)
	setup(labels)
}

func setup(labels map[string]string) {
	clusterBrokers = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "brokers"),
		"Number of Brokers in the Kafka Cluster.",
		nil, labels,
	)
	clusterBrokerInfo = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "broker_info"),
		"Information about the Kafka Broker.",
		[]string{"id", "address"}, labels,
	)
	topicPartitions = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "topic", "partitions"),
		"Number of partitions for this Topic",
		[]string{"topic"}, labels,
	)
	topicCurrentOffset = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "topic", "partition_current_offset"),
		"Current Offset of a Broker at Topic/Partition",
		[]string{"topic", "partition"}, labels,
	)
	topicOldestOffset = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "topic", "partition_oldest_offset"),
		"Oldest Offset of a Broker at Topic/Partition",
		[]string{"topic", "partition"}, labels,
	)

	topicPartitionLeader = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "topic", "partition_leader"),
		"Leader Broker ID of this Topic/Partition",
		[]string{"topic", "partition"}, labels,
	)

	topicPartitionReplicas = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "topic", "partition_replicas"),
		"Number of Replicas for this Topic/Partition",
		[]string{"topic", "partition"}, labels,
	)

	topicPartitionInSyncReplicas = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "topic", "partition_in_sync_replica"),
		"Number of In-Sync Replicas for this Topic/Partition",
		[]string{"topic", "partition"}, labels,
	)

	topicPartitionUsesPreferredReplica = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "topic", "partition_leader_is_preferred"),
		"1 if Topic/Partition is using the Preferred Broker",
		[]string{"topic", "partition"}, labels,
	)

	topicUnderReplicatedPartition = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "topic", "partition_under_replicated_partition"),
		"1 if Topic/Partition is under Replicated",
		[]string{"topic", "partition"}, labels,
	)

	consumergroupCurrentOffset = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "consumergroup", "current_offset"),
		"Current Offset of a ConsumerGroup at Topic/Partition",
		[]string{"consumergroup", "topic", "partition"}, labels,
	)

	consumergroupCurrentOffsetSum = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "consumergroup", "current_offset_sum"),
		"Current Offset of a ConsumerGroup at Topic for all partitions",
		[]string{"consumergroup", "topic"}, labels,
	)

	consumergroupLag = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "consumergroup", "lag"),
		"Current Approximate Lag of a ConsumerGroup at Topic/Partition",
		[]string{"consumergroup", "topic", "partition"}, labels,
	)

	consumergroupLagZookeeper = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "consumergroupzookeeper", "lag_zookeeper"),
		"Current Approximate Lag(zookeeper) of a ConsumerGroup at Topic/Partition",
		[]string{"consumergroup", "topic", "partition"}, nil,
	)

	consumergroupLagSum = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "consumergroup", "lag_sum"),
		"Current Approximate Lag of a ConsumerGroup at Topic for all partitions",
		[]string{"consumergroup", "topic"}, labels,
	)

	consumergroupMembers = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "consumergroup", "members"),
		"Amount of members in a consumer group",
		[]string{"consumergroup"}, labels,
	)

}
// Part 2 commit for kafka_exporter/internal/metrics/kafka_exporter.go
