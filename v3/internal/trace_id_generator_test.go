package internal

import "testing"

func TestTraceIDGenerator(t *testing.T) {
	tg := NewTraceIDGenerator(12345)
	id := tg.GenerateTraceID()
	if id != "d9466896a525ccbf" {
		t.Error(id)
	}
}
