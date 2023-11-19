package utils

import (
	"encoding/binary"
	"errors"
	"net"
)

var (
	ErrTooShort      = errors.New("stream too short")
	ErrInvalidString = errors.New("invalid string")
)

func getLength(stream []byte) int {
	if len(stream) != 4 {
		return 0
	}
	return int(binary.LittleEndian.Uint32(stream))
}

func addLength(stream []byte) []byte {
	length := binary.LittleEndian.AppendUint32(nil, uint32(len(stream)))
	return append(length, stream...)
}

func EncodeString(str string) []byte {
	stream := []byte(str)
	length := binary.LittleEndian.AppendUint32(nil, uint32(len(stream)))
	return append(length, stream...)
}

func DecodeString(stream []byte) (string, error) {
	if len(stream) < 4 {
		return "", ErrTooShort
	}
	length := binary.LittleEndian.Uint32(stream[:4])
	if len(stream[4:]) != int(length) {
		return "", ErrInvalidString
	}
	return string(stream[4:]), nil
}

func SafeWriteTo(stream []byte, sckt *net.UDPConn, to *net.UDPAddr) error {

	n, err := sckt.WriteTo(addLength(stream), to)
	if n != len(stream) {
		return err
	}
	return nil
}

func readBuf(buf []byte, sckt *net.UDPConn) (*net.UDPAddr, error) {
	n, addr, err := sckt.ReadFromUDP(buf)

	if n == 0 {
		return nil, err
	}
	if n >= 4 {
		return addr, nil
	}
	return addr, err
}

func SafeReadFrom(sckt *net.UDPConn) ([]byte, *net.UDPAddr, error) {
	buf := make([]byte, 1024)
	from, err := readBuf(buf, sckt)

	if err == nil {
		length := getLength(buf[:4])
		buf = buf[4 : 4+length]
	}

	return buf, from, err
}
