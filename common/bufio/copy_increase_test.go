package bufio

import (
	"bytes"
	"testing"

	"github.com/metacubex/sing/common/buf"

	"github.com/stretchr/testify/require"
)

// headroomWriter stands for an AEAD chunk writer (ss2022: 18 bytes of header in front,
// a 16-byte tag behind) and records the largest buffer it is handed.
type headroomWriter struct {
	n, max int
}

func (w *headroomWriter) Write(p []byte) (int, error) { return len(p), w.WriteBuffer(buf.As(p)) }
func (w *headroomWriter) WriteBuffer(b *buf.Buffer) error {
	w.n += b.Len()
	if b.Len() > w.max {
		w.max = b.Len()
	}
	b.Release()
	return nil
}
func (w *headroomWriter) FrontHeadroom() int { return 18 }
func (w *headroomWriter) RearHeadroom() int  { return 16 }

// After DefaultIncreaseBufferAfter bytes the copy reads with 64 KiB buffers, so a chunk
// writer gets full chunks instead of 32 KiB ones (T-126, as sagernet/sing CopyWithIncreateBuffer).
func TestCopyIncreasesBuffer(t *testing.T) {
	inputConn, outputConn := TCPPipe(t)
	const total = 8 << 20
	go func() {
		_, _ = inputConn.Write(bytes.Repeat([]byte{1}, total))
		_ = inputConn.Close()
	}()
	w := &headroomWriter{}
	n, err := Copy(w, outputConn)
	require.NoError(t, err)
	require.EqualValues(t, total, n)
	require.Equal(t, total, w.n)
	require.Greater(t, w.max, buf.BufferSize)
	require.LessOrEqual(t, w.max, 65535)
}
