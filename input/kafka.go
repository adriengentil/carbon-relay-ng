package input

import (
	"context"
	"fmt"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/graphite-ng/carbon-relay-ng/encoding"
	log "github.com/sirupsen/logrus"
)

type Kafka struct {
	BaseInput

	topic      string
	dispatcher Dispatcher
	client     sarama.ConsumerGroup
	ctx        context.Context
	closed     chan bool
	ready      chan bool
}

func (kafka *Kafka) Name() string {
	return "kafka"
}

func (k *Kafka) Start(d Dispatcher) error {
	k.Dispatcher = d

	k.ready = make(chan bool, 0)

	go func() {
		for err := range k.client.Errors() {
			log.Errorln("kafka input error ", err)
		}
	}()
	go func(c chan bool) {
		for {
			select {
			case <-c:
				return
			default:
			}
			err := k.client.Consume(k.ctx, strings.Fields(k.topic), k)
			if err != nil {
				log.Errorln("kafka input error Consume method ", err)
			}
			k.ready = make(chan bool, 0)
		}
	}(k.closed)
	<-k.ready // Await till the consumer has been set up
	log.Infoln("Sarama consumer up and running!...")
	return nil

}
func (k *Kafka) close() {
	err := k.client.Close()
	if err != nil {
		log.Errorln("kafka input closed with errors.", err)
	} else {
		log.Infoln("kafka input closed correctly.")
	}
}

func (k *Kafka) Stop() error {
	close(k.closed)
	k.close()
	return nil
}

func NewKafka(id string, brokers []string, topic string, autoOffsetReset int64, consumerGroup string, h encoding.FormatAdapter) *Kafka {
	kafkaConfig := sarama.NewConfig()
	if id != "" {
		kafkaConfig.ClientID = id
	}

	kafkaConfig.Consumer.Return.Errors = true
	kafkaConfig.Consumer.Offsets.Initial = autoOffsetReset
	kafkaConfig.Version = sarama.V2_2_0_0

	client, err := sarama.NewConsumerGroup(brokers, consumerGroup, kafkaConfig)
	if err != nil {
		log.Fatalln("kafka input init failed", err)
	} else {
		log.Infoln("kafka input init correctly")
	}

	return &Kafka{
		BaseInput: BaseInput{handler: h, name: fmt.Sprintf("kafka[topic=%s;cg=%s;id=%s]", topic, consumerGroup, kafkaConfig.ClientID)},
		topic:     topic,
		client:    client,
		ctx:       context.Background(),
		closed:    make(chan bool),
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (k *Kafka) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(k.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (k *Kafka) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (k *Kafka) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/master/consumer_group.go#L27-L29
	for message := range claim.Messages() {
		log.Traceln("metric value:", string(message.Value))
		if err := k.handle(message.Value); err != nil {
			log.Debugf("invalid message from kafka: %#v", message)
		}
		session.MarkMessage(message, "")
	}
	return nil
}
