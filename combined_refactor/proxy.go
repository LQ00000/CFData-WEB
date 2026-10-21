package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

var managementProxyRaw = ""
var managementProxyURL *url.URL

func parseManagementProxy(raw string) (*url.URL, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	if !strings.Contains(value, "://") {
		return nil, fmt.Errorf("代理地址缺少协议头，请使用 socks5://、socks5h://、http:// 或 https:// 开头")
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("代理地址解析失败: %v", err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if parsed.Hostname() == "" {
		return nil, fmt.Errorf("代理地址缺少主机")
	}
	if parsed.Port() == "" {
		return nil, fmt.Errorf("代理地址缺少端口")
	}
	switch scheme {
	case "http", "https":
	case "socks5", "socks5h":
		scheme = "socks5"
	default:
		return nil, fmt.Errorf("不支持的代理协议 %s://（支持 socks5://、socks5h://、http://、https://）", parsed.Scheme)
	}
	u := *parsed
	u.Scheme = scheme
	return &u, nil
}

func setManagementProxy(raw string) error {
	parsed, err := parseManagementProxy(raw)
	if err != nil {
		return err
	}
	managementProxyRaw = strings.TrimSpace(raw)
	managementProxyURL = parsed
	return nil
}

func managementProxyConfigured() bool {
	return managementProxyURL != nil
}

func managementProxyInUse() *url.URL {
	if managementProxyConfigured() {
		return managementProxyURL
	}
	for _, key := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		parsed, err := url.Parse(value)
		if err != nil {
			continue
		}
		switch strings.ToLower(parsed.Scheme) {
		case "socks5", "socks5h", "http", "https":
			return parsed
		}
	}
	return nil
}

func applyManagementProxy(transport *http.Transport) *http.Transport {
	if transport == nil || !managementProxyConfigured() {
		return transport
	}
	if managementProxyURL.Scheme == "socks5" {
		var auth *proxy.Auth
		if managementProxyURL.User != nil {
			pw, _ := managementProxyURL.User.Password()
			auth = &proxy.Auth{User: managementProxyURL.User.Username(), Password: pw}
		}
		dialer, err := proxy.SOCKS5("tcp", managementProxyURL.Host, auth, proxy.Direct)
		if err != nil {
			return transport
		}
		transport.Proxy = nil
		if cd, ok := dialer.(proxy.ContextDialer); ok {
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return cd.DialContext(ctx, network, addr)
			}
		} else {
			transport.DialContext = func(_ context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		}
		return transport
	}
	transport.Proxy = http.ProxyURL(managementProxyURL)
	return transport
}

func managementHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = dialContext
	transport.TLSClientConfig = tlsConfigWithRootCAs("")
	applyManagementProxy(transport)
	name := "upstream"
	if managementProxyConfigured() {
		name = "upstream-proxy"
	}
	client := &http.Client{Transport: wrapDebugTransport(name, transport), Timeout: upstreamHTTPClient.Timeout}
	if timeout > 0 {
		client.Timeout = timeout
	}
	return client
}

func maskProxyURL(raw *url.URL) string {
	if raw == nil {
		return ""
	}

	return raw.Scheme + "://" + raw.Host
}

func proxySourceLabel() string {
	if managementProxyConfigured() {
		return "程序配置"
	}
	for _, key := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return "系统环境变量"
		}
	}
	return ""
}

func proxyStatusLabel() string {
	if inUse := managementProxyInUse(); inUse != nil {
		return maskProxyURL(inUse)
	}
	return ""
}

func applyManagementProxyFromConfigFile(path string) {
	cfg, _, err := loadOrCreateCLIConfig(path)
	if err != nil {
		return
	}
	if u := strings.TrimSpace(cfg.URL); u != "" {
		speedURLConfigCandidates = append(speedURLConfigCandidates, u)
	}
	if u := strings.TrimSpace(cfg.NSBURL); u != "" {
		speedURLConfigCandidates = append(speedURLConfigCandidates, u)
	}
	if err := setManagementProxy(cfg.Proxy); err != nil {
		fmt.Printf("[proxy] 管理面代理配置无效，未启用: %v\n", err)
	}
}

var proxyConfigWriteMutex sync.Mutex

func updateProxyInConfigFile(path, raw string) error {
	proxyConfigWriteMutex.Lock()
	defer proxyConfigWriteMutex.Unlock()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {

			if err := writeCLIConfigTemplate(path, defaultCLIFileConfig()); err != nil {
				return err
			}
			data, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	var probe map[string]interface{}
	if err := json.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("配置文件不是合法 JSON: %w", err)
	}
	if configField, nested := probe["config"].(map[string]interface{}); nested {
		configField["proxy"] = raw
	} else {
		probe["proxy"] = raw
	}
	out, err := json.MarshalIndent(probe, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0600); err != nil {
		return err
	}
	return os.WriteFile(path+".bak", data, 0600)
}
