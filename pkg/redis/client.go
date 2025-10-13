package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// Client обертка над Redis клиентом
type Client struct {
	rdb    *redis.Client
	logger *logrus.Logger
}

// Config конфигурация Redis клиента
type Config struct {
	Address  string
	Password string
	DB       int
}

// NewClient создает новый Redis клиент
func NewClient(config Config, logger *logrus.Logger) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Password: config.Password,
		DB:       config.DB,
	})

	// Проверяем соединение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("не удалось подключиться к Redis: %v", err)
	}

	logger.Info("Успешно подключились к Redis")

	return &Client{
		rdb:    rdb,
		logger: logger,
	}, nil
}

// Publish отправляет сообщение в канал
func (c *Client) Publish(ctx context.Context, channel string, message []byte) error {
	err := c.rdb.Publish(ctx, channel, message).Err()
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"channel": channel,
			"error":   err,
		}).Error("Ошибка отправки сообщения в Redis")
		return err
	}

	c.logger.WithFields(logrus.Fields{
		"channel": channel,
		"size":    len(message),
	}).Debug("Сообщение отправлено в Redis")

	return nil
}

// Subscribe подписывается на канал и возвращает канал для получения сообщений
func (c *Client) Subscribe(ctx context.Context, channel string) (<-chan []byte, error) {
	pubsub := c.rdb.Subscribe(ctx, channel)

	// Проверяем подписку
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка подписки на канал %s: %v", channel, err)
	}

	c.logger.WithField("channel", channel).Info("Подписались на канал Redis")

	// Создаем канал для сообщений
	msgChan := make(chan []byte, 100)

	// Запускаем горутину для чтения сообщений
	go func() {
		defer close(msgChan)
		defer pubsub.Close()

		ch := pubsub.Channel()
		for {
			select {
			case msg := <-ch:
				if msg == nil {
					return
				}
				
				c.logger.WithFields(logrus.Fields{
					"channel": msg.Channel,
					"size":    len(msg.Payload),
				}).Debug("Получено сообщение из Redis")

				select {
				case msgChan <- []byte(msg.Payload):
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return msgChan, nil
}

// Close закрывает соединение с Redis
func (c *Client) Close() error {
	return c.rdb.Close()
}

// GetCommandChannel возвращает имя канала для команд
func GetCommandChannel(serverKey string) string {
	return fmt.Sprintf("cmd:%s", serverKey)
}

// GetResponseChannel возвращает имя канала для ответов
func GetResponseChannel(serverKey string) string {
	return fmt.Sprintf("resp:%s", serverKey)
}
