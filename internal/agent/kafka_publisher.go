package agent

import (
	"context"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/sirupsen/logrus"
)

// publishMetricToKafka отправляет метрику в Kafka (если настроен)
func (a *Agent) publishMetricToKafka(msg *protocol.Message) {
	// Пропускаем если publisher не настроен
	if a.metricPublisher == nil {
		return
	}

	// Пропускаем ошибки и некоторые служебные сообщения
	if msg.Type == protocol.TypeErrorResponse ||
		msg.Type == protocol.TypePong ||
		msg.Type == protocol.TypeUpdateAgentResponse {
		return
	}

	// Конвертируем в метрику
	metric := a.ConvertToMetric(msg)

	// Асинхронная отправка чтобы не блокировать обработку команд
	go func() {
		ctx := context.Background()
		if err := a.metricPublisher.Publish(ctx, metric); err != nil {
			a.logger.WithFields(logrus.Fields{
				"metric_type": metric.Type,
				"server_id":   metric.ServerID,
				"error":       err,
			}).Warn("Не удалось отправить метрику в Kafka")
		} else {
			a.logger.WithFields(logrus.Fields{
				"metric_type": metric.Type,
				"server_id":   metric.ServerID,
			}).Debug("Метрика отправлена в Kafka")
		}
	}()
}
