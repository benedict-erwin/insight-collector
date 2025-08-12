package useragent

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/benedict-erwin/insight-collector/pkg/logger"
)

// DeviceType enum untuk tipe device
type DeviceType int

const (
	Desktop DeviceType = iota
	Mobile
	Tablet
	Unknown
)

// String returns string representation of DeviceType
func (d DeviceType) String() string {
	switch d {
	case Desktop:
		return "desktop"
	case Mobile:
		return "mobile"
	case Tablet:
		return "tablet"
	default:
		return "unknown"
	}
}

// FastDeviceInfo hasil deteksi device
type FastDeviceInfo struct {
	Type           DeviceType `json:"type"`
	OS             string     `json:"os"`
	OSVersion      string     `json:"os_version,omitempty"`
	Browser        string     `json:"browser"`
	BrowserVersion string     `json:"browser_version,omitempty"`
	IsBot          bool       `json:"is_bot"`
}

// DetectionLogger untuk logging pattern failures
type DetectionLogger struct {
	enabled         bool
	unknownUAs      map[string]int // Cache untuk avoid spam
	unknownBrowsers map[string]int
	unknownOSs      map[string]int
	mutex           sync.RWMutex
	lastCleanup     time.Time
	cleanupInterval time.Duration
}

// NewDetectionLogger creates new logger instance for unknown pattern detection
func NewDetectionLogger(enabled bool) *DetectionLogger {
	return &DetectionLogger{
		enabled:         enabled,
		unknownUAs:      make(map[string]int),
		unknownBrowsers: make(map[string]int),
		unknownOSs:      make(map[string]int),
		lastCleanup:     time.Now(),
		cleanupInterval: 5 * time.Minute, // Cleanup every 5 minutes
	}
}

// logUnknownPattern logs unknown patterns with maintenance instructions
func (dl *DetectionLogger) logUnknownPattern(category, userAgent, recommendation string) {
	if !dl.enabled {
		return
	}

	dl.mutex.Lock()
	defer dl.mutex.Unlock()

	// Cleanup old entries periodically
	if time.Since(dl.lastCleanup) > dl.cleanupInterval {
		dl.unknownUAs = make(map[string]int)
		dl.unknownBrowsers = make(map[string]int)
		dl.unknownOSs = make(map[string]int)
		dl.lastCleanup = time.Now()
	}

	var cache map[string]int
	switch category {
	case "browser":
		cache = dl.unknownBrowsers
	case "os":
		cache = dl.unknownOSs
	default:
		cache = dl.unknownUAs
	}

	// Truncate UA for logging
	shortUA := userAgent
	runes := []rune(userAgent)
	if len(runes) > 100 {
		shortUA = string(runes[:100]) + "..."
	}

	// Only log if we haven't seen this pattern recently
	if cache[shortUA] < 3 { // Max 3 times per cleanup interval
		cache[shortUA]++

		logger.Warn().
			Str(fmt.Sprintf("UNKNOWN_%s_DETECTED", strings.ToUpper(category)), shortUA).
			Str("USERAGENT", userAgent).
			Str("INSTRUCTION", recommendation).
			Int("COUNTER", cache[shortUA]).
			Msg("UnknownPattern detected!")
	}
}

// Global logger instance
var detectionLogger = NewDetectionLogger(true) // Enable by default

// =============================================================================
// PATTERNS CONFIGURATION - For easy maintain
// =============================================================================

