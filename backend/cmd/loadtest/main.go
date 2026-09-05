// loadtest 是一个无依赖的 HTTP 压测小工具，用于本地/预发环境打基线。
//
// 用法示例：
//
//	go build -o bin/loadtest ./cmd/loadtest
//	bin/loadtest -url http://127.0.0.1:18081/health -d 30s -c 20
//	bin/loadtest -url http://127.0.0.1:18081/api/v1/admin/dashboard/overview \
//	  -d 30s -c 8 -header "Authorization: Bearer <token>"
//
// 模式：-n 指定总请求数；为 0 时按 -d 时长压测。
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type stats struct {
	mu        sync.Mutex
	latencies []time.Duration
	statuses  map[int]int64
	errors    int64
}

func (s *stats) record(d time.Duration, code int) {
	s.mu.Lock()
	s.latencies = append(s.latencies, d)
	s.statuses[code]++
	s.mu.Unlock()
}

func (s *stats) mean() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.latencies) == 0 {
		return 0
	}
	var sum time.Duration
	for _, d := range s.latencies {
		sum += d
	}
	return sum / time.Duration(len(s.latencies))
}

func (s *stats) percentile(p float64) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.latencies) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), s.latencies...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func (s *stats) total() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return int64(len(s.latencies))
}

func main() {
	url := flag.String("url", "http://127.0.0.1:18081/health", "target URL")
	total := flag.Int("n", 0, "total requests (0 = duration mode)")
	duration := flag.Duration("d", 30*time.Second, "test duration (duration mode)")
	conc := flag.Int("c", 8, "concurrency")
	method := flag.String("method", "GET", "HTTP method")
	timeout := flag.Duration("timeout", 15*time.Second, "per-request timeout")
	bodyFile := flag.String("body", "", "request body file (optional)")
	var headers multiFlag
	flag.Var(&headers, "header", `extra header, repeatable ("Key: Value")`)
	flag.Parse()

	var bodyReader io.Reader
	if *bodyFile != "" {
		data, err := os.ReadFile(*bodyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read body: %v\n", err)
			os.Exit(1)
		}
		bodyReader = strings.NewReader(string(data))
	}

	s := &stats{statuses: map[int]int64{}}
	// 压测必须直连目标：忽略系统代理环境变量，否则流量会打到本机代理上，
	// 导致 RPS/延迟全部失真（甚至线程爆炸）。
	client := &http.Client{
		Timeout: *timeout,
		Transport: &http.Transport{
			Proxy:               nil,
			MaxIdleConnsPerHost: *conc,
			IdleConnTimeout:     30 * time.Second,
		},
	}
	var okCount, sent int64

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deadline := time.Now().Add(*duration)

	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < *conc; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if *total > 0 && atomic.AddInt64(&sent, 1) > int64(*total) {
					return
				}
				if *total == 0 && time.Now().After(deadline) {
					return
				}

				req, err := http.NewRequestWithContext(ctx, *method, *url, bodyReader)
				if err != nil {
					atomic.AddInt64(&s.errors, 1)
					continue
				}
				for _, h := range headers {
					k, v, _ := strings.Cut(h, ":")
					req.Header.Set(strings.TrimSpace(k), strings.TrimSpace(v))
				}
				if bodyReader != nil && req.GetBody != nil {
					body, _ := req.GetBody()
					req.Body = body
				}

				t0 := time.Now()
				resp, err := client.Do(req)
				elapsed := time.Since(t0)
				if err != nil {
					atomic.AddInt64(&s.errors, 1)
					// 连续失败时稍作退避，避免错误路径上疯狂空转。
					time.Sleep(10 * time.Millisecond)
					continue
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				s.record(elapsed, resp.StatusCode)
				if resp.StatusCode < 400 {
					atomic.AddInt64(&okCount, 1)
				}
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	n := s.total()
	if n == 0 {
		fmt.Println("no completed requests")
		os.Exit(2)
	}
	fmt.Printf("target=%s method=%s concurrency=%d\n", *url, *method, *conc)
	fmt.Printf("requests=%d ok=%d errors=%d elapsed=%s rps=%.1f\n",
		n, atomic.LoadInt64(&okCount), atomic.LoadInt64(&s.errors),
		elapsed.Round(time.Millisecond), float64(n)/elapsed.Seconds())
	fmt.Printf("latency avg=%s p50=%s p90=%s p99=%s\n",
		s.mean().Round(time.Microsecond),
		s.percentile(0.5).Round(time.Microsecond),
		s.percentile(0.9).Round(time.Microsecond),
		s.percentile(0.99).Round(time.Microsecond))
	s.mu.Lock()
	codes := make([]int, 0, len(s.statuses))
	for c := range s.statuses {
		codes = append(codes, c)
	}
	sort.Ints(codes)
	parts := make([]string, 0, len(codes))
	for _, c := range codes {
		parts = append(parts, fmt.Sprintf("%d:%d", c, s.statuses[c]))
	}
	s.mu.Unlock()
	fmt.Printf("status {%s}\n", strings.Join(parts, " "))
}

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ", ") }
func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
}
