package kafka_consumer

import (
        "fmt"
        "log"
        "strings"
        "sync"

        "github.com/influxdata/telegraf"
        "github.com/influxdata/telegraf/plugins/inputs"
        "github.com/influxdata/telegraf/plugins/parsers"

        "github.com/confluentinc/confluent-kafka-go/kafka"
)

type Kafka struct {
        ConsumerGroup string
        Topics        []string
        Brokers       string
        MaxMessageLen int

        Consumer *(kafka.Consumer)
		//If SSLEnabled to true, all ssl settings will be enable.
		//Need to use SSL connection if server end have config 
		//ssl.client.auth=required	Need to fill ca,cert,key,keypwd
		//ssl.client.auth=none		Need to fill ca

        SSLEnabled bool `toml:"ssl_enabled"`
        // Path to CA file
        SSLCA string `toml:"ssl_ca"`
        // Path to host cert file
        SSLCert string `toml:"ssl_cert"`
        // Path to cert key file
        SSLKey string `toml:"ssl_key"`
        // If the key file has password, please put it here
        SSLKeypwd string `toml:"ssl_keypwd"`

        // Legacy metric buffer support
        MetricBuffer int
        // TODO remove PointBuffer, legacy support
        PointBuffer int

        Offset string
        parser parsers.Parser

        sync.Mutex

        done chan struct{}

        // keep the accumulator internally:
        acc telegraf.Accumulator

        // doNotCommitMsgs tells the parser not to call CommitUpTo on the consumer
        // this is mostly for test purposes, but there may be a use-case for it later.
        doNotCommitMsgs bool

}

var sampleConfig = `
  ## kafka servers
  brokers = "localhost:9092"
  ## topic(s) to consume
  topics = ["telegraf"]

  ## Optional SSL Config
  ssl_enabled = false
  # ssl_ca = "/etc/telegraf/ca.pem"
  # ssl_cert = "/etc/telegraf/cert.pem"
  # ssl_key = "/etc/telegraf/key.pem"
  # ssl_keypwd = ""

  ## the name of the consumer group
  consumer_group = "telegraf_metrics_consumers"
  ## Offset (must be one of the following:"beginning", "earliest", "end", "latest", "unset", "invalid", "stored")
  offset = "latest"

  ## Data format to consume.
  ## Each data format has its own unique set of configuration options, read
  ## more about them here:
  ## https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_INPUT.md
  data_format = "influx"

  ## Maximum length of a message to consume, in bytes (default 0/unlimited);
  ## larger messages are dropped
  max_message_len = 65536
`

func (k *Kafka) SampleConfig() string {
        return sampleConfig
}

func (k *Kafka) Description() string {
        return "Read metrics from Kafka topic(s)"
}

func (k *Kafka) SetParser(parser parsers.Parser) {
        k.parser = parser
}

func (k *Kafka) Start(acc telegraf.Accumulator) error {
        k.Lock()
        defer k.Unlock()

        k.acc = acc

        conf := kafka.ConfigMap{"bootstrap.servers": k.Brokers}
        conf["group.id"] = k.ConsumerGroup
        conf["session.timeout.ms"] = 6000
        conf["go.events.channel.enable"] = true
        conf["go.application.rebalance.enable"] = true

	if k.SSLEnabled == true {
		log.Printf("I! Will connect kafka with SSL configurations.")
		conf["security.protocol"] = "ssl"
		conf["ssl.ca.location"] = k.SSLCA
		conf["ssl.certificate.location"] = k.SSLCert
		conf["ssl.key.location"] = k.SSLKey
		if len(k.SSLKeypwd)>0 {
			conf["ssl.key.password"] = k.SSLKeypwd
		}
	}
        switch strings.ToLower(k.Offset) {
        case "beginning", "earliest", "end", "latest", "unset", "invalid", "stored":
                log.Printf("I! Kafka consumer offset will be set to'%s'\n",
                        k.Offset)
                conf["default.topic.config"] = kafka.ConfigMap{"auto.offset.reset": strings.ToLower(k.Offset)}
        default:
                log.Printf("I! WARNING: Kafka consumer invalid offset '%s', using 'oldest'\n",
                        k.Offset)
                conf["default.topic.config"] = kafka.ConfigMap{"auto.offset.reset": "latest"}
        }

        if k.Consumer == nil {
                c, err := kafka.NewConsumer(&conf)
                if err != nil {
                        fmt.Errorf("Failed to create consumer: %s\n", err)
                        return err
                }
                k.Consumer = c
                log.Printf("I! Created Consumer %v\n", c)
                err = k.Consumer.SubscribeTopics(k.Topics, nil)
                if err != nil {
                        fmt.Errorf("Failed to subcribe topics: %v\n", k.Topics, err)
                        return err
                }
        }

        k.done = make(chan struct{})
        // Start the kafka message reader
        go k.receiver()
        log.Printf("I! Started the kafka consumer service, brokers: %v, topics: %v\n",
                k.Brokers, k.Topics)
        return nil
}

// receiver() reads all incoming messages from the consumer, and parses them into
// influxdb metric points.
func (k *Kafka) receiver() {
        for {
                select {
                case <-k.done:
                        log.Printf("I! Done! Terminated.\n")
                        return
                case ev := <-k.Consumer.Events():
                        switch e := ev.(type) {
                        case kafka.AssignedPartitions:
                                k.Consumer.Assign(e.Partitions)
                        case kafka.RevokedPartitions:
                                k.Consumer.Unassign()
                        case *kafka.Message:
				                //TODO check max length
                                log.Printf("%% Message on %s:\n%s\n",
                                        e.TopicPartition, string(e.Value))
                                metrics, err := k.parser.Parse(e.Value)
                                if err != nil {
                                        k.acc.AddError(fmt.Errorf("Message Parse Error\nmessage: %s\nerror: %s",
                                                string(e.Value), err.Error()))
                                }
                                for _, metric := range metrics {
                                        k.acc.AddFields(metric.Name(), metric.Fields(), metric.Tags(), metric.Time())
                                }
                        case kafka.PartitionEOF:
                                fmt.Printf("%% Reached %v\n", e)
                        case kafka.Error:
                                k.acc.AddError(fmt.Errorf("%% Error: %v\n", e))
                        }
                }
        }
}

func (k *Kafka) Stop() {
        k.Lock()
        defer k.Unlock()
        close(k.done)
        if err := k.Consumer.Close(); err != nil {
                k.acc.AddError(fmt.Errorf("Error closing consumer: %s\n", err.Error()))
        }
}

func (k *Kafka) Gather(acc telegraf.Accumulator) error {
        return nil
}

func init() {
        inputs.Add("kafka_consumer", func() telegraf.Input {
                return &Kafka{}
        })
}