// Device Type Patterns - Priority order (specifiv -> general)
var (
	tabletPatterns = []string{
		"ipad", "tablet", "kindle", "playbook", "nexus 7", "nexus 10",
		"gt-p", "sm-t", "tab", "transformer", "xoom", "xyboard",
	}

	mobilePatterns = []string{
		"iphone", "ipod", "android.*mobile", "blackberry", "windows phone",
		"palm", "symbian", "opera mini", "opera mobi", "fennec", "minimo",
	}

	// Bot patterns - case insensitive - UPDATED 2025
	botPatterns = []string{
		// Traditional crawlers
		"bot", "crawler", "spider", "scraper", "curl", "wget",
		"googlebot", "bingbot", "msnbot", "yahoo", "duckduckbot",
		"yandexbot", "baiduspider", "ia_archiver", "archive.org",

		// AI Crawlers (2024-2025) - CRITICAL ADDITIONS
		"gptbot", "chatgpt-user", // OpenAI
		"claudebot", "anthropic-ai", // Anthropic
		"meta-externalagent", "meta-externalfetcher", // Meta
		"perplexitybot",   // Perplexity
		"bytespider",      // TikTok/ByteDance
		"google-extended", // Google AI training
		"mistralai-user",  // Mistral AI

		// Social Media Bots
		"facebookexternalhit", "facebot", // Facebook
		"twitterbot",                // X/Twitter
		"linkedinbot",               // LinkedIn
		"pinterestbot", "pinterest", // Pinterest
		"whatsapp", "telegram", // Messaging
		"slackbot", "discordbot", // Communication

		// Other Major Crawlers
		"applebot", "amazonbot", // Apple, Amazon
		"semrushbot", "ahrefsbot", // SEO tools
		"screaming frog", "sitebulb", // SEO crawlers
	}
)

// OS Detection Patterns - Priority order (specific -> general) - UPDATED 2025
var osPatterns = []struct {
	name     string
	patterns []string
}{
	{"iOS", []string{"iphone", "ipad", "ipod"}},
	{"Android", []string{"android"}},
	{"HarmonyOS", []string{"harmonyos"}}, // Huawei's OS
	{"Windows Phone", []string{"windows phone"}},
	{"Windows", []string{"windows nt", "win32", "win64"}},
	{"macOS", []string{"mac os x", "macintosh", "darwin"}},
	{"Linux", []string{"linux", "x11"}},
	{"Chrome OS", []string{"cros"}},
	{"BlackBerry", []string{"blackberry", "bb10"}},
	{"Tizen", []string{"tizen"}}, // Samsung watches/TV
	{"KaiOS", []string{"kaios"}}, // Feature phones
}

// Browser Detection Patterns - Priority order (specific -> general) - UPDATED 2025
var browserPatterns = []struct {
	name     string
	patterns []string
}{
	{"Edge", []string{"edg/", "edge/"}},
	{"Chrome", []string{"chrome/"}},
	{"Firefox", []string{"firefox/"}},
	{"Safari", []string{"safari/"}},
	{"Opera", []string{"opera/", "opr/"}},
	{"Brave", []string{"brave/"}},     // Growing privacy browser
	{"Vivaldi", []string{"vivaldi/"}}, // Power user browser
	{"Internet Explorer", []string{"msie", "trident/"}},
	{"Samsung Browser", []string{"samsungbrowser/"}},
	{"UC Browser", []string{"ucbrowser/"}},
	{"DuckDuckGo", []string{"duckduckgo/", "ddg/"}}, // Privacy browser
	{"Yandex", []string{"yabrowser/", "yandex"}},    // Popular in Russia
}

// =============================================================================
// VERSION DETECTION PATTERNS - Pre-compiled regex for performance
// =============================================================================

