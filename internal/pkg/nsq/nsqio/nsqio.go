package nsqio

import (
	"context"
	"errors"
	"time"

	nsqio "github.com/nsqio/go-nsq"
)

// Config of nsqio
type Config struct {
	Hostname    string
	Lookupd     LookupdConfig
	Timeout     TimeoutConfig
	Queue       QueueConfig
	Compression CompressionConfig
}

// TimeoutConfig for timeout configuration
type TimeoutConfig struct {
	Dial           time.Duration `toml:"dial" yaml:"dial"`
	Read           time.Duration `toml:"read" yaml:"read"`
	Write          time.Duration `toml:"write" yaml:"write"`
	MessageTimeout time.Duration `toml:"message" yaml:"message"`
}

// LookupdConfig for lookupd configuration
type LookupdConfig struct {
	PoolInterval time.Duration `toml:"pool_interval" yaml:"pool_interval"`
	PollJitter   float64       `toml:"pool_jitter" yaml:"pool_jitter"`
}

// QueueConfig for message configuration
type QueueConfig struct {
	MaxInFlight         int           `toml:"max_in_flight" yaml:"max_in_flight"`
	MsgTimeout          time.Duration `toml:"message_timeout" yaml:"message_timeout"`
	MaxRequeueDelay     time.Duration `toml:"max_requeue_delay" yaml:"max_requeue_delay"`
	DefaultRequeueDelay time.Duration `toml:"default_requeue_delay" yaml:"default_requeue_delay"`
}

// CompressionConfig to support compression
type CompressionConfig struct {
	Deflate      bool `toml:"deflate" yaml:"deflate"`
	DeflateLevel int  `toml:"deflate_level" yaml:"deflate_level"`
	Snappy       bool `toml:"snappy" yaml:"snappy"`
}

func newConfig(conf Config) (*nsqio.Config, error) {
	cfg := nsqio.NewConfig()

	// basic
	cfg.Hostname = conf.Hostname
	// queue
	cfg.MaxInFlight = conf.Queue.MaxInFlight
	cfg.MsgTimeout = conf.Queue.MsgTimeout
	cfg.MaxRequeueDelay = conf.Queue.MaxRequeueDelay
	cfg.DefaultRequeueDelay = conf.Queue.DefaultRequeueDelay
	// timeout
	cfg.DialTimeout = conf.Timeout.Dial
	cfg.ReadTimeout = conf.Timeout.Read
	cfg.WriteTimeout = conf.Timeout.Write
	cfg.MsgTimeout = conf.Timeout.MessageTimeout
	// lookupd config
	cfg.LookupdPollInterval = conf.Lookupd.PoolInterval
	cfg.LookupdPollJitter = conf.Lookupd.PollJitter
	// compression
	cfg.Deflate = conf.Compression.Deflate
	cfg.DeflateLevel = conf.Compression.DeflateLevel
	cfg.Snappy = conf.Compression.Snappy

	return cfg, cfg.Validate()
}

// ProducerConfig struct
type ProducerConfig struct {
	Hostname    string
	Address     string
	Compression CompressionConfig
	Timeout     TimeoutConfig
}

// NSQProducer backend
type NSQProducer struct {
	producer *nsqio.Producer
}

// NewProducer return a new producer
func NewProducer(ctx context.Context, config ProducerConfig) (*NSQProducer, error) {
	conf := Config{
		Hostname:    config.Hostname,
		Timeout:     config.Timeout,
		Compression: config.Compression,
	}
	nsqConf, err := newConfig(conf)
	if err != nil {
		return nil, err
	}

	p, err := nsqio.NewProducer(config.Address, nsqConf)
	if err != nil {
		return nil, err
	}

	prod := NSQProducer{
		producer: p,
	}
	return &prod, nil
}

// Ping the nsqd of producer
func (np *NSQProducer) Ping() error {
	return np.producer.Ping()
}

// Publish to nsqd
func (np *NSQProducer) Publish(topic string, body []byte) error {
	return np.producer.Publish(topic, body)
}

// MultiPublish to nsqd
func (np *NSQProducer) MultiPublish(topic string, body [][]byte) error {
	return np.producer.MultiPublish(topic, body)
}

// Stop the nsq producer
func (np *NSQProducer) Stop() {
	np.Stop()
}

// ConsumerConfig for nsq consumer
type ConsumerConfig struct {
	Hostname         string
	LookupdAddresses []string
	Topic            string
	Channel          string
	Concurrency      int
	Lookupd          LookupdConfig
	Timeout          TimeoutConfig
	Queue            QueueConfig
	Compression      CompressionConfig
}

// Validate consumer configuration
func (cf *ConsumerConfig) Validate() error {
	if len(cf.LookupdAddresses) == 0 {
		return errors.New("consumer_config: lookupd address cannot be empty")
	}
	if cf.Topic == "" {
		return errors.New("consumer_config: topic cannot be empty")
	}
	if cf.Channel == "" {
		return errors.New("consumer_config: channel cannot be empty")
	}
	// set default concurrency to 1
	if cf.Concurrency == 0 {
		cf.Concurrency = 1
	}
	return nil
}

// NSQConsumer backend
type NSQConsumer struct {
	consumer *nsqio.Consumer
	config   ConsumerConfig
}

// NewConsumer for nsq
func NewConsumer(ctx context.Context, config ConsumerConfig) (*NSQConsumer, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	conf := Config{
		Hostname:    config.Hostname,
		Lookupd:     config.Lookupd,
		Timeout:     config.Timeout,
		Queue:       config.Queue,
		Compression: config.Compression,
	}

	nsqioConfig, err := newConfig(conf)
	if err != nil {
		return nil, err
	}
	con, err := nsqio.NewConsumer(config.Topic, config.Channel, nsqioConfig)
	if err != nil {
		return nil, err
	}

	c := NSQConsumer{
		consumer: con,
		config:   config,
	}
	return &c, nil
}

// Topic return the topic of consumer
func (c *NSQConsumer) Topic() string {
	return c.config.Topic
}

// Channel return the channel of consumer
func (c *NSQConsumer) Channel() string {
	return c.config.Channel
}

// ConnectToNSQLookupds connecting to several nsq lookupd
func (c *NSQConsumer) ConnectToNSQLookupds(addresses []string) error {
	return c.consumer.ConnectToNSQLookupds(addresses)
}

// AddHandler to nsq
func (c *NSQConsumer) AddHandler(handler nsqio.Handler) {
	c.consumer.AddHandler(handler)
}

// AddConcurrentHandlers add concurrent handler to nsq
func (c *NSQConsumer) AddConcurrentHandlers(handler nsqio.Handler, concurrency int) {
	c.consumer.AddConcurrentHandlers(handler, concurrency)
}

// Stop nsq consumer
func (c *NSQConsumer) Stop() {
	c.consumer.Stop()
}

// Concurrency return the number of concurrency in the config
func (c *NSQConsumer) Concurrency() int {
	return c.config.Concurrency
}

// ChangeMaxInFlight will change max in flight number in nsq consumer
func (c *NSQConsumer) ChangeMaxInFlight(n int) {
	c.consumer.ChangeMaxInFlight(n)
}