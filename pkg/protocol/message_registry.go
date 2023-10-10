package protocol

import (
	"errors"
)

type Registry struct {
	registry map[byte]Message
}

func getStartingMessages() []Message {
	messages := []Message{NewHelloAckMessage()}
	messages = append(messages, NewHelloMessage(0))
	messages = append(messages, &FinMessage{})
	messages = append(messages, &ErrMessage{})

	return messages
}

func NewRegistry() *Registry {
	registry := make(map[byte]Message)
	messages := getStartingMessages()

	for _, message := range messages {
		registry[message.Number()] = message
	}

	return &Registry{
		registry: registry,
	}
}

func (reg *Registry) GetMessage(stream []byte) (Message, error) {
	if _, ok := reg.registry[stream[0]]; !ok {
		return nil, errors.New("unregistered message")
	}
	return reg.registry[stream[0]], nil
}