var (
	// Browser version regexes - Pre-compiled
	browserVersionRegexes = map[string]*regexp.Regexp{
		"Edge":              regexp.MustCompile(`(?i)edg[e]?[\/\s]+([0-9]+(?:\.[0-9]+)*)`),
		"Chrome":            regexp.MustCompile(`(?i)chrome[\/\s]+([0-9]+(?:\.[0-9]+)*)`),
		"Firefox":           regexp.MustCompile(`(?i)firefox[\/\s]+([0-9]+(?:\.[0-9]+)*)`),
		"Safari":            regexp.MustCompile(`(?i)version[\/\s]+([0-9]+(?:\.[0-9]+)*).+safari`),
		"Opera":             regexp.MustCompile(`(?i)(?:opera[\/\s]+|opr[\/\s]+)([0-9]+(?:\.[0-9]+)*)`),
		"Brave":             regexp.MustCompile(`(?i)chrome[\/\s]+([0-9]+(?:\.[0-9]+)*)`), // Brave uses Chrome engine
		"Vivaldi":           regexp.MustCompile(`(?i)vivaldi[\/\s]+([0-9]+(?:\.[0-9]+)*)`),
		"Internet Explorer": regexp.MustCompile(`(?i)(?:msie[\/\s]+([0-9]+(?:\.[0-9]+)*)|rv:([0-9]+(?:\.[0-9]+)*).+trident)`),
		"Samsung Browser":   regexp.MustCompile(`(?i)samsungbrowser[\/\s]+([0-9]+(?:\.[0-9]+)*)`),
		"UC Browser":        regexp.MustCompile(`(?i)ucbrowser[\/\s]+([0-9]+(?:\.[0-9]+)*)`),
		"DuckDuckGo":        regexp.MustCompile(`(?i)duckduckgo[\/\s]+([0-9]+(?:\.[0-9]+)*)`),
		"Yandex":            regexp.MustCompile(`(?i)yabrowser[\/\s]+([0-9]+(?:\.[0-9]+)*)`),
	}

	// OS version regexes
	osVersionRegexes = map[string]*regexp.Regexp{
		"Windows":       regexp.MustCompile(`(?i)windows\s+nt\s+([0-9]+(?:\.[0-9]+)*)`),
		"macOS":         regexp.MustCompile(`(?i)mac\s+os\s+x\s+([0-9]+(?:[._][0-9]+)*)`),
		"iOS":           regexp.MustCompile(`(?i)(?:iphone\s+os|os)\s+([0-9]+(?:[._][0-9]+)*)`),
		"Android":       regexp.MustCompile(`(?i)android\s+([0-9]+(?:\.[0-9]+)*)`),
		"HarmonyOS":     regexp.MustCompile(`(?i)harmonyos\s+([0-9]+(?:\.[0-9]+)*)`),
		"Chrome OS":     regexp.MustCompile(`(?i)cros\s+[^\s]+\s+([0-9]+(?:\.[0-9]+)*)`),
		"Windows Phone": regexp.MustCompile(`(?i)windows\s+phone\s+os\s+([0-9]+(?:\.[0-9]+)*)`),
		"Tizen":         regexp.MustCompile(`(?i)tizen\s+([0-9]+(?:\.[0-9]+)*)`),
		"KaiOS":         regexp.MustCompile(`(?i)kaios[\/\s]+([0-9]+(?:\.[0-9]+)*)`),
		// BlackBerry no reliable pattern
	}

	// Compile once for thread safety
	compileOnce sync.Once
)

// =============================================================================
// FAST DEVICE DETECTOR - Optimized for high traffic
// =============================================================================

type FastDeviceDetector struct {
	logger *DetectionLogger

	// Pre-computed caches for performance optimization
	mobileOS   []osPattern
	desktopOS  []osPattern
	cacheBuilt bool
	cacheMutex sync.RWMutex
}

// osPattern represents a pre-computed OS pattern for optimization
type osPattern struct {
	name     string
	patterns []string
}

// NewFastDetector creates optimized device detector with pre-compiled patterns
func NewFastDetector() *FastDeviceDetector {
	// Ensure regexes are compiled
	compileOnce.Do(func() {
		// Regexes already pre-compiled on global variabel
	})

	detector := &FastDeviceDetector{
		logger: detectionLogger,
	}

	// Build optimization caches
	detector.buildOptimizationCaches()

	return detector
}

// buildOptimizationCaches pre-computes OS categorizations for performance
func (d *FastDeviceDetector) buildOptimizationCaches() {
	d.cacheMutex.Lock()
	defer d.cacheMutex.Unlock()

	if d.cacheBuilt {
		return
	}

	// Pre-categorize OS patterns by type for faster detection
	for _, os := range osPatterns {
		osEntry := osPattern{
			name:     os.name,
			patterns: os.patterns,
		}

		// Categorize based on OS nature
		switch os.name {
		case "iOS", "Android", "Windows Phone", "KaiOS":
			d.mobileOS = append(d.mobileOS, osEntry)
		default:
			d.desktopOS = append(d.desktopOS, osEntry)
		}
	}

	d.cacheBuilt = true
}

