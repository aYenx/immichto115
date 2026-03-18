package api

import (
	"net/http"
	"sync"
	"time"
)

// authLimiter 实现基于 IP 的认证限速与暴力破解防护。
//
// 原理:
//   - 每个 IP 维护一个连续失败计数器
//   - 成功认证或超过 resetAfter 则重置计数
//   - 失败次数越多，强制等待越长（渐进式延迟）
//   - 超过 maxFail 次被锁定 lockoutDuration
type authLimiter struct {
	mu       sync.Mutex
	attempts map[string]*attemptState

	// maxFail 连续失败次数触发锁定
	maxFail int
	// lockoutDuration 锁定时长
	lockoutDuration time.Duration
	// resetAfter 距上次失败超过该时间后重置
	resetAfter time.Duration
	// baseDelay 第一次失败后的基础延迟
	baseDelay time.Duration
}

type attemptState struct {
	failures int
	lastFail time.Time
}

func newAuthLimiter() *authLimiter {
	return &authLimiter{
		attempts:        make(map[string]*attemptState),
		maxFail:         10,
		lockoutDuration: 15 * time.Minute,
		resetAfter:      30 * time.Minute,
		baseDelay:       500 * time.Millisecond,
	}
}

// check 检查 IP 是否被限速。返回 true 表示请求被拒绝，retryAfter 为建议等待秒数。
func (l *authLimiter) check(ip string) (blocked bool, retryAfter int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	state, ok := l.attempts[ip]
	if !ok {
		return false, 0
	}

	// 距上次失败超过 resetAfter 则清除
	if time.Since(state.lastFail) > l.resetAfter {
		delete(l.attempts, ip)
		return false, 0
	}

	// 超过 maxFail 则锁定
	if state.failures >= l.maxFail {
		remaining := l.lockoutDuration - time.Since(state.lastFail)
		if remaining > 0 {
			return true, int(remaining.Seconds()) + 1
		}
		// 锁定到期
		delete(l.attempts, ip)
		return false, 0
	}

	return false, 0
}

// recordFail 记录一次失败。返回渐进延迟时长。
func (l *authLimiter) recordFail(ip string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	state, ok := l.attempts[ip]
	if !ok {
		state = &attemptState{}
		l.attempts[ip] = state
	}

	// 距上次失败超过 resetAfter 则重新计数
	if time.Since(state.lastFail) > l.resetAfter {
		state.failures = 0
	}

	state.failures++
	state.lastFail = time.Now()

	// 渐进延迟: baseDelay * 2^(failures-1)，封顶 10s
	delay := l.baseDelay * (1 << (state.failures - 1))
	const maxDelay = 10 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}

// recordSuccess 清除该 IP 的失败记录。
func (l *authLimiter) recordSuccess(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, ip)
}

// cleanup 清理过期的 IP 记录（可定期调用）。
func (l *authLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for ip, state := range l.attempts {
		if now.Sub(state.lastFail) > l.resetAfter {
			delete(l.attempts, ip)
		}
	}
}

// rateLimitMiddleware 返回一个检查被锁定 IP 的中间件。
func rateLimitMiddleware(limiter *authLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			blocked, retryAfter := limiter.check(r.RemoteAddr)
			if blocked {
				w.Header().Set("Retry-After", string(rune(retryAfter)))
				http.Error(w, "Too many failed attempts, please try again later", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
