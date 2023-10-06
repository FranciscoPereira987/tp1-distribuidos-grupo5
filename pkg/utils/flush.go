package utils

import "io"

/*
	Implements the Flood protocol
	and returns wether the flush has ended based on a counter
*/
type Flooder struct {
	counter int
	seen bool
}

func NewFlooder() *Flooder {
	return &Flooder{
		counter: 0,
	}
}

func (f *Flooder) Increase() {
	f.counter++
}

func (f *Flooder) Flood(value int, buf []byte, writer io.Writer) error {
	if value == 0 && f.seen{
		
		return nil
	}
	_, err := writer.Write(buf)
	return err
}

func (f *Flooder) Reduce(value int) int {
	if f.seen {
		return value
	}
	
	f.seen = true
	
	return value - f.counter
}
