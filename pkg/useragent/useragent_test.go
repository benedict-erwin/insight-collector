package useragent

import (
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Test user agents for benchmarking
var testUserAgents = []string{
	// Chrome Desktop (~65%)
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",

	// Mobile Chrome (~15%)
	"Mozilla/5.0 (Linux; Android 13; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0.0.0 Mobile/15E148 Safari/604.1",

	// Safari (~10%)
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",

	// Edge (~5%)
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",

	// Firefox (~3%)
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/121.0",

	// Tablets
	"Mozilla/5.0 (iPad; CPU OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 13; SM-T870) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",

	// Bots (~1%)
	"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
	"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)",
	"GPTBot/1.0 (+https://openai.com/gptbot)",
}

// ============================================================================
// BASIC BENCHMARKS - CORE PERFORMANCE
// ============================================================================

// Benchmark single detection - most common scenario
func BenchmarkDetectSingle(b *testing.B) {
	detector := NewFastDetector()
	detector.EnableLogging(false) // Disable logging for benchmark

	// Test with Chrome desktop
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detector.Detect(ua)
	}
}

// Benchmark mixed realistic traffic
func BenchmarkDetectMixed(b *testing.B) {
	detector := NewFastDetector()
	detector.EnableLogging(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ua := testUserAgents[i%len(testUserAgents)]
		_ = detector.Detect(ua)
	}
}

// Benchmark parallel detection (simulate high concurrency)
func BenchmarkDetectParallel(b *testing.B) {
	detector := NewFastDetector()
	detector.EnableLogging(false)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ua := testUserAgents[rand.Intn(len(testUserAgents))]
			_ = detector.Detect(ua)
		}
	})
}

// Benchmark memory allocations
func BenchmarkDetectMemory(b *testing.B) {
	detector := NewFastDetector()
	detector.EnableLogging(false)
	ua := testUserAgents[0]

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detector.Detect(ua)
	}
}

// ============================================================================
// ACCURACY TESTS - CORRECTNESS VERIFICATION
// ============================================================================

// Test accuracy with expected results
func TestAccuracy(t *testing.T) {
	detector := NewFastDetector()
	detector.EnableLogging(false)

	testCases := []struct {
		name            string
		userAgent       string
		expectedType    DeviceType
		expectedOS      string
		expectedBrowser string
		expectedBot     bool
	}{
		{
			"Chrome Windows Desktop",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			Desktop, "Windows", "Chrome", false,
		},
		{
			"Safari iPhone",
			"Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1",
			Mobile, "iOS", "Safari", false,
		},
		{
			"Safari iPad",
			"Mozilla/5.0 (iPad; CPU OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1",
			Tablet, "iOS", "Safari", false,
		},
		{
			"Chrome Android Mobile",
			"Mozilla/5.0 (Linux; Android 13; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			Mobile, "Android", "Chrome", false,
		},
		{
			"Android Tablet",
			"Mozilla/5.0 (Linux; Android 13; SM-T870) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			Tablet, "Android", "Chrome", false,
		},
		{
			"Googlebot",
			"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			Unknown, "Unknown", "Unknown", true,
		},
		{
			"GPTBot (AI Crawler)",
			"GPTBot/1.0 (+https://openai.com/gptbot)",
			Unknown, "Unknown", "Unknown", true,
		},
		{
			"Edge Windows",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			Desktop, "Windows", "Edge", false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.Detect(tc.userAgent)

			if result.Type != tc.expectedType {
				t.Errorf("Expected Type=%s, got %s", tc.expectedType, result.Type)
			}
			if result.OS != tc.expectedOS {
				t.Errorf("Expected OS=%s, got %s", tc.expectedOS, result.OS)
			}
			if result.Browser != tc.expectedBrowser {
				t.Errorf("Expected Browser=%s, got %s", tc.expectedBrowser, result.Browser)
			}
			if result.IsBot != tc.expectedBot {
				t.Errorf("Expected Bot=%v, got %v", tc.expectedBot, result.IsBot)
			}
		})
	}
}

// ============================================================================
// PERFORMANCE COMPARISON
// ============================================================================

