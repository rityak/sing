package bufio

import (
	"bytes"
	"io"
	"testing"

	"github.com/metacubex/sing/common/buf"
)

// chunkSource returns one chunk per ReadBuffer, as a chunked protocol reader does.
type chunkSource struct {
	chunks [][]byte
}

func (s *chunkSource) Read(p []byte) (int, error) {
	if len(s.chunks) == 0 {
		return 0, io.EOF
	}
	n := copy(p, s.chunks[0])
	s.chunks = s.chunks[1:]
	return n, nil
}

func (s *chunkSource) ReadBuffer(buffer *buf.Buffer) error {
	if len(s.chunks) == 0 {
		return io.EOF
	}
	_, err := buffer.Write(s.chunks[0])
	s.chunks = s.chunks[1:]
	return err
}

// A short Read leaves the rest of the chunk in the cache; a later ReadBuffer with a
// buffer of a whole chunk must return that rest before reading the next chunk.
func TestChunkReaderReadBufferAfterShortRead(t *testing.T) {
	first, second := bytes.Repeat([]byte{1}, 15000), bytes.Repeat([]byte{2}, 15000)
	reader := NewChunkReader(&chunkSource{chunks: [][]byte{first, second}}, 16384)
	var got []byte
	p := make([]byte, 4096)
	n, err := reader.Read(p)
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, p[:n]...)
	for {
		buffer := buf.NewSize(32768)
		err = reader.ReadBuffer(buffer)
		got = append(got, buffer.Bytes()...)
		buffer.Release()
		if err != nil {
			break
		}
	}
	if want := append(append([]byte{}, first...), second...); !bytes.Equal(got, want) {
		t.Fatalf("read %d bytes, want %d in order", len(got), len(want))
	}
}
