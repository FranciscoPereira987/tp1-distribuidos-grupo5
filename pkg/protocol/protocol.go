package protocol

import (
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/conection"
)

/*
|OP_CODE| => 1 byte
|Message Length| => 4 bytes
|Body| => N bytes
*/
type Protocol struct {
	registry  *Registry
	source    conection.Conn
	connected bool
}

func NewProtocol(conn conection.Conn) *Protocol {
	return &Protocol{
		registry:  NewRegistry(),
		source:    conn,
		connected: false,
	}
}

func (proto *Protocol) Shutdown() error {
	return proto.source.Close()
}

// Protocol.source should be safe to read (not producing short reads)
func (proto *Protocol) readMessage() ([]byte, error) {
	header := make([]byte, 5)
	_, err := proto.source.Read(header)

	if err != nil {
		return nil, err
	}
	body_length, _ := CheckMessageLength(header)
	body := make([]byte, body_length)
	_, err = proto.source.Read(body)
	if err != nil {
		return nil, err
	}
	return append(header, body...), nil
}

func (proto *Protocol) manageResponse(message Message, sent Message) error {
	if !message.IsResponseFrom(sent) {

		if _, ok := message.(*FinMessage); ok {
			proto.sendMessage(NewFinAckMessage())
			return errors.New("connection closed")
		}

		proto.sendMessage(&ErrMessage{})
		return errors.New("got unexpected message from stream")
	}
	return nil
}

func (proto Protocol) checkConnected() (err error) {
	if !proto.connected {
		err = errors.New("not connected")
	}
	return
}

/*
Send a Hello Message and waits for an answer
if the answer is not HelloAck, then returns error
*/
func (proto *Protocol) Connect() error {
	hello := NewHelloMessage(0)
	err := proto.sendMessage(hello)

	if err != nil {
		return err
	}
	stream, err := proto.readMessage()
	if err != nil {
		return err
	}
	recovered, err := proto.registry.GetMessage(stream)
	if err != nil {
		return err
	}
	if err := recovered.UnMarshall(stream); err != nil {
		return err
	}
	if err := proto.manageResponse(recovered, hello); err != nil {
		return err
	}
	proto.connected = true
	return nil

}

/*
Waits for someone to send a Hello Message
if the message is not Hello, then returns error
*/
func (proto *Protocol) Accept() error {
	expected := NewHelloMessage(0)
	recovered, err := proto.readMessage()
	if err != nil {
		return err
	}
	if err := expected.UnMarshall(recovered); err != nil {
		return err
	}
	response := NewHelloAckMessage(0) //Todo, define the user id (va a servir para multiples clientes)
	if err := proto.sendMessage(response); err != nil {
		return err
	}
	proto.connected = true
	return nil
}

// Protocol.source should be safe to write to (not producing short writes)
func (proto *Protocol) sendMessage(message Message) error {
	_, err := proto.source.Write(message.Marshall())

	return err
}

/*
Need to check just for a Fin Message. Otherwise its an invalid message
*/
func (proto *Protocol) manageInvalidData(stream []byte, err error) error {
	fin, _ := proto.registry.GetMessage(stream)

	if finM, ok := fin.(*FinMessage); ok {
		proto.sendMessage(finM.Response())
		proto.connected = false
		return errors.New("connection closed")
	}

	proto.sendMessage(&ErrMessage{})

	return err
}
func (proto *Protocol) Recover(data Data) error {
	if err := proto.checkConnected(); err != nil {
		return err
	}
	stream, err := proto.readMessage()
	if err != nil {
		return err
	}

	if err := data.UnMarshall(stream); err != nil {

		return proto.manageInvalidData(stream, err)
	}
	response := data.Response()

	return proto.sendMessage(response)
}

func (proto *Protocol) Send(data Data) error {
	if err := proto.checkConnected(); err != nil {
		return err
	}

	if err := proto.sendMessage(data); err != nil {
		return err
	}
	stream, err := proto.readMessage()
	if err != nil {
		return err
	}
	recovered, err := proto.registry.GetMessage(stream)
	if err != nil {
		return err
	}
	if err := recovered.UnMarshall(stream); err != nil {
		return err
	}
	return proto.manageResponse(recovered, data)
}

/*
Should not be think of as closing underlying resource (source)
*/
func (proto *Protocol) Close() {
	if proto.checkConnected() != nil {
		return
	}
	fin := &FinMessage{}
	if err := proto.sendMessage(fin); err != nil {
		proto.connected = false
		return
	}

	_, _ = proto.readMessage()

	proto.connected = false
	return
}
