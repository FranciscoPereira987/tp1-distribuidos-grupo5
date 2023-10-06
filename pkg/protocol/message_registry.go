package protocol

import (
	"errors"
)

type Registry struct {
	registry map[byte]Message
}

func getStartingMessages() []Message {
	messages := []Message{NewDataAckMessage(), NewFinAckMessage(), newAckRegistry()}
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

type ackRegistry struct {
	registry  map[byte]Message
	lastFound Message
}

func registerAcks(reg map[byte]Message) {
	ack := NewHelloAckMessage(0).(*AckMessage)
	reg[ack.bodyNumber()] = ack
	ack = NewDataAckMessage().(*AckMessage)
	reg[ack.bodyNumber()] = ack
	ack = NewFinAckMessage().(*AckMessage)
	reg[ack.bodyNumber()] = ack
}

func newAckRegistry() Message {
	reg := make(map[byte]Message)
	registerAcks(reg)
	return &ackRegistry{
		registry:  reg,
		lastFound: nil,
	}
}

func (reg *ackRegistry) Number() byte {
	return byte(ACK_OP_CODE)
}

func (reg *ackRegistry) Response() Message {
	return reg.lastFound.Response()
}

func (reg *ackRegistry) UnMarshal(stream []byte) error {
	message, ok := reg.registry[stream[TYPE_INDEX]]
	if !ok {
		return errors.New("invalid ack message type")
	}
	reg.lastFound = message
	return message.UnMarshal(stream)
}

func (reg *ackRegistry) IsResponseFrom(message Message) bool {
	return reg.lastFound.IsResponseFrom(message)
}

func (reg *ackRegistry) Marshal() []byte {
	return nil
}