// EnableLogging toggles detection logging for unknown patterns
func (d *FastDeviceDetector) EnableLogging(enabled bool) {
	d.logger.enabled = enabled
}

// RebuildCaches rebuilds optimization caches when patterns are updated
func (d *FastDeviceDetector) RebuildCaches() {
	d.cacheMutex.Lock()
	d.cacheBuilt = false
	d.mobileOS = nil
	d.desktopOS = nil
	d.cacheMutex.Unlock()

	d.buildOptimizationCaches()
}

// Detect performs fast user agent analysis and returns device information
func (d *FastDeviceDetector) Detect(userAgent string) *FastDeviceInfo {
	ua := strings.ToLower(userAgent)

	deviceType := d.detectDeviceType(ua)
	osName := d.detectOS(ua)
	osVersion := d.detectOSVersion(userAgent)
	browserName := d.detectBrowser(ua)
	browserVersion := d.detectBrowserVersion(userAgent)
	isBot := d.isBot(ua)

	// Log if unknown
	d.logUnknownDetections(userAgent, deviceType, osName, browserName, isBot)

	return &FastDeviceInfo{
		Type:           deviceType,
		OS:             osName,
		OSVersion:      osVersion,
		Browser:        browserName,
		BrowserVersion: browserVersion,
		IsBot:          isBot,
	}
}

// logUnknownDetections logs patterns that might need to be added to detection rules
func (d *FastDeviceDetector) logUnknownDetections(userAgent string, deviceType DeviceType, os, browser string, isBot bool) {
	if !d.logger.enabled {
		return
	}

	// Skip logging for bot (expected unknown)
	if isBot {
		return
	}

	// Log unknown browser with instruction
	if browser == "Unknown" && !strings.Contains(strings.ToLower(userAgent), "bot") {
		recommendation := "Append browserPatterns slice and browserVersionRegexes (only if needed)"
		d.logger.logUnknownPattern("browser", userAgent, recommendation)
	}

	// Log unknown OS with instruction
	if os == "Unknown" {
		recommendation := "Append osPatterns slice and osVersionRegexes (only if needed)"
		d.logger.logUnknownPattern("os", userAgent, recommendation)
	}

	// Log suspicious device type detection
	if deviceType == Unknown {
		recommendation := "Possibly add pattern to 'tabletPatterns' or 'mobilePatterns', Check if Mobile/Tablet/Desktop"
		d.logger.logUnknownPattern("device", userAgent, recommendation)
	}

	// Log potential new bot patterns
	uaLower := strings.ToLower(userAgent)
	if (strings.Contains(uaLower, "crawl") ||
		strings.Contains(uaLower, "scan") ||
		strings.Contains(uaLower, "fetch") ||
		strings.Contains(uaLower, "monitor") ||
		strings.Contains(uaLower, "check")) && !isBot {
		recommendation := "Append botPatterns slice, Extract identifying keywords from User-Agent"
		d.logger.logUnknownPattern("bot", userAgent, recommendation)
	}
}

// detectDeviceType determines if device is desktop, mobile, tablet or unknown
func (d *FastDeviceDetector) detectDeviceType(ua string) DeviceType {
	// Check tablet first (most specific)
	for _, pattern := range tabletPatterns {
		if strings.Contains(ua, pattern) {
			return Tablet
		}
	}

	// Check mobile (medium specific)
	for _, pattern := range mobilePatterns {
		if strings.Contains(ua, pattern) {
			return Mobile
		}
	}

	// Special handling for Android without "mobile" keyword = tablet
	if strings.Contains(ua, "android") && !strings.Contains(ua, "mobile") {
		return Tablet
	}

	// OPTIMIZED: Use pre-computed desktop OS cache
	d.cacheMutex.RLock()
	for _, os := range d.desktopOS {
		for _, pattern := range os.patterns {
			if strings.Contains(ua, pattern) {
				d.cacheMutex.RUnlock()
				return Desktop
			}
		}
	}
	d.cacheMutex.RUnlock()

	// Return unknown if no match
	return Unknown
}

