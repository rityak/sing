package udpnat

import (
	"context"
	"net/netip"
	"sync"
	"testing"

	"github.com/metacubex/sing/common/buf"
	M "github.com/metacubex/sing/common/metadata"
	N "github.com/metacubex/sing/common/network"
)

type addrReader struct {
	once sync.Once
	done chan struct{}
}

func (h *addrReader) NewPacketConnection(ctx context.Context, conn N.PacketConn, metadata M.Metadata) error {
	defer h.once.Do(func() { close(h.done) }) // a new session after this one ends runs the handler again
	for i := 0; i < 1000; i++ {
		_ = conn.LocalAddr().String()
	}
	return nil
}

func (h *addrReader) NewError(ctx context.Context, err error) {}

type discardWriter struct{}

func (discardWriter) WritePacket(buffer *buf.Buffer, destination M.Socksaddr) error {
	buffer.Release()
	return nil
}

// The handler reads LocalAddr while packets of the same session arrive from a new
// source address (NAT rebinding): under -race the update must not race with the read.
func TestServiceLocalAddrUpdateWhileHandled(t *testing.T) {
	handler := &addrReader{done: make(chan struct{})}
	service := New[int](60, handler)
	destination := M.SocksaddrFromNetIP(netip.MustParseAddrPort("192.0.2.1:53"))
	for i := 0; i < 50; i++ { // under the 64 packets a session queues: the handler does not read them
		source := M.SocksaddrFromNetIP(netip.AddrPortFrom(netip.MustParseAddr("198.51.100.1"), uint16(10000+i)))
		service.NewPacket(context.Background(), 1, buf.New(), M.Metadata{Source: source, Destination: destination},
			func(natConn N.PacketConn) N.PacketWriter { return discardWriter{} })
	}
	<-handler.done
}
