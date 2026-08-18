package browser

import (
	"sync"
	"time"
)

// SpeedTestFunc 执行单个代理测速的函数类型
type SpeedTestFunc func(proxyId string) (ok bool, latencyMs int64, err string)

// ProxySpeedScheduler 代理测速定时调度器
type ProxySpeedScheduler struct {
	dao       ProxyDAO
	testFn    SpeedTestFunc
	interval  time.Duration
	concLimit int
	stopCh    chan struct{}
	mu        sync.Mutex
	running   bool
	testing   bool
}

const (
	DefaultProxySpeedInterval     = 30 * time.Minute
	DefaultProxySpeedInitialDelay = 2 * time.Minute
	DefaultProxySpeedConcurrency  = 2
)

// NewProxySpeedScheduler 创建调度器，interval 为测速间隔，concLimit 为并发数
func NewProxySpeedScheduler(dao ProxyDAO, testFn SpeedTestFunc, interval time.Duration, concLimit int) *ProxySpeedScheduler {
	if interval <= 0 {
		interval = DefaultProxySpeedInterval
	}
	if concLimit <= 0 {
		concLimit = DefaultProxySpeedConcurrency
	}
	return &ProxySpeedScheduler{
		dao:       dao,
		testFn:    testFn,
		interval:  interval,
		concLimit: concLimit,
		stopCh:    make(chan struct{}),
	}
}

// Start 启动定时任务（非阻塞）
func (s *ProxySpeedScheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	s.running = true
	go s.loop()
}

// Stop 停止定时任务
func (s *ProxySpeedScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)
}

// RunOnce 立即执行一轮测速（可手动触发）
func (s *ProxySpeedScheduler) RunOnce() {
	go s.runAll()
}

func (s *ProxySpeedScheduler) loop() {
	// 启动后延迟一段时间跑第一轮，避免启动阶段频繁拉起代理内核。
	select {
	case <-time.After(DefaultProxySpeedInitialDelay):
	case <-s.stopCh:
		return
	}
	s.runAll()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.runAll()
		case <-s.stopCh:
			return
		}
	}
}

func (s *ProxySpeedScheduler) runAll() {
	if !s.beginRun() {
		return
	}
	defer s.finishRun()

	proxies, err := s.dao.List()
	if err != nil || len(proxies) == 0 {
		return
	}

	sem := make(chan struct{}, s.concLimit)
	var wg sync.WaitGroup

	for _, p := range proxies {
		// 跳过直连（无意义测速）
		if p.ProxyConfig == "direct://" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(proxyId string) {
			defer wg.Done()
			defer func() { <-sem }()

			ok, latencyMs, _ := s.testFn(proxyId)
			testedAt := time.Now().Format(time.RFC3339)
			_ = s.dao.UpdateSpeedResult(proxyId, ok, latencyMs, testedAt)
		}(p.ProxyId)
	}
	wg.Wait()
}

func (s *ProxySpeedScheduler) beginRun() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.testing {
		return false
	}
	s.testing = true
	return true
}

func (s *ProxySpeedScheduler) finishRun() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.testing = false
}
