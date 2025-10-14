package storage

import (
	"compress/gzip"
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
