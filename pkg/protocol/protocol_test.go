package protocol_test

import (
	"testing"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/dummies"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

func TestConnection(t *testing.T) {
	connector := dummies.NewDummyConnector()
	defer connector.Close()

	connectorProtocol := protocol.NewProtocol(connector)
	acceptantProtocol := protocol.NewProtocol(connector)

	go connectorProtocol.Connect()

	if err := acceptantProtocol.Accept(); err != nil {
		t.Fatal("could not accept connection")
	}
}

func TestDataTransferFromServerPOV(t *testing.T) {
	connector := dummies.NewDummyConnector()
	defer connector.Close()

	sendant := protocol.NewProtocol(connector)
	reciever := protocol.NewProtocol(connector)

	message, _ := typing.NewStr("A very important message")
	dummyMessage, _ := typing.NewStr("")
	data := protocol.NewDataMessage(message)
	recieverData := protocol.NewDataMessage(dummyMessage)
	go func() {
		sendant.Connect()
		sendant.Send(data)
	}()
	if err := reciever.Accept(); err != nil {
		t.Fatalf("failed to initiate connection: %s", err)
	}
	if err := reciever.Recover(recieverData); err != nil {
		t.Fatalf("issue while recovering data: %s", err)
	}
}

func TestDataTransferFromClientPOV(t *testing.T) {
	connector := dummies.NewDummyConnector()
	defer connector.Close()

	sendant := protocol.NewProtocol(connector)
	reciever := protocol.NewProtocol(connector)

	message, _ := typing.NewStr("A very important message")
	dummyMessage, _ := typing.NewStr("")
	data := protocol.NewDataMessage(message)
	recieverData := protocol.NewDataMessage(dummyMessage)
	go func() {
		reciever.Accept()
		reciever.Recover(recieverData)
	}()
	if err := sendant.Connect(); err != nil {
		t.Fatalf("failed to initiate connection: %s", err)
	}
	if err := sendant.Send(data); err != nil {
		t.Fatalf("issue while recovering data: %s", err)
	}
}
