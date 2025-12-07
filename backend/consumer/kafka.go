package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/servereye/servereye-backend/config"
	"github.com/servereye/servereye-backend/storage"
	"github.com/servereye/servereye-backend/types"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type KafkaConsumer struct {
	config   config.ConsumerConfig
	reader   *kafka.Reader
	storage  storage.Storage
	logger   *logrus.Logger
	health   *HealthChecker

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	metrics *ConsumerMetrics
}

type ConsumerMetrics struct {
	MessagesProcessed int64
	MessagesFailed    int64
	LastMessageTime   time.Time
	BatchSize         int
	ProcessingTime    time.Duration
}

type HealthChecker struct {
	lastCheck time.Time
	healthy   bool
	mu        sync.RWMutex
}

func New(cfg *config.Config, storage storage.Storage, logger *logrus.Logger) (*KafkaConsumer, error) {
	consumerConfig := config.NewConsumerConfig(cfg)

	// Create reader with explicit topic configuration
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:          consumerConfig.Brokers,
		GroupID:          consumerConfig.GroupID,
		Topic:            consumerConfig.Topic,
		MinBytes:         10e3, // 10KB
		MaxBytes:         10e6, // 10MB
		CommitInterval:   consumerConfig.CommitInterval,
		StartOffset:      consumerConfig.StartOffset,
		MaxWait:          consumerConfig.BatchTimeout,
		ReadBackoffMin:   100 * time.Millisecond,
		ReadBackoffMax:   1 * time.Second,
		RebalanceTimeout: consumerConfig.RebalanceTimeout,
		WatchPartitionChanges: true,
	})

	// Create health checker
	health := &HealthChecker{}

	consumer := &KafkaConsumer{
		config:  consumerConfig,
		reader:  reader,
		storage: storage,
		logger:  logger,
		health:  health,
		metrics: &ConsumerMetrics{},
	}

	// Initialize topic validation
	if err := consumer.ensureTopics(); err != nil {
		logger.WithError(err).Error("Failed to ensure topics exist, will retry during consumption")
	}

	return consumer, nil
}

func (c *KafkaConsumer) Start(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)

	c.logger.WithFields(logrus.Fields{
		"brokers": c.config.Brokers,
		"topic":   c.config.Topic,
		"group":   c.config.GroupID,
	}).Info("Starting Kafka consumer")

	// Start health checker
	c.wg.Add(1)
	go c.healthCheckLoop()

	// Start message processing
	c.wg.Add(1)
	go c.processMessages()

	return nil
}

func (c *KafkaConsumer) processMessages() {
	defer c.wg.Done()

	batch := make([]kafka.Message, 0, c.config.BatchSize)
	ticker := time.NewTicker(c.config.BatchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			// Process remaining batch before exit
			if len(batch) > 0 {
				c.processBatch(batch)
			}
			return

		case <-ticker.C:
			// Process batch on timeout
			if len(batch) > 0 {
				c.processBatch(batch)
				batch = batch[:0] // Reset slice
			}

		default:
			// Try to fetch message with timeout
			fetchCtx, cancel := context.WithTimeout(c.ctx, 100*time.Millisecond)
			msg, err := c.reader.FetchMessage(fetchCtx)
			cancel()

			if err != nil {
				if err != context.DeadlineExceeded && err != context.Canceled {
					c.logger.WithError(err).Debug("Error fetching message")
					time.Sleep(time.Second) // Backoff on error
				}
				continue
			}

			batch = append(batch, msg)
			c.metrics.LastMessageTime = time.Now()

			// Process batch if full
			if len(batch) >= c.config.BatchSize {
				c.processBatch(batch)
				batch = batch[:0]
			}
		}
	}
}

