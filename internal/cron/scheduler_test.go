package cron

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerStartStop(t *testing.T) {
	var count int32
	s := NewScheduler(func() {
		atomic.AddInt32(&count, 1)
	})

	// 每秒触发
	if err := s.Start("*/1 * * * * *"); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if !s.IsRunning() {
		t.Fatal("expected scheduler to be running")
	}
	if s.Expression() != "*/1 * * * * *" {
		t.Fatalf("Expression() = %q", s.Expression())
	}

	// 等待触发
	time.Sleep(1500 * time.Millisecond)
	if atomic.LoadInt32(&count) == 0 {
		t.Fatal("expected at least one trigger")
	}

	s.Stop()
	if s.IsRunning() {
		t.Fatal("expected scheduler to be stopped")
	}
}

func TestSchedulerNextRun(t *testing.T) {
	s := NewScheduler(func() {})
	if s.NextRun() != "" {
		t.Fatal("expected empty NextRun when not running")
	}
	if err := s.Start("0 0 0 1 1 *"); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	defer s.Stop()

	next := s.NextRun()
	if next == "" {
		t.Fatal("expected non-empty NextRun")
	}
}

func TestSchedulerIdempotentStart(t *testing.T) {
	s := NewScheduler(func() {})

	if err := s.Start("*/1 * * * * *"); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	defer s.Stop()

	// 再次 Start 应该无缝替换，不报错
	if err := s.Start("*/2 * * * * *"); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	if s.Expression() != "*/2 * * * * *" {
		t.Fatalf("expected updated expression, got %q", s.Expression())
	}
}

func TestSchedulerStopWithoutStart(t *testing.T) {
	s := NewScheduler(func() {})
	// 不应 panic
	s.Stop()
	if s.IsRunning() {
		t.Fatal("should not be running")
	}
}
