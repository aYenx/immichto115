package open115

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestPacerWaitDoesNotBlockForever(t *testing.T) {
	p := NewPacer()
	done := make(chan struct{})
	go func() {
		_ = p.Wait(context.Background())
		close(done)
	}()
	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("Wait() blocked for too long")
	}
}

func TestPacerEndCallDecay(t *testing.T) {
	p := NewPacer()
	initial := p.Interval()
	// 成功调用后 interval 应该保持或衰减
	p.EndCall(nil) // nil = success
	after := p.Interval()
	if after > initial {
		t.Fatalf("expected interval to stay same or decrease after success, got %v -> %v", initial, after)
	}
}

func TestPacerEndCallBurstIncrease(t *testing.T) {
	p := NewPacer()
	initial := p.Interval()
	// 失败调用（限流错误）应导致 interval 增大
	p.EndCall(fmt.Errorf("refresh frequently"))
	after := p.Interval()
	if after <= initial {
		t.Fatalf("expected interval to increase after rate limit hit, got %v -> %v", initial, after)
	}
}

func TestPacerCallSuccess(t *testing.T) {
	p := NewPacer()
	ctx := context.Background()
	result, err := Call(ctx, p, "TestOp", 3, func() (string, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected 'ok', got %q", result)
	}
}

func TestPacerCallNoReturnSuccess(t *testing.T) {
	p := NewPacer()
	ctx := context.Background()
	err := CallNoReturn(ctx, p, "TestOp", 3, func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

