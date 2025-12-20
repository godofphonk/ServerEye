package agent

// TODO: Uncomment when ping functionality is needed
// handlePing обрабатывает ping команду
// func (a *Agent) handlePing(msg *protocol.Message) *protocol.Message {
// 	payload := protocol.PongPayload{
// 		Status: "healthy",
// 		Uptime: "unknown",
// 	}
//
// 	response := protocol.NewMessage(protocol.TypePong, payload)
// 	response.ID = msg.ID
// 	return response
// }

// TODO: Uncomment when unknown command handling is needed
// handleUnknownCommand обрабатывает неизвестную команду
// func (a *Agent) handleUnknownCommand(msg *protocol.Message) *protocol.Message {
// 	payload := protocol.ErrorPayload{
// 		ErrorCode:    protocol.ErrorInvalidCommand,
// 		ErrorMessage: fmt.Sprintf("Неизвестная команда: %s", msg.Type),
// 	}
//
// 	response := protocol.NewMessage(protocol.TypeErrorResponse, payload)
// 	response.ID = msg.ID
// 	return response
// }

// TODO: Uncomment when CPU temp functionality is needed
// handleGetCPUTemp обрабатывает запрос температуры CPU
// func (a *Agent) handleGetCPUTemp(msg *protocol.Message) *protocol.Message {
// 	temp, err := a.cpuMetrics.GetTemperature()
// 	if err != nil {
// 		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 			ErrorCode:    protocol.ErrorCommandTimeout,
// 			ErrorMessage: fmt.Sprintf("Failed to get CPU temperature: %v", err),
// 		})
// 	}
//
// 	response := protocol.NewMessage(protocol.TypeCPUTempResponse, map[string]interface{}{
// 		"temperature": temp,
// 		"timestamp":   msg.Timestamp,
// 	})
// 	response.ID = msg.ID
// 	return response
// }

// TODO: Uncomment when containers functionality is needed
// handleGetContainers обрабатывает запрос списка контейнеров
// func (a *Agent) handleGetContainers(msg *protocol.Message) *protocol.Message {
// 	if a.dockerClient == nil {
// 		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 			ErrorCode:    protocol.ErrorDockerUnavailable,
// 			ErrorMessage: "Docker client not available",
// 		})
// 	}
//
// 	containers, err := a.dockerClient.GetContainers(context.Background())
// 	if err != nil {
// 		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 			ErrorCode:    protocol.ErrorDockerUnavailable,
// 			ErrorMessage: fmt.Sprintf("Failed to get containers: %v", err),
// 		})
// 	}
//
// 	response := protocol.NewMessage(protocol.TypeContainersResponse, map[string]interface{}{
// 		"containers": containers,
// 		"count":      len(containers.Containers),
// 	})
// 	response.ID = msg.ID
// 	return response
// }

// TODO: Uncomment when Docker commands are needed
// Заглушки для остальных Docker команд
// func (a *Agent) handleStartContainer(msg *protocol.Message) *protocol.Message {
// 	return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 		ErrorCode:    protocol.ErrorInvalidCommand,
// 		ErrorMessage: "Start container command not implemented",
// 	})
// }
//
// func (a *Agent) handleStopContainer(msg *protocol.Message) *protocol.Message {
// 	return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 		ErrorCode:    protocol.ErrorInvalidCommand,
// 		ErrorMessage: "Stop container command not implemented",
// 	})
// }
//
// func (a *Agent) handleRestartContainer(msg *protocol.Message) *protocol.Message {
// 	return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 		ErrorCode:    protocol.ErrorInvalidCommand,
// 		ErrorMessage: "Restart container command not implemented",
// 	})
// }
//
// func (a *Agent) handleRemoveContainer(msg *protocol.Message) *protocol.Message {
// 	return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 		ErrorCode:    protocol.ErrorInvalidCommand,
// 		ErrorMessage: "Remove container command not implemented",
// 	})
// }
//
// func (a *Agent) handleCreateContainer(msg *protocol.Message) *protocol.Message {
// 	return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 		ErrorCode:    protocol.ErrorInvalidCommand,
// 		ErrorMessage: "Create container command not implemented",
// 	})
// }

