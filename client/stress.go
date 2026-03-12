package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// 压测配置
var (
	serverAddr   = flag.String("server", "localhost:8080", "Server address")
	clientCount  = flag.Int("clients", 100, "Number of concurrent clients")
	requestCount = flag.Int("requests", 10, "Requests per client")
	rampUpTime   = flag.Int("rampup", 5, "Ramp up time in seconds")
)

// 统计信息
var (
	totalRequests   int64 = 0
	successRequests int64 = 0
	failedRequests  int64 = 0
	totalLatency    int64 = 0
	startTime       time.Time
)

func main() {
	flag.Parse()

	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║       SLG Game Server - Performance Test               ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Server:         %s\n", *serverAddr)
	fmt.Printf("  Clients:        %d\n", *clientCount)
	fmt.Printf("  Requests/Client: %d\n", *requestCount)
	fmt.Printf("  Ramp Up Time:   %ds\n", *rampUpTime)
	fmt.Println()
	fmt.Println("Starting performance test...")
	fmt.Println()

	startTime = time.Now()

	var wg sync.WaitGroup
	clientResults := make(chan *ClientResult, *clientCount)

	for i := 0; i < *clientCount; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			result := runClient(clientID)
			select {
			case clientResults <- result:
			default:
				// Channel full, skip
			}
		}(i)

		if *rampUpTime > 0 {
			time.Sleep(time.Duration(*rampUpTime*1000/(*clientCount)) * time.Millisecond)
		}
	}

	// Wait for all clients to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Collect results with timeout
	timeout := time.After(120 * time.Second)
	var totalSuccess, totalFailed int
	var totalLat int64
	collected := 0

	for collected < *clientCount {
		select {
		case result := <-clientResults:
			totalSuccess += result.Success
			totalFailed += result.Failed
			totalLat += result.TotalLatency
			collected++
		case <-done:
			// All clients finished
			goto printResults
		case <-timeout:
			fmt.Println("\n⚠️  Test timeout!")
			goto printResults
		}
	}

printResults:
	printStatistics(totalSuccess, totalFailed, totalLat)
}

// ClientResult 客户端结果
type ClientResult struct {
	ClientID     int
	Success      int
	Failed       int
	TotalLatency int64
}

// runClient 运行单个客户端测试
func runClient(clientID int) *ClientResult {
	result := &ClientResult{ClientID: clientID}

	client := NewTestClient(*serverAddr)

	if err := client.Connect(); err != nil {
		log.Printf("[Client %d] Connection failed: %v", clientID, err)
		result.Failed++
		return result
	}
	defer client.Close()

	username := fmt.Sprintf("loadtest_%d_%d", clientID, time.Now().UnixNano())
	password := "password123"
	email := fmt.Sprintf("test%d@example.com", clientID)

	// 注册
	start := time.Now()
	regResp, err := client.Register(username, password, email)
	latency := time.Since(start).Milliseconds()
	result.TotalLatency += latency

	if err != nil || !regResp.Success {
		log.Printf("[Client %d] Register failed: %v", clientID, err)
		result.Failed++
	} else {
		result.Success++
		atomic.AddInt64(&successRequests, 1)
	}

	// 登录
	start = time.Now()
	loginResp, err := client.Login(username, password)
	latency = time.Since(start).Milliseconds()
	result.TotalLatency += latency

	if err != nil || !loginResp.Success {
		log.Printf("[Client %d] Login failed: %v", clientID, err)
		result.Failed++
	} else {
		result.Success++
		atomic.AddInt64(&successRequests, 1)
	}

	// 移动测试
	for i := 0; i < *requestCount; i++ {
		start = time.Now()
		x := rand.Int31n(1000)
		y := rand.Int31n(1000)
		moveResp, err := client.Move(x, y)
		latency = time.Since(start).Milliseconds()
		result.TotalLatency += latency

		if err != nil || !moveResp.Success {
			result.Failed++
			atomic.AddInt64(&failedRequests, 1)
		} else {
			result.Success++
			atomic.AddInt64(&successRequests, 1)
		}

		atomic.AddInt64(&totalRequests, 1)
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	}

	// 建造测试
	start = time.Now()
	buildResp, err := client.Build("farm", rand.Int31n(100), rand.Int31n(100))
	latency = time.Since(start).Milliseconds()
	result.TotalLatency += latency

	if err != nil || !buildResp.Success {
		result.Failed++
		atomic.AddInt64(&failedRequests, 1)
	} else {
		result.Success++
		atomic.AddInt64(&successRequests, 1)
	}

	atomic.AddInt64(&totalRequests, 1)

	return result
}

// printStatistics 打印统计信息
func printStatistics(success, failed int, totalLatency int64) {
	elapsed := time.Since(startTime)
	total := success + failed

	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║                  Test Results                          ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Duration:        %v\n", elapsed.Round(time.Second))
	fmt.Printf("Total Requests:  %d\n", total)
	fmt.Printf("Successful:      %d (%.2f%%)\n", success, float64(success)/float64(total)*100)
	fmt.Printf("Failed:          %d (%.2f%%)\n", failed, float64(failed)/float64(total)*100)
	fmt.Println()
	fmt.Printf("Throughput:      %.2f requests/second\n", float64(total)/elapsed.Seconds())
	fmt.Printf("Avg Latency:     %dms\n", totalLatency/int64(total))
	fmt.Println()

	rating := getPerformanceRating(float64(total)/elapsed.Seconds(), float64(totalLatency)/float64(total))
	fmt.Printf("Performance Rating: %s\n", rating)
	fmt.Println()
}

func getPerformanceRating(throughput, avgLatency float64) string {
	if throughput > 1000 && avgLatency < 50 {
		return "⭐⭐⭐⭐⭐ Excellent"
	} else if throughput > 500 && avgLatency < 100 {
		return "⭐⭐⭐⭐ Very Good"
	} else if throughput > 200 && avgLatency < 200 {
		return "⭐⭐⭐ Good"
	} else if throughput > 100 && avgLatency < 500 {
		return "⭐⭐ Fair"
	} else {
		return "⭐ Poor (needs optimization)"
	}
}
