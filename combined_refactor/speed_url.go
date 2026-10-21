package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const autoSpeedURLValue = "auto"

const (
	cmSpeedURL              = "cf.090227.xyz/__down?bytes=99999999"
	mobileDedicatedSpeedURL = "speed.okl.abrdns.com"
	cloudflareSpeedURL      = "speed.cloudflare.com/__down?bytes=99999999"
	ispProbeURL             = "https://cf.090227.xyz/cf.json"
)

type ispProbeInfo struct {
	ASN            int    `json:"asn"`
	ASOrganization string `json:"asOrganization"`
}

var autoSpeedURLState = struct {
	sync.RWMutex
	value string
}{value: cloudflareSpeedURL}

func resolveSpeedTestURL(rawURL string) string {
	value := strings.TrimSpace(rawURL)
	if value == "" {
		value = strings.TrimSpace(speedTestURL)
	}
	if strings.EqualFold(value, autoSpeedURLValue) || value == "自动选择" {
		return pickAutoSpeedURL()
	}
	return value
}

func pickAutoSpeedURL() string {
	value := currentAutoSpeedURLDefault()
	if available, known := isSpeedURLAvailable(value); known && !available {
		if alt := firstAvailableSpeedURL(cloudflareSpeedURL, cmSpeedURL, mobileDedicatedSpeedURL); alt != "" {
			value = alt
		} else {
			value = cloudflareSpeedURL
		}
		setAutoSpeedURLDefault(value)
	}
	if strings.TrimSpace(value) == "" {
		return cloudflareSpeedURL
	}
	return value
}

func currentAutoSpeedURLDefault() string {
	autoSpeedURLState.RLock()
	value := autoSpeedURLState.value
	autoSpeedURLState.RUnlock()
	return value
}

func resolveStartupSpeedTestURL(ctx context.Context, rawURL string) (string, ispProbeInfo, error) {
	value := strings.TrimSpace(rawURL)
	if value == "" {
		value = autoSpeedURLValue
	}
	if !isAutoSpeedURL(value) {
		return value, ispProbeInfo{}, nil
	}
	info, err := detectSpeedTestISP(ctx)
	if err != nil {
		return autoSpeedURLValue, info, err
	}
	if isChinaMobileISP(info) {

		usable := make([]string, 0, 2)
		for _, candidate := range []string{cmSpeedURL, mobileDedicatedSpeedURL} {
			if available, _ := isSpeedURLAvailable(candidate); available {
				usable = append(usable, candidate)
			}
		}
		if len(usable) == 0 {
			if available, _ := isSpeedURLAvailable(cloudflareSpeedURL); available {
				usable = []string{cloudflareSpeedURL}
			}
		}
		if len(usable) == 0 {
			setAutoSpeedURLDefault(cloudflareSpeedURL)
			return autoSpeedURLValue, info, nil
		}
		setAutoSpeedURLDefault(pickRandomSpeedURL(usable))
		return autoSpeedURLValue, info, nil
	}

	if available, known := isSpeedURLAvailable(cloudflareSpeedURL); known && !available {
		if alt := firstAvailableSpeedURL(cmSpeedURL, mobileDedicatedSpeedURL); alt != "" {
			setAutoSpeedURLDefault(alt)
			return autoSpeedURLValue, info, nil
		}
	}
	setAutoSpeedURLDefault(cloudflareSpeedURL)
	return autoSpeedURLValue, info, nil
}

func setAutoSpeedURLDefault(value string) {
	autoSpeedURLState.Lock()
	defer autoSpeedURLState.Unlock()
	if strings.TrimSpace(value) == "" {
		autoSpeedURLState.value = cloudflareSpeedURL
		return
	}
	autoSpeedURLState.value = value
}

func isAutoSpeedURL(value string) bool {
	value = strings.TrimSpace(value)
	return strings.EqualFold(value, autoSpeedURLValue) || value == "自动选择"
}

func detectSpeedTestISP(ctx context.Context) (ispProbeInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	data, err := getURLBytesWithContext(ctx, ispProbeURL)
	if err != nil {
		return ispProbeInfo{}, err
	}
	var info ispProbeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return ispProbeInfo{}, err
	}
	return info, nil
}

func isChinaMobileISP(info ispProbeInfo) bool {
	org := strings.ToLower(info.ASOrganization)
	mobileKeywords := []string{"cmi", "cmnet", "chinamobile", "china mobile", "cmcc", "mobile communications", "移动"}
	for _, keyword := range mobileKeywords {
		if strings.Contains(org, keyword) {
			return true
		}
	}
	switch info.ASN {
	case 9808, 24400, 56040, 56041, 56044:
		return true
	default:
		return false
	}
}

func pickRandomSpeedURL(urls []string) string {
	idx := 0
	if n, err := rand.Int(rand.Reader, big.NewInt(int64(len(urls)))); err == nil {
		idx = int(n.Int64())
	}
	return urls[idx]
}