func (c *KafkaConsumer) processBatch(batch []kafka.Message) {
	start := time.Now()
	successCount := 0

	for _, msg := range batch {
		if err := c.processMessage(msg); err != nil {
			c.logger.WithFields(logrus.Fields{
				"offset":    msg.Offset,
				"partition": msg.Partition,
				"error":     err,
			}).Error("Failed to process message")
			
			// Store in DLQ for failed messages
			if dlqErr := c.storage.StoreDLQMessage(c.ctx, c.config.Topic, int(msg.Partition), msg.Offset, msg.Value, err.Error()); dlqErr != nil {
				c.logger.WithError(dlqErr).Error("Failed to store message in DLQ")
			}
			
			c.metrics.MessagesFailed++
		} else {
			successCount++
			c.metrics.MessagesProcessed++
		}
	}

	// Commit successful messages
	if successCount > 0 {
		if err := c.reader.CommitMessages(c.ctx, batch...); err != nil {
			c.logger.WithError(err).Error("Failed to commit messages")
		}
	}

	c.metrics.BatchSize = len(batch)
	c.metrics.ProcessingTime = time.Since(start)

	c.logger.WithFields(logrus.Fields{
		"batch_size":     len(batch),
		"success_count":  successCount,
		"processing_ms":  c.metrics.ProcessingTime.Milliseconds(),
	}).Debug("Processed batch")
}

func (c *KafkaConsumer) processMessage(msg kafka.Message) error {
	var metric types.Metric
	if err := json.Unmarshal(msg.Value, &metric); err != nil {
		return fmt.Errorf("failed to unmarshal metric: %w", err)
	}

	// Validate metric
	if metric.ServerID == "" || metric.Type == "" {
		return fmt.Errorf("invalid metric: missing server_id or type")
	}

	// Store metric
	if err := c.storage.StoreMetric(c.ctx, &metric); err != nil {
		return fmt.Errorf("failed to store metric: %w", err)
	}

	return nil
}

func (c *KafkaConsumer) ensureTopics() error {
	// Create Kafka admin client
	conn, err := kafka.Dial("tcp", c.config.Brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	defer conn.Close()

	// Get controller
	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", controller.Addr)
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer controllerConn.Close()

	// Check if topic exists
	partitions, err := controllerConn.ReadPartitions(c.config.Topic)
	if err != nil {
		// Topic doesn't exist, create it
		topicConfig := kafka.TopicConfig{
			Topic:             c.config.Topic,
			NumPartitions:     3,
			ReplicationFactor: 1,
		}

		err = controllerConn.CreateTopics(topicConfig)
		if err != nil {
			return fmt.Errorf("failed to create topic: %w", err)
		}

		c.logger.WithField("topic", c.config.Topic).Info("Created topic")
	} else {
		c.logger.WithFields(logrus.Fields{
			"topic":      c.config.Topic,
			"partitions": len(partitions),
		}).Info("Topic exists")
	}

	return nil
}

func (c *KafkaConsumer) healthCheckLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.checkHealth()
		}
	}
}

func (c *KafkaConsumer) checkHealth() {
	c.health.mu.Lock()
	defer c.health.mu.Unlock()

	// Check if we can connect to Kafka
	conn, err := kafka.Dial("tcp", c.config.Brokers[0])
	if err != nil {
		c.health.healthy = false
		c.logger.WithError(err).Warn("Kafka health check failed")
		return
	}
	conn.Close()

	// Check if we received messages recently
	if time.Since(c.metrics.LastMessageTime) > 5*time.Minute {
		c.logger.Warn("No messages received in 5 minutes")
		c.health.healthy = false
		return
	}

	c.health.healthy = true
	c.health.lastCheck = time.Now()
}

func (c *KafkaConsumer) IsHealthy() bool {
	c.health.mu.RLock()
	defer c.health.mu.RUnlock()
	return c.health.healthy
}

func (c *KafkaConsumer) GetMetrics() ConsumerMetrics {
	return *c.metrics
}

func (c *KafkaConsumer) Shutdown(ctx context.Context) error {
	c.logger.Info("Shutting down Kafka consumer")
	c.cancel()

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines finished
	case <-ctx.Done():
		c.logger.Warn("Shutdown timeout, forcing exit")
	}

	// Close reader
	if err := c.reader.Close(); err != nil {
		c.logger.WithError(err).Error("Error closing Kafka reader")
		return err
	}

	c.logger.Info("Kafka consumer shutdown complete")
	return nil
}
