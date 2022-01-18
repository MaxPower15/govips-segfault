package seq

import (
	"strconv"
	"sync/atomic"
)

type Seq uint64

func (seq *Seq) Reset(val uint64) {
	atomic.StoreUint64((*uint64)(seq), val)
}

func (seq *Seq) Next() uint64 {
	return atomic.AddUint64((*uint64)(seq), 1)
}

func (seq *Seq) NextAsInt() int {
	return int(seq.Next())
}

func (seq *Seq) NextAsString() string {
	return strconv.Itoa(seq.NextAsInt())
}

// Often we don't want to think about setting up counters for special contexts.
// If we just want uniqueness, we can use a global counter.
var globalSeq Seq

func Reset() {
	globalSeq.Reset(0)
}

func Next() uint64 {
	return globalSeq.Next()
}

func NextAsInt() int {
	return globalSeq.NextAsInt()
}

func NextAsString() string {
	return globalSeq.NextAsString()
}