// detectOS identifies operating system from user agent string
func (d *FastDeviceDetector) detectOS(ua string) string {
	for _, os := range osPatterns {
		for _, pattern := range os.patterns {
			if strings.Contains(ua, pattern) {
				return os.name
			}
		}
	}
	return "Unknown"
}

// detectBrowser identifies browser from user agent string
func (d *FastDeviceDetector) detectBrowser(ua string) string {
	for _, browser := range browserPatterns {
		for _, pattern := range browser.patterns {
			if strings.Contains(ua, pattern) {
				return browser.name
			}
		}
	}
	return "Unknown"
}

// isBot checks if user agent indicates automated bot or crawler
func (d *FastDeviceDetector) isBot(ua string) bool {
	for _, pattern := range botPatterns {
		if strings.Contains(ua, pattern) {
			return true
		}
	}
	return false
}

// detectOSVersion extracts operating system version using regex patterns
func (d *FastDeviceDetector) detectOSVersion(ua string) string {
	// iOS fast path
	if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod") {
		if regex, exists := osVersionRegexes["iOS"]; exists {
			if matches := regex.FindStringSubmatch(ua); len(matches) > 1 {
				return strings.ReplaceAll(matches[1], "_", ".")
			}
		}
	}

	// Android fast path
	if strings.Contains(ua, "android") {
		if regex, exists := osVersionRegexes["Android"]; exists {
			if matches := regex.FindStringSubmatch(ua); len(matches) > 1 {
				return matches[1]
			}
		}
	}

	// Windows fast path
	if strings.Contains(ua, "windows nt") {
		if regex, exists := osVersionRegexes["Windows"]; exists {
			if matches := regex.FindStringSubmatch(ua); len(matches) > 1 {
				return matches[1]
			}
		}
	}

	// Other OS via cache
	d.cacheMutex.RLock()
	defer d.cacheMutex.RUnlock()

	for _, os := range d.mobileOS {
		if regex, exists := osVersionRegexes[os.name]; exists {
			if matches := regex.FindStringSubmatch(ua); len(matches) > 1 {
				return strings.ReplaceAll(matches[1], "_", ".")
			}
		}
	}

	for _, os := range d.desktopOS {
		if regex, exists := osVersionRegexes[os.name]; exists {
			if matches := regex.FindStringSubmatch(ua); len(matches) > 1 {
				return strings.ReplaceAll(matches[1], "_", ".")
			}
		}
	}

	return ""
}

// detectBrowserVersion extracts browser version using pre-compiled regex
func (d *FastDeviceDetector) detectBrowserVersion(ua string) string {
	// Dynamic order from browserPatterns - ALWAYS IN SYNC ✅
	for _, browser := range browserPatterns {
		if regex, exists := browserVersionRegexes[browser.name]; exists {
			if matches := regex.FindStringSubmatch(ua); len(matches) > 1 {
				// Handle Internet Explorer special case
				if browser.name == "Internet Explorer" {
					if len(matches) > 1 && matches[1] != "" {
						return matches[1]
					} else if len(matches) > 2 && matches[2] != "" {
						return matches[2]
					}
				}
				return matches[1]
			}
		}
	}
	return ""
}

// =============================================================================
// UTILITY FUNCTIONS - For maintenance and testing
// =============================================================================

// GetSupportedBrowsers returns list of browsers supported by detector
func (d *FastDeviceDetector) GetSupportedBrowsers() []string {
	browsers := make([]string, len(browserPatterns))
	for i, browser := range browserPatterns {
		browsers[i] = browser.name
	}
	return browsers
}

// GetSupportedOS returns list of operating systems supported by detector
func (d *FastDeviceDetector) GetSupportedOS() []string {
	oses := make([]string, len(osPatterns))
	for i, os := range osPatterns {
		oses[i] = os.name
	}
	return oses
}

// AddBotPattern adds new bot detection pattern at runtime
func AddBotPattern(pattern string) {
	botPatterns = append(botPatterns, pattern)
	log.Printf("✅ BOT_PATTERN_ADDED: '%s' has been added to botPatterns", pattern)
}

