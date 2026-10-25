package bufio

import N "github.com/metacubex/sing/common/network"

// No readv path on Windows (T-127): the copy stays on one buffer per read.
func vectorisedFrom(N.ReadWaiter) (vectorisedReadWaiter, bool) { return nil, false }