func parseSpeedTestURL(rawURL string, fallbackScheme string) (*url.URL, error) {
	value := resolveSpeedTestURL(rawURL)
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("测速地址为空")
	}

	fallbackScheme = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(fallbackScheme)), "://")
	if fallbackScheme != "http" && fallbackScheme != "https" {
		fallbackScheme = "https"
	}

	lowerValue := strings.ToLower(value)
	if strings.HasPrefix(value, "//") {
		value = fallbackScheme + ":" + value
	} else if !strings.HasPrefix(lowerValue, "http://") && !strings.HasPrefix(lowerValue, "https://") {
		value = fallbackScheme + "://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return nil, err
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("不支持的测速地址协议: %s", parsed.Scheme)
	}
	if strings.TrimSpace(parsed.Hostname()) == "" {
		return nil, fmt.Errorf("测速地址缺少域名")
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed, nil
}

var speedURLAvailabilityState = struct {
	sync.RWMutex
	ready  bool
	status map[string]bool
}{status: map[string]bool{}}

func speedURLLabel(rawURL string) string {
	switch rawURL {
	case cloudflareSpeedURL:
		return "Cloudflare"
	case cmSpeedURL:
		return "CM提供"
	case mobileDedicatedSpeedURL:
		return "移动专属"
	default:
		return "自定义"
	}
}

func speedURLCandidates() []string {
	candidates := []string{cloudflareSpeedURL, cmSpeedURL, mobileDedicatedSpeedURL}
	custom := strings.TrimSpace(speedTestURL)
	if custom != "" && !isAutoSpeedURL(custom) {
		duplicate := false
		for _, existing := range candidates {
			if existing == custom {
				duplicate = true
				break
			}
		}
		if !duplicate {
			candidates = append(candidates, custom)
		}
	}
	return candidates
}

func speedURLProbeTarget(rawURL string) string {
	marker := "bytes="
	idx := strings.Index(rawURL, marker)
	if idx < 0 {
		return rawURL
	}
	end := idx + len(marker)
	for end < len(rawURL) && rawURL[end] >= '0' && rawURL[end] <= '9' {
		end++
	}
	if end == idx+len(marker) {
		return rawURL
	}
	return rawURL[:idx+len(marker)] + "1024" + rawURL[end:]
}

func checkSpeedURLAvailability(rawURL string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	parsed, err := parseSpeedTestURL(speedURLProbeTarget(rawURL), "https")
	if err != nil {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return false
	}
	resp, err := upstreamHTTPClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return false
	}
	buf := make([]byte, 4096)
	n, _ := io.ReadFull(io.LimitReader(resp.Body, int64(len(buf))), buf)
	return n > 0
}

func ensureSpeedURLAvailability() {
	speedURLAvailabilityState.Lock()
	if speedURLAvailabilityState.ready {
		speedURLAvailabilityState.Unlock()
		return
	}
	speedURLAvailabilityState.ready = true
	speedURLAvailabilityState.Unlock()

	candidates := speedURLCandidates()
	results := make(map[string]bool, len(candidates))
	var resultMutex sync.Mutex
	var wg sync.WaitGroup
	for _, candidate := range candidates {
		wg.Add(1)
		go func(target string) {
			defer wg.Done()
			available := checkSpeedURLAvailability(target)
			resultMutex.Lock()
			results[target] = available
			resultMutex.Unlock()
		}(candidate)
	}
	wg.Wait()

	speedURLAvailabilityState.Lock()
	for target, available := range results {
		speedURLAvailabilityState.status[target] = available
	}
	speedURLAvailabilityState.Unlock()
}

func isSpeedURLAvailable(rawURL string) (available bool, known bool) {
	speedURLAvailabilityState.RLock()
	defer speedURLAvailabilityState.RUnlock()
	available, known = speedURLAvailabilityState.status[rawURL]
	return available, known
}

func firstAvailableSpeedURL(order ...string) string {
	for _, candidate := range order {
		if available, _ := isSpeedURLAvailable(candidate); available {
			return candidate
		}
	}
	return ""
}

func speedURLStatusPayload() map[string]interface{} {
	speedURLAvailabilityState.RLock()
	defer speedURLAvailabilityState.RUnlock()
	entries := make([]map[string]interface{}, 0, 4)
	for _, candidate := range speedURLCandidates() {
		available, known := speedURLAvailabilityState.status[candidate]
		entries = append(entries, map[string]interface{}{
			"url":       candidate,
			"label":     speedURLLabel(candidate),
			"available": available,
			"tested":    known,
		})
	}
	return map[string]interface{}{"ready": speedURLAvailabilityState.ready, "urls": entries}
}

func logSpeedURLAvailabilitySummary() {
	speedURLAvailabilityState.RLock()
	if !speedURLAvailabilityState.ready {
		speedURLAvailabilityState.RUnlock()
		return
	}
	parts := make([]string, 0, 4)
	for _, candidate := range speedURLCandidates() {
		available, known := speedURLAvailabilityState.status[candidate]
		mark := "未测"
		if known {
			if available {
				mark = "可用"
			} else {
				mark = "不可用"
			}
		}
		parts = append(parts, fmt.Sprintf("%s(%s)=%s", speedURLLabel(candidate), candidate, mark))
	}
	speedURLAvailabilityState.RUnlock()
	fmt.Println("测速地址可用性: " + strings.Join(parts, " | "))
}
