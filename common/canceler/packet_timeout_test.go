package canceler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/metacubex/sing/common/buf"
	M "github.com/metacubex/sing/common/metadata"
	N "github.com/metacubex/sing/common/network"
)

type concurrentPacket struct{ N.PacketConn }

func (concurrentPacket) ReadPacket(*buf.Buffer) (M.Socksaddr, error)    { return M.Socksaddr{}, nil }
func (concurrentPacket) WritePacket(b *buf.Buffer, _ M.Socksaddr) error { b.Release(); return nil }
func (concurrentPacket) SetReadDeadline(time.Time) error                { return nil }
func (concurrentPacket) InitializeReadWaiter(N.ReadWaitOptions) bool    { return false }
func (concurrentPacket) WaitReadPacket() (*buf.Buffer, M.Socksaddr, error) {
	return buf.New(), M.Socksaddr{}, nil
}

func TestTimeoutPacketConcurrentActivity(t *testing.T) {
	_, conn := NewTimeoutPacketConn(context.Background(), concurrentPacket{}, time.Second)
	c := conn.(*TimeoutPacketConn)
	waiter, ok := c.CreateReadWaiter()
	if !ok {
		t.Fatal("read waiter unavailable")
	}
	var wg sync.WaitGroup
	for _, operation := range []func(){
		func() { b := buf.New(); defer b.Release(); c.ReadPacket(b) },
		func() { c.WritePacket(buf.New(), M.Socksaddr{}) },
		func() { b, _, _ := waiter.WaitReadPacket(); b.Release() },
		func() { c.SetTimeout(time.Second); c.Timeout(); c.isInactive() },
	} {
		wg.Add(1)
		go func(operation func()) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				operation()
			}
		}(operation)
	}
	wg.Wait()
	if c.isInactive() {
		t.Fatal("successful traffic did not refresh activity")
	}
}
