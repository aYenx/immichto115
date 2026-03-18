package api

import (
	"testing"
	"time"
)

func TestAuthLimiter_InitiallyNotBlocked(t *testing.T) {
	l := newAuthLimiter()
	blocked, _ := l.check("1.2.3.4")
	if blocked {
		t.Fatal("expected not blocked initially")
	}
}

func TestAuthLimiter_ProgressiveDelay(t *testing.T) {
	l := newAuthLimiter()
	l.baseDelay = 10 * time.Millisecond // 加快测试

	d1 := l.recordFail("1.2.3.4")
	d2 := l.recordFail("1.2.3.4")
	d3 := l.recordFail("1.2.3.4")

	if d2 <= d1 {
		t.Fatalf("expected d2(%v) > d1(%v)", d2, d1)
	}
	if d3 <= d2 {
		t.Fatalf("expected d3(%v) > d2(%v)", d3, d2)
	}
}

func TestAuthLimiter_LockoutAfterMaxFail(t *testing.T) {
	l := newAuthLimiter()
	l.maxFail = 3
	l.lockoutDuration = 1 * time.Second
	l.baseDelay = 1 * time.Millisecond

	for i := 0; i < 3; i++ {
		l.recordFail("1.2.3.4")
	}

	blocked, retryAfter := l.check("1.2.3.4")
	if !blocked {
		t.Fatal("expected blocked after max failures")
	}
	if retryAfter <= 0 {
		t.Fatalf("expected positive retryAfter, got %d", retryAfter)
	}
}

func TestAuthLimiter_SuccessResetsCounter(t *testing.T) {
	l := newAuthLimiter()
	l.baseDelay = 1 * time.Millisecond

	l.recordFail("1.2.3.4")
	l.recordFail("1.2.3.4")
	l.recordSuccess("1.2.3.4")

	blocked, _ := l.check("1.2.3.4")
	if blocked {
		t.Fatal("expected not blocked after success")
	}
}

func TestAuthLimiter_DifferentIPsIndependent(t *testing.T) {
	l := newAuthLimiter()
	l.maxFail = 2
	l.lockoutDuration = 1 * time.Second
	l.baseDelay = 1 * time.Millisecond

	l.recordFail("1.1.1.1")
	l.recordFail("1.1.1.1")

	blocked1, _ := l.check("1.1.1.1")
	blocked2, _ := l.check("2.2.2.2")

	if !blocked1 {
		t.Fatal("expected 1.1.1.1 to be blocked")
	}
	if blocked2 {
		t.Fatal("expected 2.2.2.2 to NOT be blocked")
	}
}

func TestAuthLimiter_Cleanup(t *testing.T) {
	l := newAuthLimiter()
	l.resetAfter = 1 * time.Millisecond
	l.baseDelay = 1 * time.Millisecond

	l.recordFail("1.2.3.4")
	time.Sleep(5 * time.Millisecond)
	l.cleanup()

	l.mu.Lock()
	count := len(l.attempts)
	l.mu.Unlock()

	if count != 0 {
		t.Fatalf("expected 0 attempts after cleanup, got %d", count)
	}
}