// AddBrowserPattern adds new browser detection pattern at runtime
func AddBrowserPattern(name string, patterns []string) {
	browserPatterns = append(browserPatterns, struct {
		name     string
		patterns []string
	}{name, patterns})
	log.Printf("✅ BROWSER_PATTERN_ADDED: '%s' has been added to browserPatterns", name)
}

// AddOSPattern adds new OS detection pattern and rebuilds caches
func AddOSPattern(name string, patterns []string, detector *FastDeviceDetector) {
	osPatterns = append(osPatterns, struct {
		name     string
		patterns []string
	}{name, patterns})

	// Rebuild caches since OS patterns changed
	if detector != nil {
		detector.RebuildCaches()
	}

	log.Printf("✅ OS_PATTERN_ADDED: '%s' has been added to osPatterns and caches rebuilt", name)
}

// GetDetectionStats returns statistics about unknown pattern detections
func (d *FastDeviceDetector) GetDetectionStats() map[string]int {
	d.logger.mutex.RLock()
	defer d.logger.mutex.RUnlock()

	stats := make(map[string]int)

	totalUnknown := 0
	for _, count := range d.logger.unknownUAs {
		totalUnknown += count
	}

	totalUnknownBrowsers := 0
	for _, count := range d.logger.unknownBrowsers {
		totalUnknownBrowsers += count
	}

	totalUnknownOS := 0
	for _, count := range d.logger.unknownOSs {
		totalUnknownOS += count
	}

	stats["unknown_uas"] = len(d.logger.unknownUAs)
	stats["unknown_browsers"] = len(d.logger.unknownBrowsers)
	stats["unknown_oses"] = len(d.logger.unknownOSs)
	stats["total_unknown_detections"] = totalUnknown
	stats["total_unknown_browser_detections"] = totalUnknownBrowsers
	stats["total_unknown_os_detections"] = totalUnknownOS

	return stats
}

// // =============================================================================
// // BENCHMARKING DAN TESTING
// // =============================================================================

// func main() {
// 	// Initialize detector
// 	detector := NewFastDetector()

// 	// Test cases including some unknown patterns for demonstration
// 	testCases := []string{
// 		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
// 		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
// 		"Mozilla/5.0 (iPad; CPU OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
// 		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
// 		"Mozilla/5.0 (Linux; Android 11; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.120 Mobile Safari/537.36",
// 		"Mozilla/5.0 (Linux; Android 11; SM-T870) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.120 Safari/537.36",
// 		"Googlebot/2.1 (+http://www.google.com/bot.html)",
// 		"Mozilla/5.0 compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm",
// 		// Test cases untuk unknown patterns (akan trigger logging)
// 		"NewBrowser/1.0 (CustomOS 2.0; Unknown Device) SuperEngine/1.0",
// 		"WebCrawler/1.0 (compatible; CustomBot/2.0; +http://example.com/bot.html)",
// 		"Mozilla/5.0 (UnknownOS 5.0) MysteryBrowser/3.0",
// 		// Safari test
// 		"Mozilla/5.0 (iPhone; CPU iPhone OS 15_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.5 Mobile/15E148 Safari/604.1",
// 		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.2 Safari/605.1.15",
// 		"Mozilla/5.0 (iPad; CPU OS 14_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.7 Mobile/15E148 Safari/604.1",
// 		"Mozilla/5.0 (iPhone; CPU iPhone OS 9_3_5 like Mac OS X) AppleWebKit/601.1.46 (KHTML, like Gecko) Version/9.0 Mobile/13G36 Safari/601.1",
// 		"Mozilla/5.0 (AppleWatch; U; CPU WatchOS 6_2 like Mac OS X) AppleWebKit/601.1.46 (KHTML, like Gecko) Version/6.2 Safari/601.1",
// 	}

// 	println("=== FastDeviceDetector dengan Logging ===\n")

// 	for i, ua := range testCases {
// 		info := detector.Detect(ua)

