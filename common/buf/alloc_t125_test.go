package buf

import "testing"

// A full ss2022/AEAD chunk is 65535 bytes plus a 16-byte tag: its buffer comes from the
// pool, not from the heap on every chunk (T-125, as sagernet/sing cbeb0ed).
func TestNewSizeFullChunkIsPooled(t *testing.T) {
	for _, size := range []int{65535, 65536, 65535 + 16, MaxPooledBufferSize} {
		b := NewSize(size)
		if !b.managed || b.FreeLen() != size {
			t.Errorf("NewSize(%d): managed %v, free %d", size, b.managed, b.FreeLen())
		}
		b.Release()
	}
	if b := NewSize(MaxPooledBufferSize + 1); b.managed || b.FreeLen() != MaxPooledBufferSize+1 {
		t.Errorf("NewSize above the pool: managed %v, free %d", b.managed, b.FreeLen())
	}
	if err := DefaultAllocator.Put(make([]byte, MaxPooledBufferSize)); err != nil {
		t.Errorf("Put of a pool-sized buffer: %v", err)
	}
	if err := DefaultAllocator.Put(make([]byte, 70000)); err == nil {
		t.Error("Put of an odd size accepted")
	}
}