// Benchmark comparison with naive implementation
func BenchmarkNaiveVsOptimized(b *testing.B) {
	detector := NewFastDetector()
	detector.EnableLogging(false)

	// Naive implementation for comparison
	naiveDetect := func(ua string) (string, string, bool) {
		uaLower := strings.ToLower(ua)

		// Browser detection - brute force
		browser := "Unknown"
		browsers := []string{"chrome", "firefox", "safari", "edge", "opera"}
		for _, br := range browsers {
			if strings.Contains(uaLower, br) {
				browser = br
				break
			}
		}

		// OS detection - brute force
		os := "Unknown"
		oses := []string{"windows", "android", "ios", "mac", "linux"}
		for _, o := range oses {
			if strings.Contains(uaLower, o) {
				os = o
				break
			}
		}

		// Bot detection - brute force
		isBot := strings.Contains(uaLower, "bot") || strings.Contains(uaLower, "crawler")

		return browser, os, isBot
	}

	ua := testUserAgents[0]

	b.Run("Naive", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, _ = naiveDetect(ua)
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = detector.Detect(ua)
		}
	})
}

// ============================================================================
// COMPREHENSIVE TESTS
// ============================================================================

// Test performance profiling
func TestPerformanceProfile(t *testing.T) {
	detector := NewFastDetector()
	detector.EnableLogging(false)

	const iterations = 50000

	t.Logf("=== Performance Profile ===")
	t.Logf("Test iterations: %d", iterations)
	t.Logf("Test cases: %d different User-Agents", len(testUserAgents))

	// Single threaded performance
	start := time.Now()
	for i := 0; i < iterations; i++ {
		ua := testUserAgents[i%len(testUserAgents)]
		_ = detector.Detect(ua)
	}
	duration := time.Since(start)

	avgLatency := duration / iterations
	throughput := float64(iterations) / duration.Seconds()

	t.Logf("--- Single Thread Performance ---")
	t.Logf("Total time: %v", duration)
	t.Logf("Average per detection: %v", avgLatency)
	t.Logf("Detections per second: %.0f", throughput)

	// Performance rating
	if avgLatency < 500*time.Nanosecond {
		t.Logf("Rating: 🚀 EXCELLENT (<%v)", 500*time.Nanosecond)
	} else if avgLatency < 1*time.Microsecond {
		t.Logf("Rating: ✅ GOOD (<%v)", 1*time.Microsecond)
	} else {
		t.Logf("Rating: ⚠️ NEEDS OPTIMIZATION (>%v)", 1*time.Microsecond)
	}

	// Memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	t.Logf("--- Memory Usage ---")
	t.Logf("Allocated memory: %d KB", m.Alloc/1024)
	t.Logf("System memory: %d KB", m.Sys/1024)
	t.Logf("GC cycles: %d", m.NumGC)

	// Assertions for performance
	if avgLatency > 2*time.Microsecond {
		t.Errorf("Performance degradation: average latency %v > 2μs", avgLatency)
	}

	if throughput < 50000 { // Minimum 50K RPS
		t.Errorf("Throughput too low: %.0f < 50K RPS", throughput)
	}
}

// Test cache efficiency
func TestCacheEfficiency(t *testing.T) {
	detector := NewFastDetector()

	// Test cache building
	if !detector.cacheBuilt {
		t.Error("Cache should be built during initialization")
	}

	// Test cache content
	if len(detector.mobileOS) == 0 {
		t.Error("Mobile OS cache should not be empty")
	}

	if len(detector.desktopOS) == 0 {
		t.Error("Desktop OS cache should not be empty")
	}

	// Test cache rebuild
	detector.RebuildCaches()
	if !detector.cacheBuilt {
		t.Error("Cache should be rebuilt after RebuildCaches()")
	}

	t.Logf("Cache efficiency: %d mobile OS, %d desktop OS patterns cached",
		len(detector.mobileOS), len(detector.desktopOS))
}

