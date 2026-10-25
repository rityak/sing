package bufio

import (
	"errors"
	"io"
	"net"

	"github.com/metacubex/sing/common/buf"
	N "github.com/metacubex/sing/common/network"
)

// vectorisedBatchSize: buffers per readv, as sagernet/sing DefaultBatchSize.
const vectorisedBatchSize = 8

type vectorisedReadWaiter interface {
	N.ReadWaitable
	WaitReadBuffers() (buffers []*buf.Buffer, err error)
	release()
}

// explicitVectorisedWriter is the destination's own vectorised path: a writer that
// implements N.VectorisedWriter (after the replaceable wrappers), or a plain TCP or Unix
// socket. A wrapper that only exposes SyscallConn is not taken: writev on its fd would
// skip what the wrapper does to the bytes (T-127).
func explicitVectorisedWriter(destination io.Writer) (N.VectorisedWriter, bool) {
	switch w := N.UnwrapWriter(destination).(type) {
	case N.VectorisedWriter:
		return w, true
	case *net.TCPConn:
		return &NetVectorisedWriterWrapper{w}, true
	case *net.UnixConn:
		return &NetVectorisedWriterWrapper{w}, true
	}
	return nil, false
}

func copyWaitVectorisedWithPool(originSource io.Reader, destination N.VectorisedWriter, source vectorisedReadWaiter, readCounters []N.CountFunc, writeCounters []N.CountFunc) (n int64, err error) {
	defer source.release()
	for {
		var buffers []*buf.Buffer
		buffers, err = source.WaitReadBuffers()
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
			}
			return
		}
		var dataLen int
		for _, buffer := range buffers {
			dataLen += buffer.Len()
		}
		err = destination.WriteVectorised(buffers)
		if err != nil {
			for _, buffer := range buffers {
				buffer.Leak()
			}
			return
		}
		n += int64(dataLen)
		for _, counter := range readCounters {
			counter(int64(dataLen))
		}
		for _, counter := range writeCounters {
			counter(int64(dataLen))
		}
	}
}
