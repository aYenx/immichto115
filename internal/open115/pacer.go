package open115

import (
	"context"
	"log"
	"sync"
	"time"
)

// Pacer 是自适应速率限制器，类似 rclone 的 pacer 机制。
// 核心思想：
//   - 每次 API 调用前通过时间槽预留机制排队等待
//   - 成功时逐渐缩短间隔（加速）
//   - 遇到限速错误时大幅增加间隔（退避）
//   - 间隔有上下界限制
//
// 并发安全：多个 goroutine 可同时调用 Wait，通过时间槽预留
// 自动排成有序队列，避免突发穿透。
type Pacer struct {
	mu          sync.Mutex
	minInterval time.Duration // 最小间隔（最快速度）
	maxInterval time.Duration // 最大间隔（最慢速度）
	interval    time.Duration // 当前间隔
	decayFactor float64       // 成功后缩短比例，如 0.8 表示每次成功后间隔变为 80%
	burstFactor float64       // 限速后放大比例，如 3.0 表示间隔变为 3 倍
	nextSlot    time.Time     // 下一个可用的 API 调用时间槽
	consecutive int           // 连续成功次数（用于加速判断）
}

// PacerOption 用于自定义 Pacer 参数。
type PacerOption func(*Pacer)

func WithMinInterval(d time.Duration) PacerOption {
	return func(p *Pacer) { p.minInterval = d }
}

func WithMaxInterval(d time.Duration) PacerOption {
	return func(p *Pacer) { p.maxInterval = d }
}

func WithDecayFactor(f float64) PacerOption {
	return func(p *Pacer) { p.decayFactor = f }
}

func WithBurstFactor(f float64) PacerOption {
	return func(p *Pacer) { p.burstFactor = f }
}

// NewPacer 创建一个自适应速率限制器。
// 默认参数：最小间隔 50ms，最大间隔 60s，初始间隔 100ms。
func NewPacer(opts ...PacerOption) *Pacer {
	p := &Pacer{
		minInterval: 50 * time.Millisecond,
		maxInterval: 60 * time.Second,
		interval:    100 * time.Millisecond,
		decayFactor: 0.8,
		burstFactor: 3.0,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Wait 在 API 调用前预留一个时间槽并等待。
// 多个 goroutine 同时调用时，会自动排成有序队列：
//
//	Worker A → t=0ms
//	Worker B → t=50ms
//	Worker C → t=100ms
//	Worker D → t=150ms
//
// 返回 error 仅在 context 被取消时。
func (p *Pacer) Wait(ctx context.Context) error {
	p.mu.Lock()
	now := time.Now()

	// 计算本次调用的时间槽
	earliest := p.nextSlot
	if earliest.Before(now) {
		earliest = now
	}

	// 预留此时间槽，下一个调用者排在后面
	p.nextSlot = earliest.Add(p.interval)
	p.mu.Unlock()

	// 锁外等待到预留的时间槽
	wait := earliest.Sub(now)
	if wait <= 0 {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(wait):
		return nil
	}
}

// EndCall 在 API 调用完成后更新速率。
// 传入 err 为 nil 表示调用成功，否则检查是否为限速错误。
func (p *Pacer) EndCall(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err == nil {
		p.consecutive++
		// 连续成功 1 次以上即开始加速
		if p.consecutive >= 1 {
			p.interval = time.Duration(float64(p.interval) * p.decayFactor)
			if p.interval < p.minInterval {
				p.interval = p.minInterval
			}
		}
		return
	}

	if IsRateLimitedError(err) {
		p.consecutive = 0
		p.interval = time.Duration(float64(p.interval) * p.burstFactor)
		if p.interval > p.maxInterval {
			p.interval = p.maxInterval
		}
		log.Printf("[open115-pacer] rate limited, interval increased to %s", p.interval)
	}
	// 非限速错误不影响间隔
}

// Interval 返回当前间隔（用于日志等）。
func (p *Pacer) Interval() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.interval
}

// Call 通过 pacer 执行一个 API 调用，自带等待 + 重试。
// maxRetries 为最大重试次数（不含首次调用）。
func Call[T any](ctx context.Context, p *Pacer, label string, maxRetries int, fn func() (T, error)) (T, error) {
	var zero T
	for i := 0; i <= maxRetries; i++ {
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		if err := p.Wait(ctx); err != nil {
			return zero, err
		}
		value, err := fn()
		p.EndCall(err)

		if err == nil {
			return value, nil
		}
		if !IsRateLimitedError(err) || i == maxRetries {
			return zero, err
		}
		log.Printf("[open115-pacer] %s rate limited (attempt %d/%d), next interval %s",
			label, i+1, maxRetries, p.Interval())
	}
	return zero, nil // unreachable
}

// CallNoReturn 同 Call 但用于无返回值的函数。
func CallNoReturn(ctx context.Context, p *Pacer, label string, maxRetries int, fn func() error) error {
	_, err := Call(ctx, p, label, maxRetries, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}