// Test concurrent access safety
func TestConcurrentSafety(t *testing.T) {
	detector := NewFastDetector()
	detector.EnableLogging(false)

	const numRoutines = 100
	const iterationsPerRoutine = 100

	done := make(chan bool, numRoutines)
	errors := make(chan error, numRoutines)

	// Start concurrent goroutines
	for i := 0; i < numRoutines; i++ {
		go func(routineID int) {
			defer func() {
				if r := recover(); r != nil {
					errors <- fmt.Errorf("routine %d panicked: %v", routineID, r)
					return
				}
				done <- true
			}()

			for j := 0; j < iterationsPerRoutine; j++ {
				ua := testUserAgents[(routineID*iterationsPerRoutine+j)%len(testUserAgents)]
				result := detector.Detect(ua)

				// Basic validation
				if result == nil {
					errors <- fmt.Errorf("routine %d got nil result", routineID)
					return
				}
			}
		}(i)
	}

	// Wait for completion
	completedRoutines := 0
	for completedRoutines < numRoutines {
		select {
		case <-done:
			completedRoutines++
		case err := <-errors:
			t.Fatal(err)
		case <-time.After(30 * time.Second):
			t.Fatal("Test timeout - possible deadlock")
		}
	}

	t.Logf("Concurrent safety test passed: %d goroutines x %d iterations",
		numRoutines, iterationsPerRoutine)
}

// Test version detection accuracy
func TestVersionDetection(t *testing.T) {
	detector := NewFastDetector()
	detector.EnableLogging(false)

	versionTestCases := []struct {
		name                   string
		userAgent              string
		expectedOSVersion      string
		expectedBrowserVersion string
	}{
		{
			"iOS Version",
			"Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 Safari/604.1",
			"17.1", // Should convert _ to .
			"",
		},
		{
			"Android Version",
			"Mozilla/5.0 (Linux; Android 13; SM-G991B) AppleWebKit/537.36 Chrome/120.0.4472.120",
			"13",
			"",
		},
		{
			"Windows Version",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0",
			"10.0",
			"",
		},
		{
			"Chrome Version",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.4472.124 Safari/537.36",
			"10.0",
			"120.0.4472.124",
		},
	}

	for _, tc := range versionTestCases {
		t.Run(tc.name, func(t *testing.T) {
			result := detector.Detect(tc.userAgent)

			if tc.expectedOSVersion != "" && result.OSVersion != tc.expectedOSVersion {
				t.Errorf("Expected OS version=%s, got %s", tc.expectedOSVersion, result.OSVersion)
			}

			if tc.expectedBrowserVersion != "" && result.BrowserVersion != tc.expectedBrowserVersion {
				t.Errorf("Expected browser version=%s, got %s", tc.expectedBrowserVersion, result.BrowserVersion)
			}
		})
	}
}

// ============================================================================
// DEMO & EXAMPLE OUTPUT
// ============================================================================

// Test example usage for demo
func TestExampleUsage(t *testing.T) {
	detector := NewFastDetector()
	detector.EnableLogging(false)

	fmt.Println("\n=== FastDeviceDetector Demo ===")

	examples := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) Safari/604.1",
		"Mozilla/5.0 (iPad; CPU OS 17_1 like Mac OS X) Safari/604.1",
		"GPTBot/1.0 (+https://openai.com/gptbot)",
	}

	for i, ua := range examples {
		result := detector.Detect(ua)

		fmt.Printf("\nExample %d:\n", i+1)
		fmt.Printf("  UA: %s...\n", ua[:min(len(ua), 50)])
		fmt.Printf("  Type: %s\n", result.Type)

		osInfo := result.OS
		if result.OSVersion != "" {
			osInfo += " " + result.OSVersion
		}
		fmt.Printf("  OS: %s\n", osInfo)

		browserInfo := result.Browser
		if result.BrowserVersion != "" {
			browserInfo += " " + result.BrowserVersion
		}
		fmt.Printf("  Browser: %s\n", browserInfo)

		if result != nil {
			fmt.Printf("  Bot: %v\n", result.IsBot)
		}

		// Validate tidak ada nil result
		if result == nil {
			t.Error("Detection result should not be nil")
		}
	}

	fmt.Println("\n✅ All examples processed successfully!")
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
