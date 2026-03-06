package rclone

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// LogLine 表示 Rclone 输出的一行日志。
type LogLine struct {
	Stream string `json:"stream"` // "stdout" 或 "stderr"
	Text   string `json:"text"`
}

// ErrCancelled 表示任务被用户主动停止。
var ErrCancelled = errors.New("cancelled")

// Runner 管理 Rclone 子进程的生命周期。
type Runner struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	running bool
}

// NewRunner 创建一个新的 Rclone Runner 实例。
func NewRunner() *Runner {
	return &Runner{}
}

// IsRunning 返回当前是否有 Rclone 进程在运行。
func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Stop 终止当前正在运行的 Rclone 进程。
func (r *Runner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running || r.cancel == nil {
		return fmt.Errorf("no rclone process is running")
	}
	r.cancel()
	return nil
}

// Run 启动 rclone 备份命令，通过返回的 Channel 实时推送输出。
//
// mode:   rclone 子命令，"copy"（增量备份）或 "sync"（镜像同步）
// source: 本地源目录路径
// dest:   Rclone 远端目标路径 (如 "remote:path")
// flags:  附加的 Rclone 命令行参数
// configPath: Rclone 配置文件路径（为空则使用默认）
//
// 返回值:
//   - logCh: 实时日志 channel，进程结束后自动关闭
//   - errCh: 进程退出结果 channel，发送 nil（成功）或 error（失败/取消），仅发送一次
//   - error: 启动失败时的错误
func (r *Runner) Run(mode, source, dest string, flags []string, configPath string) (<-chan LogLine, <-chan error, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return nil, nil, fmt.Errorf("rclone is already running")
	}

	ctx, cancel := context.WithCancel(context.Background())

	args := []string{mode, source, dest, "--verbose", "--stats", "5s", "--stats-one-line"}
	if configPath != "" {
		args = append([]string{"--config", configPath}, args...)
	}
	args = append(args, flags...)

	cmd := exec.CommandContext(ctx, "rclone", args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to start rclone: %w", err)
	}

	r.cmd = cmd
	r.cancel = cancel
	r.running = true

	logCh := make(chan LogLine, 128)
	errCh := make(chan error, 1)
	modeName := "copy (增量备份)"
	if mode == "sync" {
		modeName = "sync (镜像同步)"
	}
	logCh <- LogLine{Stream: "stdout", Text: "[immichto115] 已启动 rclone " + modeName + "，正在扫描文件差异..."}
	logCh <- LogLine{Stream: "stdout", Text: fmt.Sprintf("[immichto115] 源目录: %s", source)}
	logCh <- LogLine{Stream: "stdout", Text: fmt.Sprintf("[immichto115] 目标目录: %s", dest)}

	// 将 stdout 和 stderr 合并输出到同一个 channel
	var wg sync.WaitGroup
	wg.Add(2)

	scanAndSend := func(pipe io.ReadCloser, stream string) {
		defer wg.Done()
		scanner := bufio.NewScanner(pipe)
		// Rclone 的进度输出有时单行很长，增大缓冲区
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			// 过滤掉重复的进度统计行，只保留有意义的输出
			if isProgressNoise(line) {
				continue
			}
			select {
			case logCh <- LogLine{Stream: stream, Text: line}:
			case <-ctx.Done():
				return
			}
		}
	}

	go scanAndSend(stdoutPipe, "stdout")
	go scanAndSend(stderrPipe, "stderr")

	// 等待所有管道读完 + 进程退出，然后关闭 channel
	go func() {
		wg.Wait()
		exitErr := cmd.Wait()

		r.mu.Lock()
		r.running = false
		r.cmd = nil
		r.cancel = nil
		r.mu.Unlock()

		if exitErr != nil {
			// context 取消不算错误（手动停止）
			if ctx.Err() != nil {
				logCh <- LogLine{Stream: "stderr", Text: "[immichto115] rclone 进程已收到停止信号，正在安全退出"}
				errCh <- ErrCancelled
			} else {
				logCh <- LogLine{Stream: "stderr", Text: fmt.Sprintf("[immichto115] rclone exited with error: %v", exitErr)}
				errCh <- exitErr
			}
		} else {
			logCh <- LogLine{Stream: "stdout", Text: "[immichto115] rclone sync completed successfully"}
			logCh <- LogLine{Stream: "stdout", Text: "[immichto115] 当前同步阶段已成功完成"}
			errCh <- nil
		}
		close(logCh)
		close(errCh)
	}()

	return logCh, errCh, nil
}

// 缓存 rclone 版本信息，避免每次请求都启动子进程
var (
	versionOnce   sync.Once
	cachedVersion string
	cachedVerErr  error
)

// GetVersion 获取系统上安装的 Rclone 版本号（结果会缓存，只执行一次子进程）。
func GetVersion() (string, error) {
	versionOnce.Do(func() {
		out, err := exec.Command("rclone", "version", "--check").CombinedOutput()
		if err != nil {
			// 回退：尝试不带 --check
			out, err = exec.Command("rclone", "version").CombinedOutput()
			if err != nil {
				cachedVerErr = fmt.Errorf("rclone not found or failed: %w", err)
				return
			}
		}
		cachedVersion = string(out)
	})
	return cachedVersion, cachedVerErr
}

// ResetVersionCache 清除版本缓存，下次调用 GetVersion 时重新检测。
func ResetVersionCache() {
	versionOnce = sync.Once{}
	cachedVersion = ""
	cachedVerErr = nil
}

// isProgressNoise 判断一行是否是重复的进度统计信息。
// 这些行每 0.5-5 秒输出一次，会刷屏日志，应该过滤掉。
// 重要的实际传输/错误日志不会被过滤。
func isProgressNoise(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	// 过滤 rclone 进度行（Transferred: 0 B / 0 B, Checks: xxx, Elapsed time: xxx 等）
	noisePatterns := []string{
		"Transferred:",
		"Checks:",
		"Elapsed time:",
		"Transferring:",
	}
	for _, pattern := range noisePatterns {
		if strings.HasPrefix(trimmed, pattern) {
			return true
		}
	}
	return false
}
