package agent

import (
	"time"

	"github.com/servereye/servereye/pkg/protocol"
	"github.com/servereye/servereye/pkg/publisher"
)

// ConvertToMetric конвертирует протокольное сообщение в метрику для publisher
func (a *Agent) ConvertToMetric(msg *protocol.Message) *publisher.Metric {
	return &publisher.Metric{
		ServerID:   a.config.Server.SecretKey, // Используем secret key как server ID
		ServerKey:  a.config.Server.SecretKey,
		ServerName: a.config.Server.Name,
		Type:       string(msg.Type),
		Timestamp:  msg.Timestamp,
		Value:      msg.Payload,
		Tags: map[string]string{
			"version":     msg.Version,
			"description": a.config.Server.Description,
		},
		Version: "1.0",
	}
}

// CreateMetricFromData создает метрику напрямую из данных
func (a *Agent) CreateMetricFromData(metricType string, value interface{}, tags map[string]string) *publisher.Metric {
	if tags == nil {
		tags = make(map[string]string)
	}
	
	// Добавляем дефолтные теги
	tags["server_name"] = a.config.Server.Name
	if a.config.Server.Description != "" {
		tags["description"] = a.config.Server.Description
	}
	
	return &publisher.Metric{
		ServerID:   a.config.Server.SecretKey,
		ServerKey:  a.config.Server.SecretKey,
		ServerName: a.config.Server.Name,
		Type:       metricType,
		Timestamp:  time.Now(),
		Value:      value,
		Tags:       tags,
		Version:    "1.0",
	}
}
