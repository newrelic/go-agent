package internal

import (
	"fmt"
	"math/rand"
	"sync"
)

// TraceIDGenerator creates identifiers for distributed tracing.
type TraceIDGenerator struct {
	sync.Mutex
	rnd *rand.Rand
}

// NewTraceIDGenerator creates a new trace identifier generator.
func NewTraceIDGenerator(seed int64) *TraceIDGenerator {
	return &TraceIDGenerator{
		rnd: rand.New(rand.NewSource(seed)),
	}
}

// GenerateTraceID creates a new trace identifier, which is a 32 character hex string.
func (tg *TraceIDGenerator) GenerateTraceID() string {
	return tg.generateID(16)
}

// GenerateSpanID creates a new span identifier, which is a 16 character hex string.
func (tg *TraceIDGenerator) GenerateSpanID() string {
	return tg.generateID(8)
}

func (tg *TraceIDGenerator) generateID(len int) string {
	bits := make([]byte, len)
	tg.Lock()
	defer tg.Unlock()
	tg.rnd.Read(bits)
	return fmt.Sprintf("%016x", bits)
}
