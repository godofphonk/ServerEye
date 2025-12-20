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
