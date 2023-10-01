package ack_types

import "github.com/franciscopereira987/tp1-distribuidos/pkg/typing"

type AckType interface {
	typing.Type
	IsAckFrom(ack typing.Type) bool
}
