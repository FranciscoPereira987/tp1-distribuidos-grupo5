package invitation

import "net"

type Config struct {
	Peers     []uint
	Id        uint
	Mapping   map[uint]string
	Conn      *net.UDPConn
	Heartbeat string
	Name      string
	Names     []string
}

/*
Configures the Id and starts the UDPConn on the desired port
*/
func NewConfigOn(id uint, port string) (*Config, error) {
	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+port)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	return &Config{
		Peers:   make([]uint, 0),
		Id:      id,
		Mapping: make(map[uint]string),
		Conn:    conn,
	}, nil
}
