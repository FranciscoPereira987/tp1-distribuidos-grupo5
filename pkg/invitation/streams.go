package invitation

import (
	"encoding/binary"
	"errors"
)

var (
	InvalidStreamErr = errors.New("Invalid stream")
	InvalidGroupSize = errors.New("Invalid group size")
)

type serializable interface {
	serialize() []byte
}

func (inv invite) serialize() (stream []byte) {
	stream = []byte{Invite}
	stream = binary.LittleEndian.AppendUint32(stream, uint32(inv.Id))
	stream = binary.LittleEndian.AppendUint32(stream, uint32(inv.GroupSize))
	return
}

func deserializeInv(stream []byte) (inv invite, err error) {
	if len(stream) != 8 {
		err = InvalidStreamErr
	}

	if err == nil {
		inv.Id = uint(binary.LittleEndian.Uint32(stream[:4]))
		inv.GroupSize = uint(binary.LittleEndian.Uint32(stream[4:]))
	}

	return
}

func (rej reject) serialize() (stream []byte) {
	stream = []byte{Reject}
	stream = binary.LittleEndian.AppendUint32(stream, uint32(rej.LeaderId))
	return
}

func deserializeRej(stream []byte) (rej reject, err error) {
	if len(stream) != 4 {
		err = InvalidStreamErr
	}

	if err == nil {
		rej.LeaderId = uint(binary.LittleEndian.Uint32(stream))
	}

	return
}

func (acc accept) serialize() (stream []byte) {
	stream = []byte{Accept}
	stream = binary.LittleEndian.AppendUint32(stream, uint32(acc.From))
	stream = binary.LittleEndian.AppendUint32(stream, uint32(acc.GroupSize))
	for _, member := range acc.Members {
		stream = binary.LittleEndian.AppendUint32(stream, uint32(member))
	}
	return
}

func deserializeAcc(stream []byte) (acc accept, err error) {
	if len(stream) < 8 {
		err = InvalidStreamErr
	}

	if err == nil {
		acc.From = uint(binary.LittleEndian.Uint32(stream[:4]))
	}
	stream = stream[4:]
	if err == nil {
		acc.GroupSize = uint(binary.LittleEndian.Uint32(stream[:4]))
	}

	if err == nil {
		stream = stream[4:]
		for len(stream) > 0 {
			acc.Members = append(acc.Members, uint(binary.LittleEndian.Uint32(stream[:4])))
			stream = stream[4:]
		}
	}

	if len(acc.Members) != int(acc.GroupSize) {
		err = InvalidGroupSize
	}

	return
}

func (ch change) serialize() (stream []byte) {
	stream = []byte{Change}
	return binary.LittleEndian.AppendUint32(stream, uint32(ch.NewLeaderId))
}

func deserializeChange(stream []byte) (ch change, err error) {
	if len(stream) != 4 {
		err = InvalidStreamErr
	}

	if err == nil {
		ch.NewLeaderId = uint(binary.LittleEndian.Uint32(stream))
	}

	return
}

func (hb heartbeat) serialize() []byte {
	return []byte{Heartbeat}
}

func (k ok) serialize() []byte {
	return []byte{Ok}
}