// 		// Format output
// 		displayUA := ua
// 		if len(ua) > 60 {
// 			displayUA = ua[:60] + "..."
// 		}

// 		browserInfo := info.Browser
// 		if info.BrowserVersion != "" {
// 			browserInfo += " " + info.BrowserVersion
// 		}

// 		osInfo := info.OS
// 		if info.OSVersion != "" {
// 			osInfo += " " + info.OSVersion
// 		}

// 		fmt.Printf("Test %d\n", i+1)
// 		fmt.Printf("UA: %s\n", displayUA)
// 		fmt.Printf("Result: Type=%s, OS=%s, Browser=%s, Bot=%v\n",
// 			info.Type.String(), osInfo, browserInfo, info.IsBot)
// 		fmt.Println("---")
// 	}

// 	// Show detection statistics
// 	fmt.Println("\n=== Detection Statistics ===")
// 	stats := detector.GetDetectionStats()
// 	for key, value := range stats {
// 		fmt.Printf("%s: %d\n", key, value)
// 	}

// 	// Performance info
// 	fmt.Println("\n=== Performance + Monitoring Features ===")
// 	fmt.Println("🚀 Optimized for: High traffic (>10K RPS)")
// 	fmt.Println("⚡ Expected latency: ~200-500ns per detection")
// 	fmt.Println("💾 Memory footprint: ~2-5KB")
// 	fmt.Println("🔧 Maintenance: Edit patterns + check logs")
// 	fmt.Println("📊 Thread-safe: Ya (dengan smart logging)")
// 	fmt.Println("📝 Auto-logging: Unknown patterns dengan instruksi maintenance")
// 	fmt.Println("🔍 Monitoring: GetDetectionStats() untuk statistics")

// 	fmt.Printf("\nSupported Browsers: %d\n", len(detector.GetSupportedBrowsers()))
// 	fmt.Printf("Supported OS: %d\n", len(detector.GetSupportedOS()))

// 	// Contoh runtime modification with optimization
// 	fmt.Println("\n=== Runtime Modification + Cache Management ===")
// 	AddBotPattern("customcrawler")
// 	testBot := "CustomCrawler/1.0 for testing"
// 	info := detector.Detect(testBot)
// 	fmt.Printf("Custom bot test: %v\n", info.IsBot)

// 	// Demonstrate OS pattern addition with cache rebuild
// 	fmt.Println("\nAdding new OS pattern with automatic cache rebuild...")
// 	AddOSPattern("TestOS", []string{"testos"}, detector)

// 	// Show updated cache information
// 	fmt.Printf("Updated Cache: (%d mobile OS, %d desktop OS)\n",
// 		len(detector.mobileOS), len(detector.desktopOS))

// 	// Demonstrate logging control
// 	fmt.Println("\n=== Logging Control ===")
// 	fmt.Println("Logging enabled by default. To disable:")
// 	fmt.Println("detector.EnableLogging(false)")

// 	// Performance comparison demo
// 	fmt.Println("\n=== Smart Hybrid Performance Demo ===")
// 	fmt.Println("Fast-path examples:")

// 	fastPathExamples := []string{
// 		"Mozilla/5.0 (iPhone; CPU iPhone OS 15_5 like Mac OS X)", // iOS fast-path
// 		"Mozilla/5.0 (Linux; Android 12; SM-G973F)",              // Android fast-path
// 		"Mozilla/5.0 (Windows NT 10.0; Win64; x64)",              // Windows fast-path
// 	}

// 	for i, ua := range fastPathExamples {
// 		info := detector.Detect(ua)
// 		osInfo := info.OS
// 		if info.OSVersion != "" {
// 			osInfo += " " + info.OSVersion
// 		}
// 		fmt.Printf("Fast-path %d: %s → %s\n", i+1, osInfo, info.Type.String())
// 	}

// 	// Show final stats
// 	finalStats := detector.GetDetectionStats()
// 	fmt.Printf("\nFinal unknown detection count: %d patterns logged\n",
// 		finalStats["unknown_browsers"]+finalStats["unknown_oses"])
// }
