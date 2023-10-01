package dummies

type DummyConnector struct {
	buffer chan byte
}

func NewDummyConnector() *DummyConnector {
	buffer := make(chan byte)
	return &DummyConnector{
		buffer: buffer,
	}
}

func (con *DummyConnector) Close() {
	close(con.buffer)
}

func (con *DummyConnector) Read(buf []byte) (readed int, err error) {

	for ; readed < len(buf); readed++ {
		buf[readed] = <-con.buffer
	}

	return
}

func (con *DummyConnector) Write(buf []byte) (written int, err error) {
	for _, value := range buf {
		con.buffer <- value
		written++
	}
	return
}