// TODO: Uncomment when memory info functionality is needed
// handleGetMemoryInfo обрабатывает запрос информации о памяти
// func (a *Agent) handleGetMemoryInfo(msg *protocol.Message) *protocol.Message {
// 	if a.systemMonitor == nil {
// 		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 			ErrorCode:    protocol.ErrorCommandTimeout,
// 			ErrorMessage: "System monitor not available",
// 		})
// 	}
//
// 	memInfo, err := a.systemMonitor.GetMemoryInfo()
// 	if err != nil {
// 		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 			ErrorCode:    protocol.ErrorCommandTimeout,
// 			ErrorMessage: fmt.Sprintf("Failed to get memory info: %v", err),
// 		})
// 	}
//
// 	response := protocol.NewMessage(protocol.TypeMemoryInfoResponse, memInfo)
// 	response.ID = msg.ID
// 	return response
// }

// TODO: Uncomment when disk info functionality is needed
// handleGetDiskInfo обрабатывает запрос информации о диске
// func (a *Agent) handleGetDiskInfo(msg *protocol.Message) *protocol.Message {
// 	if a.systemMonitor == nil {
// 		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 			ErrorCode:    protocol.ErrorCommandTimeout,
// 			ErrorMessage: "System monitor not available",
// 		})
// 	}
//
// 	diskInfo, err := a.systemMonitor.GetDiskInfo()
// 	if err != nil {
// 		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 			ErrorCode:    protocol.ErrorCommandTimeout,
// 			ErrorMessage: fmt.Sprintf("Failed to get disk info: %v", err),
// 		})
// 	}
//
// 	response := protocol.NewMessage(protocol.TypeDiskInfoResponse, map[string]interface{}{
// 		"disks": diskInfo.Disks,
// 	})
// 	response.ID = msg.ID
// 	return response
// }

// TODO: Uncomment when uptime functionality is needed
// handleGetUptime обрабатывает запрос uptime
// func (a *Agent) handleGetUptime(msg *protocol.Message) *protocol.Message {
// 	if a.systemMonitor == nil {
// 		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 			ErrorCode:    protocol.ErrorCommandTimeout,
// 			ErrorMessage: "System monitor not available",
// 		})
// 	}
//
// 	uptime, err := a.systemMonitor.GetUptime()
// 	if err != nil {
// 		return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 			ErrorCode:    protocol.ErrorCommandTimeout,
// 			ErrorMessage: fmt.Sprintf("Failed to get uptime: %v", err),
// 		})
// 	}
//
// 	response := protocol.NewMessage(protocol.TypeUptimeResponse, map[string]interface{}{
// 		"uptime":    uptime.Uptime,
// 		"boot_time": uptime.BootTime,
// 	})
// 	response.ID = msg.ID
// 	return response
// }

// TODO: Uncomment when processes functionality is needed
// handleGetProcesses обрабатывает запрос списка процессов
// func (a *Agent) handleGetProcesses(msg *protocol.Message) *protocol.Message {
// 	return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 		ErrorCode:    protocol.ErrorInvalidCommand,
// 		ErrorMessage: "Get processes command not implemented",
// 	})
// }

// TODO: Uncomment when network info functionality is needed
// handleGetNetworkInfo обрабатывает запрос информации о сети
// func (a *Agent) handleGetNetworkInfo(msg *protocol.Message) *protocol.Message {
// 	return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 		ErrorCode:    protocol.ErrorInvalidCommand,
// 		ErrorMessage: "Get network info command not implemented",
// 	})
// }

// TODO: Uncomment when update functionality is needed
// handleUpdateAgent обрабатывает запрос обновления агента
// func (a *Agent) handleUpdateAgent(msg *protocol.Message) *protocol.Message {
// 	return protocol.NewMessage(protocol.TypeErrorResponse, protocol.ErrorPayload{
// 		ErrorCode:    protocol.ErrorInvalidCommand,
// 		ErrorMessage: "Update agent command not implemented",
// 	})
// }
