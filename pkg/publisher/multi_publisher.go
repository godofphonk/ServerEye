package publisher

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// MultiPublisher отправляет метрики в несколько publishers одновременно
type MultiPublisher struct {
	publishers []Publisher
	logger     *logrus.Logger
	
	// Стратегия обработки ошибок
	failureStrategy FailureStrategy
}

// FailureStrategy определяет, как обрабатывать ошибки от publishers
type FailureStrategy int

const (
	// FailIfAll - возвращает ошибку только если все publishers упали
	FailIfAll FailureStrategy = iota
	
	// FailIfAny - возвращает ошибку если хотя бы один publisher упал
	FailIfAny
	
	// FailIfPrimary - возвращает ошибку только если первый (основной) publisher упал
	FailIfPrimary
)

// NewMultiPublisher создает новый multi-publisher
func NewMultiPublisher(publishers []Publisher, strategy FailureStrategy, logger *logrus.Logger) *MultiPublisher {
	if logger == nil {
		logger = logrus.New()
	}
	
	return &MultiPublisher{
		publishers:      publishers,
		failureStrategy: strategy,
		logger:          logger,
	}
}

// Publish отправляет метрику во все publishers параллельно
func (m *MultiPublisher) Publish(ctx context.Context, metric *Metric) error {
	if len(m.publishers) == 0 {
		return fmt.Errorf("no publishers configured")
	}
	
	// Если только один publisher, отправляем напрямую
	if len(m.publishers) == 1 {
		return m.publishers[0].Publish(ctx, metric)
	}
	
	// Параллельная отправка в несколько publishers
	var wg sync.WaitGroup
	errors := make([]error, len(m.publishers))
	
	for i, pub := range m.publishers {
		wg.Add(1)
		go func(index int, publisher Publisher) {
			defer wg.Done()
			
			if err := publisher.Publish(ctx, metric); err != nil {
				errors[index] = err
				m.logger.WithFields(logrus.Fields{
					"publisher":   publisher.Name(),
					"metric_type": metric.Type,
					"server_id":   metric.ServerID,
					"error":       err,
				}).Warn("Publisher failed to send metric")
			} else {
				m.logger.WithFields(logrus.Fields{
					"publisher":   publisher.Name(),
					"metric_type": metric.Type,
					"server_id":   metric.ServerID,
				}).Debug("Metric sent successfully")
			}
		}(i, pub)
	}
	
	wg.Wait()
	
	// Анализируем ошибки согласно стратегии
	return m.evaluateErrors(errors)
}

// PublishBatch отправляет пакет метрик во все publishers
func (m *MultiPublisher) PublishBatch(ctx context.Context, metrics []*Metric) error {
	if len(m.publishers) == 0 {
		return fmt.Errorf("no publishers configured")
	}
	
	if len(metrics) == 0 {
		return nil
	}
	
	var wg sync.WaitGroup
	errors := make([]error, len(m.publishers))
	
	for i, pub := range m.publishers {
		wg.Add(1)
		go func(index int, publisher Publisher) {
			defer wg.Done()
			
			if err := publisher.PublishBatch(ctx, metrics); err != nil {
				errors[index] = err
				m.logger.WithFields(logrus.Fields{
					"publisher": publisher.Name(),
					"count":     len(metrics),
					"error":     err,
				}).Warn("Publisher failed to send batch")
			}
		}(i, pub)
	}
	
	wg.Wait()
	
	return m.evaluateErrors(errors)
}

// Close закрывает все publishers
func (m *MultiPublisher) Close() error {
	var lastErr error
	
	for _, pub := range m.publishers {
		if err := pub.Close(); err != nil {
			m.logger.WithFields(logrus.Fields{
				"publisher": pub.Name(),
				"error":     err,
			}).Error("Failed to close publisher")
			lastErr = err
		}
	}
	
	return lastErr
}

// Name возвращает имя multi-publisher
func (m *MultiPublisher) Name() string {
	names := make([]string, len(m.publishers))
	for i, pub := range m.publishers {
		names[i] = pub.Name()
	}
	return fmt.Sprintf("multi[%v]", names)
}

// evaluateErrors анализирует ошибки согласно стратегии
func (m *MultiPublisher) evaluateErrors(errors []error) error {
	switch m.failureStrategy {
	case FailIfAll:
		// Возвращаем ошибку только если ВСЕ упали
		allFailed := true
		for _, err := range errors {
			if err == nil {
				allFailed = false
				break
			}
		}
		if allFailed {
			return fmt.Errorf("all publishers failed: %v", errors)
		}
		return nil
		
	case FailIfAny:
		// Возвращаем ошибку если хотя бы один упал
		for i, err := range errors {
			if err != nil {
				return fmt.Errorf("publisher %d failed: %w", i, err)
			}
		}
		return nil
		
	case FailIfPrimary:
		// Возвращаем ошибку только если первый (основной) publisher упал
		if len(errors) > 0 && errors[0] != nil {
			return fmt.Errorf("primary publisher failed: %w", errors[0])
		}
		return nil
		
	default:
		return fmt.Errorf("unknown failure strategy: %d", m.failureStrategy)
	}
}

// GetPublishers возвращает список publishers (для тестирования)
func (m *MultiPublisher) GetPublishers() []Publisher {
	return m.publishers
}
