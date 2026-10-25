//go:build !windows

package bufio

import (
	"bytes"
	"testing"

	"github.com/metacubex/sing/common/buf"

	"github.com/stretchr/testify/require"
)

// vectorWriter stands for a chunk writer with a vectorised path (the ss2022 server
// connection) and records what it is handed.
type vectorWriter struct {
	data            bytes.Buffer
	calls, maxBatch int
}

func (w *vectorWriter) Write(p []byte) (int, error) { return w.data.Write(p) }
func (w *vectorWriter) WriteBuffer(b *buf.Buffer) error {
	_, err := w.data.Write(b.Bytes())
	b.Release()
	return err
}
func (w *vectorWriter) WriteVectorised(buffers []*buf.Buffer) error {
	w.calls++
	if len(buffers) > w.maxBatch {
		w.maxBatch = len(buffers)
	}
	for _, b := range buffers {
		w.data.Write(b.Bytes())
		b.Release()
	}
	return nil
}

// Past DefaultIncreaseBufferAfter a TCP source is read with readv into several frame-sized
// buffers at once and handed to a vectorised writer whole (T-127, as sagernet/sing
// copyWaitVectorisedWithPool); the bytes and their order are unchanged.
func TestCopyVectorisedRead(t *testing.T) {
	inputConn, outputConn := TCPPipe(t)
	payload := make([]byte, 16<<20)
	for i := range payload {
		payload[i] = byte(i * 7)
	}
	go func() {
		_, _ = inputConn.Write(payload)
		_ = inputConn.Close()
	}()
	w := &vectorWriter{}
	n, err := Copy(w, outputConn)
	require.NoError(t, err)
	require.EqualValues(t, len(payload), n)
	require.True(t, bytes.Equal(payload, w.data.Bytes()))
	require.Greater(t, w.calls, 0)
	require.Greater(t, w.maxBatch, 1)
}
