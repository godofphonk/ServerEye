// handlePostMetrics handles incoming metrics from agents
func (s *Server) handlePostMetrics(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var metric struct {
		ServerID   string                 `json:"server_id"`
		ServerKey  string                 `json:"server_key"`
		Type       string                 `json:"type"`
		Value      interface{}            `json:"value"`
		Timestamp  time.Time              `json:"timestamp"`
		Tags       map[string]string      `json:"tags"`
		Data       map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		s.writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if metric.ServerID == "" {
		s.writeError(w, "server_id is required", http.StatusBadRequest)
		return
	}
	if metric.ServerKey == "" {
		s.writeError(w, "server_key is required", http.StatusBadRequest)
		return
	}
	if metric.Type == "" {
		s.writeError(w, "type is required", http.StatusBadRequest)
		return
	}

	// Store metric
	metricToStore := &publisher.Metric{
		ServerID:   metric.ServerID,
		ServerKey:  metric.ServerKey,
		Type:       metric.Type,
		Value:      metric.Value,
		Timestamp:  metric.Timestamp,
		Tags:       metric.Tags,
		Data:       metric.Data,
	}

	if err := s.storage.StoreMetric(r.Context(), metricToStore); err != nil {
		s.logger.WithError(err).Error("Failed to store metric")
		s.writeError(w, "Failed to store metric", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	s.writeJSON(w, map[string]string{"status": "accepted"})
}
