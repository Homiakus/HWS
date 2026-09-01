package nativeframe

import (
	"encoding/binary"
	"fmt"
	"io"
)

const MaxMessage = 4 << 20

func Read(r io.Reader) ([]byte, error) {
	var n uint32
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return nil, err
	}
	if n == 0 || n > MaxMessage {
		return nil, fmt.Errorf("native message size %d out of range", n)
	}
	buf := make([]byte, n)
	_, err := io.ReadFull(r, buf)
	return buf, err
}
func Write(w io.Writer, p []byte) error {
	if len(p) == 0 || len(p) > MaxMessage {
		return fmt.Errorf("native message size %d out of range", len(p))
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(p))); err != nil {
		return err
	}
	_, err := w.Write(p)
	return err
}
