//go:build !windows

package bufio

import (
	"io"
	"os"
	"syscall"
	"unsafe"

	"github.com/metacubex/sing/common/buf"
	E "github.com/metacubex/sing/common/exceptions"
	N "github.com/metacubex/sing/common/network"

	"golang.org/x/sys/unix"
)

// vectorisedSyscallReadWaiter reads a stream fd with readv into up to vectorisedBatchSize
// buffers at once (T-127, as sagernet/sing): a bulk copy then makes one read syscall per
// several frames instead of one per frame.
type vectorisedSyscallReadWaiter struct {
	rawConn     syscall.RawConn
	readBuffers int
	readErr     error
	readFunc    func(fd uintptr) (done bool)
	buffers     []*buf.Buffer
	iovecList   []unix.Iovec
	options     N.ReadWaitOptions
}

// vectorisedFrom gives the readv waiter for the same fd a syscall read waiter reads: only
// a raw stream fd, which the copy already reads directly, is read this way.
func vectorisedFrom(readWaiter N.ReadWaiter) (vectorisedReadWaiter, bool) {
	if w, ok := readWaiter.(*syscallReadWaiter); ok {
		return &vectorisedSyscallReadWaiter{rawConn: w.rawConn}, true
	}
	return nil, false
}

func (w *vectorisedSyscallReadWaiter) InitializeReadWaiter(options N.ReadWaitOptions) (needCopy bool) {
	w.options = options
	w.buffers = make([]*buf.Buffer, vectorisedBatchSize)
	w.iovecList = make([]unix.Iovec, vectorisedBatchSize)
	w.readFunc = func(fd uintptr) (done bool) {
		for i := range w.buffers {
			if w.buffers[i] == nil {
				w.buffers[i] = w.options.NewBuffer()
			}
			w.iovecList[i] = w.buffers[i].Iovec(w.buffers[i].FreeLen())
		}
		//nolint:staticcheck
		readN, _, readErrno := unix.Syscall(unix.SYS_READV, fd, uintptr(unsafe.Pointer(&w.iovecList[0])), uintptr(len(w.iovecList)))
		//goland:noinspection GoDirectComparisonOfErrors
		if readErrno == unix.EAGAIN || readErrno == unix.EWOULDBLOCK {
			return false
		} else if readErrno != 0 {
			w.readErr = readErrno
		} else {
			w.readErr = nil
		}
		if readErrno == 0 && readN > 0 {
			pendingN := int(readN)
			for _, buffer := range w.buffers {
				w.readBuffers++
				if pendingN > buffer.FreeLen() {
					pendingN -= buffer.FreeLen()
					buffer.Truncate(buffer.FreeLen())
					w.options.PostReturn(buffer)
				} else {
					buffer.Truncate(pendingN)
					w.options.PostReturn(buffer)
					break
				}
			}
		}
		if readN == 0 && w.readErr == nil {
			w.readErr = io.EOF
		}
		return true
	}
	return false
}

func (w *vectorisedSyscallReadWaiter) WaitReadBuffers() (buffers []*buf.Buffer, err error) {
	if w.readFunc == nil {
		return nil, os.ErrInvalid
	}
	err = w.rawConn.Read(w.readFunc)
	if err != nil {
		return
	}
	if w.readErr != nil {
		if w.readErr == io.EOF {
			return nil, io.EOF
		}
		return nil, E.Cause(w.readErr, "raw readv")
	}
	buffers = make([]*buf.Buffer, w.readBuffers)
	for i := 0; i < w.readBuffers; i++ {
		buffers[i] = w.buffers[i]
		w.buffers[i] = nil
	}
	w.readBuffers = 0
	return
}

// release returns the buffers readv did not fill.
func (w *vectorisedSyscallReadWaiter) release() {
	for i, buffer := range w.buffers {
		if buffer != nil {
			buffer.Release()
			w.buffers[i] = nil
		}
	}
}
