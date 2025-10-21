package storage

import (
	"compress/gzip"
	"io"
)

func writeFrame(w *gzip.Writer, data []byte) error {
	var lenBuf [8]byte
	l := uint64(len(data))
	for i := 7; i >= 0; i-- {
		lenBuf[i] = byte(l & 0xff)
		l >>= 8
	}
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	return nil
}

func readFrame(r *gzip.Reader) ([]byte, error) {
	var lenBuf [8]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	var l uint64
	for i := 0; i < 8; i++ {
		l = (l << 8) | uint64(lenBuf[i])
	}
	data := make([]byte, l)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}
