package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

type cliConfig struct {
	enabled         bool
	configResolved  bool
	mode            string
	scanMode        string
	ipType          int
	threads         int
	offThreads      int
	nsbThreads      int
	port            int
	delay           int
	offDelay        int
	nsbDelay        int
	resultLimit     int
	dc              string
	file            string
	sourceURL       string
	offURL          string
	nsbURL          string
	outFile         string
	nsbFallbackPort int
	speedTest       int
	speedLimit      int
	speedMin        float64
	enableTLS       bool
	compactNSB      bool
	nsbDC           string
	nsbSpeedMin     float64
	nsbSpeedLimit   int
	relink          bool
	relinkHost      string
	showProgress    bool
	noColor         bool
	compactIPv4     bool
	outIPType       string
	outQualified    string
	startRow        int
	endRow          int
	export          cliExportConfig
}

type cliExportConfig struct {
	ConfigFile    string `json:"-"`
	Format        string `json:"outformat"`
	Fields        string `json:"outfields"`
	Custom        string `json:"outcustom"`
	V6Bracket     bool   `json:"outv6bracket"`
	V6BracketSet  bool   `json:"-"`
	Separator     string `json:"outseparator"`
	GitHub        bool   `json:"github"`
	GitHubSet     bool   `json:"-"`
	GHRepo        string `json:"ghrepo"`
	GHBranch      string `json:"ghbranch"`
	GHPath        string `json:"ghpath"`
	GHMessage     string `json:"ghmessage"`
	GHToken       string `json:"ghtoken"`
	GHTokenFile   string `json:"ghtokenfile"`
	GHUpload      string `json:"ghupload"`
	EdgeTunnel    bool   `json:"edgetunnel"`
	EdgeTunnelSet bool   `json:"-"`
	ETHost        string `json:"ethost"`
	ETPassword    string `json:"etpassword"`
	ETMode        string `json:"etmode"`
}

type cliFileConfig struct {
	CLI             bool    `json:"cli"`
	Mode            string  `json:"mode"`
	ScanMode        string  `json:"scanmode"`
	IPType          int     `json:"offiptype"`
	Threads         int     `json:"offthreads"`
	NSBThreads      int     `json:"nsbthreads"`
	Out             string  `json:"out"`
	SpeedTest       int     `json:"nsbspeedtest"`
	Progress        bool    `json:"progress"`
	NoColor         bool    `json:"nocolor"`
	SkipGeo         bool    `json:"skipgeo"`
	URL             string  `json:"offurl"`
	NSBURL          string  `json:"nsburl"`
	DNS             string  `json:"dns"`
	Debug           any     `json:"debug"`
	CompactIPv4     bool    `json:"compactipv4"`
	TestPort        int     `json:"offport"`
	Delay           int     `json:"offdelay"`
	NSBDelay        int     `json:"nsbdelay"`
	DC              string  `json:"offdc"`
	SpeedLimit      int     `json:"offspeedlimit"`
	SpeedMin        float64 `json:"offspeedmin"`
	File            string  `json:"nsbfile"`
	SourceURL       string  `json:"nsbsourceurl"`
	NSBFallbackPort int     `json:"nsbfallbackport"`
	NSBDC           string  `json:"nsbdc"`
	Relink          bool    `json:"relink"`
	RelinkHost      string  `json:"relinkhost"`
	OutIPType       string  `json:"outiptype"`
	OutQualified    string  `json:"outqualified"`
	TLS             bool    `json:"nsbtls"`
	Compact         bool    `json:"nsbcompact"`
	ResultLimit     int     `json:"nsbresultlimit"`
	NSBSpeedMin     float64 `json:"nsbspeedmin"`
	NSBSpeedLimit   int     `json:"nsbspeedlimit"`
	Format          string  `json:"outformat"`
	Fields          string  `json:"outfields"`
	Custom          string  `json:"outcustom"`
	V6Bracket       bool    `json:"outv6bracket"`
	Separator       string  `json:"outseparator"`
	OutStartRow     int     `json:"outstartrow"`
	OutEndRow       int     `json:"outendrow"`
	GitHub          bool    `json:"github"`
	GHRepo          string  `json:"ghrepo"`
	GHBranch        string  `json:"ghbranch"`
	GHPath          string  `json:"ghpath"`
	GHMessage       string  `json:"ghmessage"`
	GHToken         string  `json:"ghtoken"`
	GHTokenFile     string  `json:"ghtokenfile"`
	GHUpload        string  `json:"ghupload"`
	EdgeTunnel      bool    `json:"edgetunnel"`
	ETHost          string  `json:"ethost"`
	ETPassword      string  `json:"etpassword"`
	ETMode          string  `json:"etmode"`
}

type cliResultRow map[string]string

type cliResultField struct {
	Key   string
	Label string
}

type cliCustomField struct {
	Key   string
	Label string
	Value string
}

var cliResultFields = []cliResultField{
	{Key: "ipport", Label: "ip:port"},
	{Key: "ip", Label: "IP地址"},
	{Key: "port", Label: "端口号"},
	{Key: "tls", Label: "TLS"},
	{Key: "lossRate", Label: "丢包率"},
	{Key: "latency", Label: "网络延迟"},
	{Key: "scanMode", Label: "扫描方式"},
	{Key: "speed", Label: "下载速度"},
	{Key: "outboundIP", Label: "出站IP"},
	{Key: "ipType", Label: "IP类型"},
	{Key: "originalInput", Label: "原始输入"},
	{Key: "dc", Label: "数据中心"},
	{Key: "dcCountry", Label: "落地区域"},
	{Key: "loc", Label: "源IP位置"},
	{Key: "region", Label: "地区"},
	{Key: "city", Label: "城市"},
	{Key: "asnNumber", Label: "ASN号码"},
	{Key: "asnOrg", Label: "ASN组织"},
	{Key: "visitScheme", Label: "访问协议"},
	{Key: "tlsVersion", Label: "TLS版本"},
	{Key: "sni", Label: "SNI"},
	{Key: "httpVersion", Label: "HTTP版本"},
	{Key: "warp", Label: "WARP"},
	{Key: "gateway", Label: "Gateway"},
	{Key: "rbi", Label: "RBI"},
	{Key: "kex", Label: "密钥交换"},
	{Key: "timestamp", Label: "时间戳"},
}

type cliFlagInfo struct {
	name         string
	description  string
	defaultValue string
}

var (
	ansiReset       = "\033[0m"
	ansiBold        = "\033[1m"
	ansiGreen       = "\033[32m"
	ansiBrightGreen = "\033[92m"
	ansiYellow      = "\033[33m"
	ansiRed         = "\033[31m"
	ansiCyan        = "\033[36m"
	ansiMagenta     = "\033[35m"

	cliCommonFlags = []cliFlagInfo{
		{name: "cli", description: "启用命令行模式：不带时启动 Web；-cli 进入交互菜单；-cli qs 按配置快速启动（请用 -cli 或 -cli=true）", defaultValue: "false"},
		{name: "port", description: "Web 服务监听端口", defaultValue: "13335"},
		{name: "user", description: "Web 认证用户名（不设置则不启用认证）", defaultValue: ""},
		{name: "password", description: "Web 认证密码（需同时设置 -user）", defaultValue: ""},
		{name: "session", description: "Web 登录会话有效期（分钟）", defaultValue: "720"},
		{name: "host", description: "服务监听地址；留空监听全部地址，Android APK 建议使用 127.0.0.1", defaultValue: ""},
		{name: "mode", description: "运行模式：official 或 nsb", defaultValue: "official"},
		{name: "scanmode", description: "扫描方式：tcping（默认，TCP 握手延迟）或 httping（HTTP TTFB，延迟比 tcping 高属正常，不同模式数据不可对比）", defaultValue: "tcping"},
		{name: "out", description: "输出文件名", defaultValue: "cfdata-results"},
		{name: "progress", description: "是否输出进度日志", defaultValue: "true"},
		{name: "nocolor", description: "禁用颜色输出（cmd 等不支持 ANSI 的终端可开启避免乱码）", defaultValue: "false"},
		{name: "skipgeo", description: "跳过地区/代理环境验证，CLI 启动测试前不再确认代理警告", defaultValue: "false"},
		{name: "url", description: "测速下载地址；auto 表示由后端自动选择内置测速源", defaultValue: autoSpeedURLValue},
		{name: "dns", description: "自定义 DNS 服务器，例如 1.1.1.1 或 223.5.5.5,8.8.8.8；默认系统 DNS 优先，失败回退内置 DNS；显式设置时强制使用指定 DNS", defaultValue: defaultDNSServers},
		{name: "debug", description: "调试输出等级：error、all；true 等同 error", defaultValue: "false"},
		{name: "compactipv4", description: "精简本地 IPv4 地址库：按 /24 子网测 TCP:80 连通性并覆盖 ips-v4.txt", defaultValue: "false"},
		{name: "config", description: "CLI 配置文件路径，不存在时在二进制目录自动生成模板", defaultValue: "二进制目录/cfdata-config.json"},
		{name: "outformat", description: "CLI 导出格式：csv 或 txt", defaultValue: "txt"},
		{name: "outfields", description: "CLI 导出字段：compact、all、ipport 或逗号分隔字段 key；可用 -outcustom 增加常量字段", defaultValue: "compact"},
		{name: "outcustom", description: "CLI 自定义导出字段，格式 标题:内容，多项用逗号分隔；在 -outfields 中可用标题排序，未写入则默认追加到最后", defaultValue: ""},
		{name: "outv6bracket", description: "TXT 导出时对 IPv6 地址加方括号（[IPv6]:端口）", defaultValue: "true"},
		{name: "outseparator", description: "TXT 导出字段分隔符；写 空格 表示空格", defaultValue: "-"},
		{name: "outqualified", description: "导出/上传合格结果筛选：all 全部 / qualified 仅合格 / unqualified 仅不合格", defaultValue: "all"},
		{name: "outiptype", description: "导出/上传 IP 类型筛选：all、ipv4 或 ipv6", defaultValue: "all"},
		{name: "outstartrow", description: "导出/上传开始行（筛选后第 1 行起计），只影响导出和上传内容", defaultValue: "1"},
		{name: "outendrow", description: "导出/上传结束行；0 表示至末尾，只影响导出和上传内容", defaultValue: "20"},
		{name: "github", description: "CLI 导出后上传到 GitHub", defaultValue: "false"},
		{name: "ghrepo", description: "GitHub 仓库，格式 owner/repo", defaultValue: ""},
		{name: "ghbranch", description: "GitHub 分支", defaultValue: "main"},
		{name: "ghpath", description: "GitHub 目标路径；留空时按 -outformat 自动使用 results/ip.csv 或 results/ip.txt", defaultValue: "<自动>"},
		{name: "ghmessage", description: "GitHub 提交信息", defaultValue: "update cfdata results"},
		{name: "ghtoken", description: "GitHub token（不推荐直接写入配置；强烈建议使用仅限制指定仓库读写权限的 token，并确保仓库内无重要数据）", defaultValue: ""},
		{name: "ghtokenfile", description: "GitHub token 文件路径（强烈建议文件内 token 仅限制指定仓库读写权限，并确保仓库内无重要数据）", defaultValue: ""},
		{name: "ghupload", description: "快速上传指定文件到 GitHub，不执行测试；需配合 -github", defaultValue: ""},
		{name: "edgetunnel", description: "CLI 导出后上传到 edgetunnel", defaultValue: "false"},
		{name: "ethost", description: "edgetunnel 主机地址，例如 https://example.com", defaultValue: ""},
		{name: "etpassword", description: "edgetunnel 密码", defaultValue: ""},
		{name: "etmode", description: "edgetunnel 上传模式：overwrite 覆盖 / append 追加", defaultValue: "overwrite"},
	}
	cliOfficialFlags = []cliFlagInfo{
		{name: "offiptype", description: "官方模式 IP 类型：4 或 6", defaultValue: "4"},
		{name: "offthreads", description: "官方模式扫描并发数", defaultValue: "100"},
		{name: "offport", description: "官方模式详细测试与测速端口", defaultValue: "443"},
		{name: "offdelay", description: "官方模式延迟阈值（毫秒）", defaultValue: "500"},
		{name: "offdc", description: "指定数据中心；不填时自动选择最低延迟数据中心", defaultValue: ""},
		{name: "offspeedlimit", description: "官方模式测速达标结果上限；0 表示关闭官方测速", defaultValue: "5"},
		{name: "offspeedmin", description: "官方模式测速达标下限，单位 MB/s", defaultValue: "0.1"},
		{name: "offurl", description: "官方模式测速下载地址；auto 表示由后端自动选择内置测速源", defaultValue: autoSpeedURLValue},
	}
	cliNSBFlags = []cliFlagInfo{
		{name: "nsbfile", description: "非标模式输入文件路径", defaultValue: ""},
		{name: "nsbsourceurl", description: "非标模式网络输入 URL", defaultValue: ""},
		{name: "nsbthreads", description: "非标模式扫描并发数", defaultValue: "100"},
		{name: "nsbdelay", description: "非标模式延迟阈值（毫秒）；0 表示不筛延迟", defaultValue: "0"},
		{name: "nsbfallbackport", description: "非标输入缺省端口；不填时随 TLS 自动使用 443/80", defaultValue: "0"},
		{name: "nsbdc", description: "非标模式指定结果数据中心；留空不限制", defaultValue: ""},
		{name: "nsbspeedtest", description: "非标测速线程数；0 表示不测速。多 IP 并发影响实际速度，需要准确应设为 1", defaultValue: "0"},
		{name: "nsburl", description: "非标模式测速下载地址；auto 表示由后端自动选择内置测速源", defaultValue: autoSpeedURLValue},
		{name: "nsbtls", description: "非标模式是否启用 TLS", defaultValue: "true"},
		{name: "nsbcompact", description: "非标模式导出精简表格列", defaultValue: "true"},
		{name: "nsbresultlimit", description: "非标模式延迟测试结果上限；必须为非 0 正整数", defaultValue: "1000"},
		{name: "nsbspeedmin", description: "非标模式测速结果阈值，单位 MB/s", defaultValue: "0.1"},
		{name: "nsbspeedlimit", description: "非标模式测速结果上限；0 表示关闭测速", defaultValue: "5"},
		{name: "relink", description: "链路复验开关；开启后扫描完成时对结果做真实链路验证，不可用的候选将被剔除", defaultValue: "true"},
		{name: "relinkhost", description: "链路复验探针域名；默认使用内置公共探针，配合 relink 开关使用", defaultValue: "probe.609909.xyz"},
	}
)

var errCLIConfigCreated = errors.New("CLI 配置文件已生成")

func registerCLIFlags() *cliConfig {
	cfg := &cliConfig{}
	flag.Usage = printCLIUsage
	flag.BoolVar(&cfg.enabled, "cli", false, "启用命令行模式（默认启动 Web）")
	flag.StringVar(&cfg.mode, "mode", "official", "CLI 模式：official 或 nsb")
	flag.StringVar(&cfg.scanMode, "scanmode", "tcping", "扫描方式：tcping（默认，TCP 握手延迟）或 httping（HTTP TTFB，延迟比 tcping 高属正常，不同模式数据不可对比）")
	flag.IntVar(&cfg.ipType, "offiptype", 4, "官方模式 IP 类型：4 或 6")
	flag.IntVar(&cfg.offThreads, "offthreads", 100, "官方扫描并发数")
	flag.IntVar(&cfg.nsbThreads, "nsbthreads", 100, "非标扫描并发数")
	flag.IntVar(&cfg.speedTest, "nsbspeedtest", 0, "非标测速线程数；表示同时测速的 IP 数量，0 表示不测速。多 IP 并发测速会影响实际下载速度，需要准确速度应设置为 1")
	flag.IntVar(&cfg.port, "offport", 443, "目标测试端口")
	flag.IntVar(&cfg.offDelay, "offdelay", 500, "官方延迟阈值（毫秒）")
	flag.IntVar(&cfg.nsbDelay, "nsbdelay", 0, "非标延迟阈值（毫秒）；0 表示不筛延迟")
	flag.StringVar(&cfg.dc, "offdc", "", "官方模式指定数据中心，不填则自动选择最低延迟数据中心")
	flag.StringVar(&cfg.file, "nsbfile", "", "非标模式输入文件路径")
	flag.StringVar(&cfg.sourceURL, "nsbsourceurl", "", "非标模式网络输入 URL")
	flag.IntVar(&cfg.nsbFallbackPort, "nsbfallbackport", 0, "非标输入缺省端口；不填时随 TLS 自动使用 443/80")
	flag.StringVar(&cfg.outIPType, "outiptype", "all", "导出/上传 IP 类型筛选：all、ipv4 或 ipv6")
	flag.StringVar(&cfg.outQualified, "outqualified", "all", "导出/上传合格结果筛选：all、qualified 或 unqualified")
	flag.StringVar(&cfg.nsbDC, "nsbdc", "", "非标模式指定结果数据中心")
	flag.StringVar(&cfg.outFile, "out", "cfdata-results", "输出文件名")
	flag.IntVar(&cfg.startRow, "outstartrow", 1, "导出/上传开始行（筛选后第 1 行起计）")
	flag.IntVar(&cfg.endRow, "outendrow", 20, "导出/上传结束行；0 表示至末尾")
	flag.IntVar(&cfg.speedLimit, "offspeedlimit", 5, "官方模式测速达标结果上限；0 表示关闭官方测速")
	flag.Float64Var(&cfg.speedMin, "offspeedmin", 0.1, "官方模式测速达标下限，单位 MB/s")
	flag.StringVar(&cfg.offURL, "offurl", autoSpeedURLValue, "官方测速下载地址")
	flag.StringVar(&cfg.nsbURL, "nsburl", autoSpeedURLValue, "非标测速下载地址")
	flag.BoolVar(&cfg.enableTLS, "nsbtls", true, "非标模式是否启用 TLS")
	flag.BoolVar(&cfg.compactNSB, "nsbcompact", true, "非标模式导出精简表格列")
	flag.IntVar(&cfg.resultLimit, "nsbresultlimit", 1000, "非标模式延迟测试结果上限；必须为非 0 正整数")
	flag.BoolVar(&cfg.relink, "relink", true, "链路复验开关；开启后非标扫描完成时对结果做真实链路验证（TCP+TLS+HTTP 访问探针），不可用的候选将被剔除")
	flag.StringVar(&cfg.relinkHost, "relinkhost", "probe.609909.xyz", "链路复验探针域名；默认使用内置公共探针，配合 -relink 开关使用")
	flag.Float64Var(&cfg.nsbSpeedMin, "nsbspeedmin", 0.1, "非标模式测速结果阈值，单位 MB/s")
	flag.IntVar(&cfg.nsbSpeedLimit, "nsbspeedlimit", 5, "非标模式测速结果上限；0 表示关闭测速")
	flag.BoolVar(&cfg.showProgress, "progress", true, "CLI 模式输出进度日志")
	flag.BoolVar(&cfg.noColor, "nocolor", false, "禁用 ANSI 颜色输出（cmd 等不支持的终端建议开启）")
	flag.BoolVar(&cfg.compactIPv4, "compactipv4", false, "精简本地 IPv4 地址库，按 /24 子网探测 TCP:80 连通性后覆盖 ips-v4.txt")
	flag.StringVar(&cfg.export.ConfigFile, "config", "", "CLI 导出/GitHub 配置文件路径")
	flag.StringVar(&cfg.export.Format, "outformat", "", "CLI 导出格式：csv 或 txt")
	flag.StringVar(&cfg.export.Fields, "outfields", "", "CLI 导出字段：compact、all、ipport 或逗号分隔字段 key")
	flag.StringVar(&cfg.export.Custom, "outcustom", "", "CLI 自定义导出字段，格式 标题:内容，多项用逗号分隔")
	flag.StringVar(&cfg.export.Separator, "outseparator", "", "TXT 导出字段分隔符；写 空格 表示空格")
	flag.BoolVar(&cfg.export.V6Bracket, "outv6bracket", true, "TXT 导出时对 IPv6 地址加方括号（[IPv6]:端口）")
	flag.BoolVar(&cfg.export.GitHub, "github", false, "CLI 导出后上传到 GitHub")
	flag.StringVar(&cfg.export.GHRepo, "ghrepo", "", "GitHub 仓库 owner/repo")
	flag.StringVar(&cfg.export.GHBranch, "ghbranch", "", "GitHub 分支")
	flag.StringVar(&cfg.export.GHPath, "ghpath", "", "GitHub 目标路径")
	flag.StringVar(&cfg.export.GHMessage, "ghmessage", "", "GitHub 提交信息")
	flag.StringVar(&cfg.export.GHToken, "ghtoken", "", "GitHub token")
	flag.StringVar(&cfg.export.GHTokenFile, "ghtokenfile", "", "GitHub token 文件")
	flag.StringVar(&cfg.export.GHUpload, "ghupload", "", "快速上传指定文件到 GitHub，不执行测试")
	flag.BoolVar(&cfg.export.EdgeTunnel, "edgetunnel", false, "CLI 导出后上传到 edgetunnel")
	flag.StringVar(&cfg.export.ETHost, "ethost", "", "edgetunnel 主机地址")
	flag.StringVar(&cfg.export.ETPassword, "etpassword", "", "edgetunnel 密码")
	flag.StringVar(&cfg.export.ETMode, "etmode", "overwrite", "edgetunnel 上传模式：overwrite 覆盖 / append 追加")
	return cfg
}

func runCLI(cfg *cliConfig) error {
	if !cfg.configResolved {
		if err := prepareCLIConfig(cfg); err != nil {
			return err
		}
	}
	applyCLISpeedDefault()
	printCLIConfig(cfg)

	if !skipGeoCheck {
		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		country, ok := detectCloudflareTraceCountry(ctx)
		cancel()
		if !confirmCLIProxyCountry(country, ok) {
			return fmt.Errorf("已取消：当前网络环境标签为 %s", firstNonEmpty(country, "未知"))
		}
	} else {
		fmt.Println("[proxy-check] 已跳过地区/代理环境验证")
	}

	if cfg.compactIPv4 {
		return runCompactIPv4CLI(cfg)
	}
	if strings.TrimSpace(cfg.export.GHUpload) != "" {
		return runCLIQuickGitHubUpload(cfg)
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.mode))
	switch mode {
	case "official":
		return runOfficialCLI(cfg)
	case "nsb":
		return runNSBCLI(cfg)
	default:
		return fmt.Errorf("不支持的 -mode: %s（仅支持 official 或 nsb）", cfg.mode)
	}
}

func applyCLISpeedDefault() {
	if !isAutoSpeedURL(speedTestURL) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, info, err := resolveStartupSpeedTestURL(ctx, speedTestURL)
	cancel()
	if err != nil {
		recordDebugError("speed_isp_check", err.Error())
		return
	}
	recordDebugByLevel("all", "speed_isp_check", fmt.Sprintf("cli asn=%d org=%s mobile=%v selected=%s", info.ASN, info.ASOrganization, isChinaMobileISP(info), currentAutoSpeedURLDefault()))
}

func prepareCLIConfig(cfg *cliConfig) error {
	if err := resolveCLIExportConfig(cfg); err != nil {
		return err
	}
	if err := validateCLIOutputConfig(cfg); err != nil {
		return err
	}
	if cfg.noColor {
		disableANSIColors()
	}
	cfg.configResolved = true
	return nil
}

func runCLIQuickGitHubUpload(cfg *cliConfig) error {
	if !cfg.export.GitHub {
		return fmt.Errorf("使用 -ghupload 快速上传时需要同时启用 -github")
	}
	path := expandHome(cfg.export.GHUpload)
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.export.GHPath) == "" || cfg.export.GHPath == "results/ip."+cfg.export.Format {
		cfg.export.GHPath = "results/" + filepath.Base(path)
	}
	return uploadCLIExportToGitHub(cfg, string(content))
}

func runCompactIPv4CLI(cfg *cliConfig) error {
	session := newCLISession(cfg)
	if err := session.runTaskSync(func(ctx context.Context, session *appSession) {
		runCompactIPv4Task(ctx, session)
	}); err != nil {
		return cliTaskError(err)
	}
	return nil
}

func resolveCLIExportConfig(cfg *cliConfig) error {
	provided := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { provided[f.Name] = true })
	configPath := cfg.export.ConfigFile
	if configPath == "" {
		configPath = defaultCLIConfigPath()
	}
	fileCfg, created, err := loadOrCreateCLIConfig(configPath)
	if err != nil {
		return err
	}
	if created {
		fmt.Printf("[config] 已生成配置文件: %s\n", configPath)
		fmt.Println("[config] 优先级: 命令行参数 > 配置文件 > 默认值")
		fmt.Println("[config] 请退出后按需编辑配置文件，再重新开始测试。")
		return errCLIConfigCreated
	}
	return applyCLIExportResolution(cfg, fileCfg, configPath, provided)
}

func applyCLIExportResolution(cfg *cliConfig, fileCfg cliFileConfig, configPath string, provided map[string]bool) error {
	merged := defaultCLIExportConfig()
	applyCLIFileConfig(cfg, fileCfg, provided)
	if !provided["nsbfallbackport"] && fileCfg.NSBFallbackPort <= 0 {
		cfg.nsbFallbackPort = defaultNSBPort(cfg.enableTLS)
	}
	mergeCLIExportConfig(&merged, fileCfg.Export(), false)
	mergeCLIExportConfig(&merged, cfg.export, true, provided)
	merged.ConfigFile = configPath
	merged.Format = strings.ToLower(strings.TrimSpace(merged.Format))
	if merged.Format == "" {
		merged.Format = "txt"
	}
	if merged.Format != "csv" && merged.Format != "txt" {
		return fmt.Errorf("不支持的 -outformat: %s", merged.Format)
	}
	if strings.TrimSpace(merged.Fields) == "" {
		merged.Fields = "compact"
	}
	if strings.TrimSpace(merged.Separator) == "" {
		merged.Separator = "-"
	}
	if strings.TrimSpace(merged.GHBranch) == "" {
		merged.GHBranch = "main"
	}
	if strings.TrimSpace(merged.GHMessage) == "" {
		merged.GHMessage = "update cfdata results"
	}
	if strings.TrimSpace(merged.GHPath) == "" || (!provided["ghpath"] && fileCfg.GHPath == "") {
		merged.GHPath = "results/ip." + merged.Format
	}
	if merged.GHToken == "" && strings.TrimSpace(merged.GHTokenFile) != "" {
		data, err := os.ReadFile(expandHome(merged.GHTokenFile))
		if err != nil {
			return fmt.Errorf("读取 token 文件失败: %w", err)
		}
		merged.GHToken = strings.TrimSpace(string(data))
	}
	cfg.export = merged
	return nil
}

func validateCLIOutputConfig(cfg *cliConfig) error {
	qualified, ok := normalizeCLIQualifiedFilter(cfg.outQualified)
	if !ok {
		return fmt.Errorf("-outqualified 仅支持 all、qualified 或 unqualified")
	}
	cfg.outQualified = qualified
	ipType := normalizeIPTypeFilter(cfg.outIPType)
	if ipType == "" {
		return fmt.Errorf("-outiptype 仅支持 all、ipv4 或 ipv6")
	}
	cfg.outIPType = ipType
	if cfg.startRow < 1 {
		return fmt.Errorf("-outstartrow 必须为正整数")
	}
	if cfg.endRow < 0 {
		return fmt.Errorf("-outendrow 不能为负数")
	}
	if cfg.endRow > 0 && cfg.endRow < cfg.startRow {
		return fmt.Errorf("-outendrow 不能小于 -outstartrow")
	}
	return nil
}

func defaultCLIExportConfig() cliExportConfig {
	return cliExportConfig{Format: "txt", Fields: "compact", Custom: "", V6Bracket: true, Separator: "-", GitHub: false, GHBranch: "main", GHPath: "", GHMessage: "update cfdata results", EdgeTunnel: false, ETMode: "overwrite"}
}

func defaultCLIFileConfig() cliFileConfig {
	return cliFileConfig{CLI: false, Mode: "official", ScanMode: "tcping", IPType: 4, Threads: 100, NSBThreads: 100, Out: "cfdata-results", SpeedTest: 0, Progress: true, NoColor: false, URL: autoSpeedURLValue, NSBURL: autoSpeedURLValue, DNS: defaultDNSServers, Debug: false, CompactIPv4: false, TestPort: 443, Delay: 500, NSBDelay: 0, DC: "", SpeedLimit: 5, SpeedMin: 0.1, File: "", SourceURL: "", NSBFallbackPort: 0, OutIPType: "all", OutQualified: "all", NSBDC: "", Relink: true, RelinkHost: "probe.609909.xyz", TLS: true, Compact: true, ResultLimit: 1000, NSBSpeedMin: 0.1, NSBSpeedLimit: 5, Format: "txt", Fields: "compact", Custom: "", V6Bracket: true, Separator: "-", OutStartRow: 1, OutEndRow: 20, GitHub: false, GHBranch: "main", GHPath: "", GHMessage: "update cfdata results", EdgeTunnel: false, ETMode: "overwrite"}
}

func (c cliFileConfig) Export() cliExportConfig {
	return cliExportConfig{Format: c.Format, Fields: c.Fields, Custom: c.Custom, V6Bracket: c.V6Bracket, V6BracketSet: true, Separator: c.Separator, GitHub: c.GitHub, GitHubSet: true, GHRepo: c.GHRepo, GHBranch: c.GHBranch, GHPath: c.GHPath, GHMessage: c.GHMessage, GHToken: c.GHToken, GHTokenFile: c.GHTokenFile, GHUpload: c.GHUpload, EdgeTunnel: c.EdgeTunnel, EdgeTunnelSet: true, ETHost: c.ETHost, ETPassword: c.ETPassword, ETMode: c.ETMode}
}

type cliExportConfigTemplate struct {
	ConfigVersion   any              `json:"_config_version"`
	Config          cliFileConfig    `json:"config"`
	Description     string           `json:"_description"`
	Priority        string           `json:"_priority"`
	Usage           string           `json:"_usage"`
	ConfigHelp      []cliConfigHelp  `json:"_config_help"`
	FormatValues    []string         `json:"_format_values"`
	FieldsValues    []string         `json:"_fields_values"`
	ModeValues      []string         `json:"_mode_values"`
	AvailableFields []cliResultField `json:"_available_fields"`
}

type cliConfigHelp struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Default     string   `json:"default"`
	Options     []string `json:"options,omitempty"`
}

func mergeCLIExportConfig(dst *cliExportConfig, src cliExportConfig, onlyProvided bool, provided ...map[string]bool) {
	isSet := func(flagName string, value string) bool {
		if onlyProvided {
			return len(provided) > 0 && provided[0][flagName]
		}
		return strings.TrimSpace(value) != ""
	}
	if isSet("config", src.ConfigFile) {
		dst.ConfigFile = src.ConfigFile
	}
	if isSet("outformat", src.Format) {
		dst.Format = src.Format
	}
	if isSet("outfields", src.Fields) {
		dst.Fields = src.Fields
	}
	if isSet("outcustom", src.Custom) {
		dst.Custom = src.Custom
	}
	if isSet("outseparator", src.Separator) {
		dst.Separator = src.Separator
	}
	if (!onlyProvided && src.V6BracketSet) || (onlyProvided && len(provided) > 0 && provided[0]["outv6bracket"]) {
		dst.V6Bracket = src.V6Bracket
		dst.V6BracketSet = true
	}
	if (!onlyProvided && src.GitHubSet) || (onlyProvided && len(provided) > 0 && provided[0]["github"]) {
		dst.GitHub = src.GitHub
		dst.GitHubSet = true
	}
	if isSet("ghrepo", src.GHRepo) {
		dst.GHRepo = src.GHRepo
	}
	if isSet("ghbranch", src.GHBranch) {
		dst.GHBranch = src.GHBranch
	}
	if isSet("ghpath", src.GHPath) {
		dst.GHPath = src.GHPath
	}
	if isSet("ghmessage", src.GHMessage) {
		dst.GHMessage = src.GHMessage
	}
	if isSet("ghtoken", src.GHToken) {
		dst.GHToken = src.GHToken
	}
	if isSet("ghtokenfile", src.GHTokenFile) {
		dst.GHTokenFile = src.GHTokenFile
	}
	if isSet("ghupload", src.GHUpload) {
		dst.GHUpload = src.GHUpload
	}
	if (!onlyProvided && src.EdgeTunnelSet) || (onlyProvided && len(provided) > 0 && provided[0]["edgetunnel"]) {
		dst.EdgeTunnel = src.EdgeTunnel
		dst.EdgeTunnelSet = true
	}
	if isSet("ethost", src.ETHost) {
		dst.ETHost = src.ETHost
	}
	if isSet("etpassword", src.ETPassword) {
		dst.ETPassword = src.ETPassword
	}
	if isSet("etmode", src.ETMode) {
		dst.ETMode = src.ETMode
	}
}

// cliFileConfigJSONKeys 返回配置结构体全部 json 字段名，作为新模板的 key 基准。
func cliFileConfigJSONKeys() map[string]bool {
	keys := make(map[string]bool)
	t := reflect.TypeOf(cliFileConfig{})
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name == "" || name == "-" {
			continue
		}
		keys[name] = true
	}
	return keys
}

// migrateConfigFileToLatest 新迁移机制：
//   - 文件 key 与新模板 key 完全一致且 _config_version 为当前版本 → 不做任何改动
//   - 无旧名键（仅版本旧或缺新键）→ 静默迁移：备份原文件 → 以新模板为基准把同名值迁入 → 写回新文件
//   - 文件存在新模板不认识的旧名键 → 交互提示「配置文件存在更新，是否迁移同变量名的配置？」
//     选 y：同上静默流程（只保留同名键，新增/改名项为默认值，提示用户自行设置）
//     选 n 或非交互环境（stdin=/dev/null）：不改文件继续运行，下次启动再提示
func migrateConfigFileToLatest(path string, data []byte) ([]byte, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return data, nil // 非 JSON：交由后续解析流程报错
	}
	templateKeys := cliFileConfigJSONKeys()
	cfgObj := raw
	nested := false
	if inner, ok := raw["config"].(map[string]interface{}); ok {
		cfgObj = inner
		nested = true
	}
	var unknown, missing []string
	for k := range cfgObj {
		if !templateKeys[k] {
			unknown = append(unknown, k)
		}
	}
	for k := range templateKeys {
		if _, ok := cfgObj[k]; !ok {
			missing = append(missing, k)
		}
	}
	version := ""
	if nested {
		version, _ = raw["_config_version"].(string)
		version = strings.TrimSpace(version)
	}
	if len(unknown) == 0 && len(missing) == 0 && version == appVersion {
		return data, nil // 与最新模板完全一致
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		if target, err := os.Readlink("/proc/self/fd/0"); err == nil && target == "/dev/null" {
			fmt.Printf("[migrate] 配置文件存在更新（旧变量名: %s），非交互环境跳过迁移，继续使用当前文件\n", strings.Join(unknown, ", "))
			return data, nil
		}
		fmt.Println("[migrate] 检测到配置文件存在更新（变量名有变化）")
		fmt.Printf("[migrate] 新模板中不存在的旧变量名: %s\n", strings.Join(unknown, ", "))
		fmt.Print("[migrate] 是否迁移同变量名的配置？(y=迁移并备份原文件，新增/改名项为默认值请自行设置 / n=跳过，下次启动再提示) > ")
		line, outcome := readMenuLine()
		if outcome != promptOK || !strings.EqualFold(strings.TrimSpace(line), "y") {
			fmt.Println("[migrate] 已跳过迁移，下次启动将再次提示")
			return data, nil
		}
	}
	backupPath := path + ".bak"
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		fmt.Printf("[migrate] 备份原配置失败，跳过本次迁移: %v\n", err)
		return data, nil
	}
	newCfg := defaultCLIFileConfig()
	if cfgBytes, err := json.Marshal(cfgObj); err == nil {
		_ = json.Unmarshal(cfgBytes, &newCfg) // 同名值迁入新模板基准；缺键保持默认；旧名键自然丢弃
	}
	if err := writeCLIConfigTemplate(path, newCfg); err != nil {
		return data, fmt.Errorf("写入迁移后的配置失败 %s: %w", path, err)
	}
	fmt.Printf("[migrate] 配置已按新模板迁移: %s（原文件备份: %s）\n", path, backupPath)
	if len(unknown) > 0 {
		fmt.Println("[migrate] 新增/改名的配置项已按默认值生成，请通过菜单 3「修改配置参数」自行设置")
	}
	migrated, err := os.ReadFile(path)
	if err != nil {
		return data, nil
	}
	return migrated, nil
}

func loadOrCreateCLIConfig(path string) (cliFileConfig, bool, error) {
	path = expandHome(path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return cliFileConfig{}, false, err
		}
		if err := writeCLIConfigTemplate(path, defaultCLIFileConfig()); err != nil {
			return cliFileConfig{}, false, err
		}
		return defaultCLIFileConfig(), true, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cliFileConfig{}, false, err
	}
	cfg := defaultCLIFileConfig()
	if len(strings.TrimSpace(string(data))) == 0 {
		return cfg, false, nil
	}

	data, err = migrateConfigFileToLatest(path, data)
	if err != nil {
		return cliFileConfig{}, false, err
	}

	var probe map[string]interface{}
	nested := false
	if err := json.Unmarshal(data, &probe); err == nil {
		_, nested = probe["config"].(map[string]interface{})
	}
	if nested {
		template := cliExportConfigTemplate{Config: defaultCLIFileConfig()}
		if err := json.Unmarshal(data, &template); err != nil {
			return cliFileConfig{}, false, fmt.Errorf("解析配置文件失败 %s: %w", path, err)
		}
		cfg = template.Config
	} else if err := json.Unmarshal(data, &cfg); err != nil {
		return cliFileConfig{}, false, fmt.Errorf("解析配置文件失败 %s: %w", path, err)
	}
	return cfg, false, nil
}

func newCLIConfigTemplate(cfg cliFileConfig) cliExportConfigTemplate {
	return cliExportConfigTemplate{
		ConfigVersion:   appVersion,
		Config:          cfg,
		Description:     "CFData CLI 全量配置；真正配置项在 config 内。",
		Priority:        "命令行参数 > 配置文件 > 默认值",
		Usage:           "首次生成后建议退出并编辑本文件，再重新运行测试。debug 支持 false、error、all、true。",
		ConfigHelp:      buildCLIConfigHelp(),
		FormatValues:    []string{"csv", "txt"},
		FieldsValues:    []string{"compact", "all", "ipport", "ipport,dc,loc", "ipport,latency,dc,loc"},
		ModeValues:      []string{"official", "nsb"},
		AvailableFields: cliResultFields,
	}
}

func writeCLIConfigTemplate(path string, cfg cliFileConfig) error {
	template := newCLIConfigTemplate(cfg)
	var buf strings.Builder
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(template); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(buf.String()), 0600)
}

func buildCLIConfigHelp() []cliConfigHelp {
	return []cliConfigHelp{
		{Name: "cli", Description: "CLI 模式开关；已废弃本字段，CLI 仅由命令行 -cli 控制（保留仅为兼容旧配置）", Default: "false", Options: []string{"true", "false"}},
		{Name: "mode", Description: "运行模式", Default: "official", Options: []string{"official", "nsb"}},
		{Name: "scanmode", Description: "扫描方式；tcping：仅测量 TCP 握手延迟（默认），httping：测量 HTTP TTFB 全链路延迟，延迟比 tcping 高属正常，不同模式数据不可互相比较", Default: "tcping", Options: []string{"tcping", "httping"}},
		{Name: "offiptype", Description: "官方模式 IP 类型", Default: "4", Options: []string{"4", "6"}},
		{Name: "offthreads", Description: "官方扫描并发数", Default: "100"},
		{Name: "nsbthreads", Description: "非标扫描并发数", Default: "100"},
		{Name: "out", Description: "输出文件名", Default: "cfdata-results"},
		{Name: "nsbspeedtest", Description: "非标测速线程数；0 表示不测速。多 IP 并发影响实际速度，需要准确应设为 1", Default: "0"},
		{Name: "progress", Description: "输出进度日志", Default: "true", Options: []string{"true", "false"}},
		{Name: "nocolor", Description: "禁用 ANSI 颜色输出", Default: "false", Options: []string{"true", "false"}},
		{Name: "skipgeo", Description: "跳过地区/代理环境验证（等同命令行 -skipgeo）", Default: "false", Options: []string{"true", "false"}},
		{Name: "offurl", Description: "官方测速下载地址；auto 表示由后端自动选择内置测速源", Default: autoSpeedURLValue},
		{Name: "nsburl", Description: "非标测速下载地址；auto 表示由后端自动选择内置测速源", Default: autoSpeedURLValue},
		{Name: "dns", Description: "自定义 DNS 服务器；默认系统 DNS 优先，失败回退内置 DNS；显式设置时强制使用指定 DNS。用于 IP 库、locations、ASN、GitHub、网络 URL 输入等需要 DNS 的外部请求", Default: defaultDNSServers},
		{Name: "debug", Description: "调试输出等级；error 记录程序错误和下载/更新/API 异常，all 额外包含测速失败等全部明细", Default: "false", Options: []string{"false", "error", "all", "true"}},
		{Name: "compactipv4", Description: "精简本地 IPv4 地址库并覆盖 ips-v4.txt", Default: "false", Options: []string{"true", "false"}},
		{Name: "offport", Description: "官方模式测试端口", Default: "443"},
		{Name: "offdelay", Description: "官方延迟阈值，单位毫秒", Default: "500"},
		{Name: "nsbdelay", Description: "非标延迟阈值，单位毫秒；0=不筛延迟", Default: "0"},
		{Name: "offdc", Description: "官方模式指定数据中心；留空自动选择最低延迟数据中心", Default: ""},
		{Name: "offspeedlimit", Description: "官方模式测速达标结果上限；0 表示关闭官方测速", Default: "5"},
		{Name: "offspeedmin", Description: "官方模式测速达标下限，单位 MB/s", Default: "0.1"},
		{Name: "nsbfile", Description: "非标模式输入文件路径", Default: ""},
		{Name: "nsbsourceurl", Description: "非标模式网络输入 URL", Default: ""},
		{Name: "nsbfallbackport", Description: "非标输入缺省端口；不填时随 TLS 自动使用 443/80；显式设置时必须为 1-65535", Default: "自动"},
		{Name: "nsbdc", Description: "非标模式指定结果数据中心；留空不限制", Default: ""},
		{Name: "nsbtls", Description: "非标模式启用 TLS；缺省端口随 TLS 为 443/80", Default: "true", Options: []string{"true", "false"}},
		{Name: "nsbcompact", Description: "非标模式本地 CSV 是否默认精简字段", Default: "true", Options: []string{"true", "false"}},
		{Name: "nsbresultlimit", Description: "非标模式延迟测试结果上限；必须为非 0 正整数", Default: "1000"},
		{Name: "nsbspeedmin", Description: "非标模式测速结果阈值，单位 MB/s", Default: "0.1"},
		{Name: "nsbspeedlimit", Description: "非标模式测速结果上限；0 表示关闭测速", Default: "5"},
		{Name: "relink", Description: "链路复验开关；开启后非标扫描完成时对结果做真实链路验证（TCP+TLS+HTTP 访问探针域），不可用的候选将被剔除", Default: "true", Options: []string{"true", "false"}},
		{Name: "relinkhost", Description: "链路复验探针域名；默认使用内置公共探针，配合 relink 开关使用", Default: "probe.609909.xyz"},
		{Name: "outformat", Description: "导出/上传内容格式", Default: "txt", Options: []string{"csv", "txt"}},
		{Name: "outfields", Description: "导出字段；支持 compact、all、ipport 或逗号分隔字段 key；自定义字段可写在这里排序", Default: "compact", Options: []string{"compact", "all", "ipport", "ipport,dc,loc", "ipport,latency,dc,loc"}},
		{Name: "outcustom", Description: "自定义导出字段，格式 标题:内容，多项用逗号分隔；未在 outfields 中排序时默认追加到最后。兼容 key=标题:内容", Default: ""},
		{Name: "outv6bracket", Description: "TXT 导出时对 IPv6 地址加方括号（[IPv6]:端口），仅对 IPv6 行生效", Default: "true", Options: []string{"true", "false"}},
		{Name: "outseparator", Description: "TXT 导出字段分隔符；写 空格 表示空格", Default: "-"},
		{Name: "outqualified", Description: "导出/上传合格结果筛选；只影响导出和上传内容", Default: "all", Options: []string{"all", "qualified", "unqualified"}},
		{Name: "outiptype", Description: "导出/上传 IP 类型筛选；只影响导出和上传内容", Default: "all", Options: []string{"all", "ipv4", "ipv6"}},
		{Name: "outstartrow", Description: "导出/上传开始行（筛选后第 1 行起计）；只影响导出和上传内容", Default: "1"},
		{Name: "outendrow", Description: "导出/上传结束行；0 表示至末尾；只影响导出和上传内容", Default: "20"},
		{Name: "github", Description: "导出后上传到 GitHub", Default: "false", Options: []string{"true", "false"}},
		{Name: "ghrepo", Description: "GitHub 仓库，格式 owner/repo", Default: ""},
		{Name: "ghbranch", Description: "GitHub 分支", Default: "main"},
		{Name: "ghpath", Description: "GitHub 目标路径；留空时按 outformat 自动使用 results/ip.csv 或 results/ip.txt；文件不存在会新建，存在会覆盖", Default: "自动按 outformat 生成"},
		{Name: "ghmessage", Description: "GitHub 提交信息", Default: "update cfdata results"},
		{Name: "ghtoken", Description: "GitHub token；不推荐直接写入配置。强烈建议使用仅限制指定仓库读写权限的 token，并确保仓库内无重要数据，避免 token 泄露造成不必要的意外", Default: ""},
		{Name: "ghtokenfile", Description: "GitHub token 文件路径。强烈建议文件内 token 仅限制指定仓库读写权限，并确保仓库内无重要数据", Default: ""},
		{Name: "ghupload", Description: "快速上传指定文件到 GitHub，不执行测试；需 github=true", Default: ""},
		{Name: "edgetunnel", Description: "导出后上传到 edgetunnel", Default: "false", Options: []string{"true", "false"}},
		{Name: "ethost", Description: "edgetunnel 主机地址，例如 https://example.com", Default: ""},
		{Name: "etpassword", Description: "edgetunnel 密码", Default: ""},
		{Name: "etmode", Description: "edgetunnel 上传模式", Default: "overwrite", Options: []string{"overwrite", "append"}},
	}
}

func applyCLIFileConfig(cfg *cliConfig, fileCfg cliFileConfig, provided map[string]bool) {
	setString := func(name string, target *string, value string) {
		if !provided[name] && strings.TrimSpace(value) != "" {
			*target = value
		}
	}
	setInt := func(name string, target *int, value int) {
		if !provided[name] {
			*target = value
		}
	}
	setFloat := func(name string, target *float64, value float64) {
		if !provided[name] {
			*target = value
		}
	}
	if !provided["debug"] {
		applyConfigDebug(fileCfg.Debug)
	}
	setString("mode", &cfg.mode, fileCfg.Mode)
	setString("scanmode", &cfg.scanMode, fileCfg.ScanMode)
	isNSBRun := strings.EqualFold(strings.TrimSpace(cfg.mode), "nsb")
	if !provided["offiptype"] {
		cfg.ipType = fileCfg.IPType
	}
	if !provided["offthreads"] {
		cfg.offThreads = fileCfg.Threads
	}
	if !provided["nsbthreads"] {
		cfg.nsbThreads = fileCfg.NSBThreads
	}
	if !provided["out"] {
		cfg.outFile = fileCfg.Out
	}
	setInt("nsbspeedtest", &cfg.speedTest, fileCfg.SpeedTest)
	if !provided["progress"] {
		cfg.showProgress = fileCfg.Progress
	}
	if !provided["nocolor"] {
		cfg.noColor = fileCfg.NoColor
	}
	if !provided["skipgeo"] {
		skipGeoCheck = fileCfg.SkipGeo
	}
	if !provided["offurl"] && strings.TrimSpace(fileCfg.URL) != "" {
		cfg.offURL = fileCfg.URL
	}
	if !provided["nsburl"] && strings.TrimSpace(fileCfg.NSBURL) != "" {
		cfg.nsbURL = fileCfg.NSBURL
	}
	if !provided["dns"] && strings.TrimSpace(fileCfg.DNS) != "" {
		customDNSServer = fileCfg.DNS
	}
	if provided["dns"] {
		customDNSForced = true
	}
	if !provided["compactipv4"] {
		cfg.compactIPv4 = fileCfg.CompactIPv4
	}
	if !provided["offport"] {
		cfg.port = fileCfg.TestPort
	}
	if !provided["offdelay"] {
		cfg.offDelay = fileCfg.Delay
	}
	if !provided["nsbdelay"] {
		cfg.nsbDelay = fileCfg.NSBDelay
	}
	if !provided["offdc"] {
		cfg.dc = fileCfg.DC
	}
	setInt("offspeedlimit", &cfg.speedLimit, fileCfg.SpeedLimit)
	setFloat("offspeedmin", &cfg.speedMin, fileCfg.SpeedMin)
	setString("nsbfile", &cfg.file, fileCfg.File)
	setString("nsbsourceurl", &cfg.sourceURL, fileCfg.SourceURL)
	if !provided["nsbfallbackport"] && fileCfg.NSBFallbackPort > 0 {
		cfg.nsbFallbackPort = fileCfg.NSBFallbackPort
	}
	setString("outiptype", &cfg.outIPType, fileCfg.OutIPType)
	setString("outqualified", &cfg.outQualified, fileCfg.OutQualified)
	setInt("outstartrow", &cfg.startRow, fileCfg.OutStartRow)
	setInt("outendrow", &cfg.endRow, fileCfg.OutEndRow)
	setString("nsbdc", &cfg.nsbDC, fileCfg.NSBDC)
	if !provided["nsbtls"] {
		cfg.enableTLS = fileCfg.TLS
	}
	if !provided["nsbcompact"] {
		cfg.compactNSB = fileCfg.Compact
	}
	setInt("nsbresultlimit", &cfg.resultLimit, fileCfg.ResultLimit)
	setFloat("nsbspeedmin", &cfg.nsbSpeedMin, fileCfg.NSBSpeedMin)
	setInt("nsbspeedlimit", &cfg.nsbSpeedLimit, fileCfg.NSBSpeedLimit)
	if !provided["relink"] {
		cfg.relink = fileCfg.Relink
	}
	setString("relinkhost", &cfg.relinkHost, fileCfg.RelinkHost)

	genericURL := speedTestURL
	if isNSBRun {
		cfg.threads = cfg.nsbThreads
		cfg.delay = cfg.nsbDelay
		if isAutoSpeedURL(cfg.nsbURL) && provided["url"] && strings.TrimSpace(genericURL) != "" {
			speedTestURL = genericURL
		} else {
			speedTestURL = cfg.nsbURL
		}
	} else {
		cfg.threads = cfg.offThreads
		cfg.delay = cfg.offDelay
		if isAutoSpeedURL(cfg.offURL) && provided["url"] && strings.TrimSpace(genericURL) != "" {
			speedTestURL = genericURL
		} else {
			speedTestURL = cfg.offURL
		}
	}
}

func defaultCLIConfigPath() string {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return "cfdata-config.json"
	}
	return filepath.Join(filepath.Dir(exe), "cfdata-config.json")
}

func expandHome(path string) string {
	path = strings.TrimSpace(path)
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func applyConfigDebug(value any) {
	switch v := value.(type) {
	case bool:
		if v {
			_ = setDebugFlag("error")
		} else {
			_ = setDebugFlag("false")
		}
	case string:
		_ = setDebugFlag(v)
	default:
		_ = setDebugFlag("false")
	}
}

func newCLISession(cfg *cliConfig) *appSession {
	session := &appSession{progressState: map[string][2]int{}}
	session.emit = func(msgType string, data interface{}) {
		handleCLIMessage(cfg, session, msgType, data)
	}
	return session
}

func handleCLIMessage(cfg *cliConfig, session *appSession, msgType string, data interface{}) {
	debugWrongType := func(expected string) {
		if debugMode {
			fmt.Fprintf(os.Stderr, "%s[cli-debug]%s 消息 %s 类型断言失败，期望 %s，实际 %T\n", ansiYellow, ansiReset, msgType, expected, data)
		}
	}
	switch msgType {
	case "log":
		fmt.Println(data)
	case "error":
		recordProgramDebugError("cli_message_error", data)
		fmt.Fprintln(os.Stderr, colorize(fmt.Sprint(data), ansiRed))
	case "scan_progress":
		if cfg.showProgress {
			m, ok := data.(map[string]interface{})
			if !ok {
				debugWrongType("map[string]interface{}")
				break
			}
			current := asInt(m["current"])
			total := asInt(m["total"])
			setCLIProgress(session, "scan", current, total)
		}
	case "test_progress":
		if cfg.showProgress {
			m, ok := data.(map[string]interface{})
			if !ok {
				debugWrongType("map[string]interface{}")
				break
			}
			current := asInt(m["current"])
			total := asInt(m["total"])
			setCLIProgress(session, "test", current, total)
		}
	case "nsb_progress":
		if cfg.showProgress {
			m, ok := data.(map[string]interface{})
			if !ok {
				debugWrongType("map[string]interface{}")
				break
			}
			phase := fmt.Sprint(m["phase"])
			label := "scan"
			if phase == "speed" {
				label = "speed"
			}
			current := asInt(m["current"])
			total := asInt(m["total"])
			setCLIProgress(session, label, current, total)
		}
	case "scan_result":
		res, ok := data.(ScanResult)
		if !ok {
			debugWrongType("ScanResult")
			break
		}
		fmt.Printf("%s[scan-result]%s %s %s:%d %s %s %s\n", ansiMagenta, ansiReset, advanceCLIProgress(session, "scan"), res.IP, res.Port, res.DataCenter, res.City, colorizeLatencyString(res.LatencyStr))
	case "nsb_scan_result":
		m, ok := data.(nsbScanMessage)
		if !ok {
			debugWrongType("nsbScanMessage")
			break
		}
		if m.Speed != "" && m.Speed != "-" {
			progress := advanceCLIProgress(session, "speed")
			fmt.Printf("%s[speed]%s %s %s:%s %s\n", ansiMagenta, ansiReset, progress, m.IP, m.Port, colorizeSpeedString(m.Speed))
		} else {
			progress := advanceCLIProgress(session, "scan")
			fmt.Printf("%s[scan-result]%s %s %s:%s %s\n", ansiMagenta, ansiReset, progress, m.IP, m.Port, colorizeLatencyString(m.Latency))
		}
	case "test_result":
		res, ok := data.(TestResult)
		if !ok {
			debugWrongType("TestResult")
			break
		}
		fmt.Printf("%s[test-result]%s %s %s loss=%s avg=%s\n", ansiMagenta, ansiReset, advanceCLIProgress(session, "test"), res.IP, colorizeLossRate(res.LossRate), colorizeLatencyMS(int(res.AvgLatency/time.Millisecond)))
	case "test_complete":
		results, ok := data.([]TestResult)
		if !ok {
			debugWrongType("[]TestResult")
			break
		}
		session.testMutex.Lock()
		session.testResults = append([]TestResult(nil), results...)
		session.testMutex.Unlock()
		fmt.Printf("%s[test-complete]%s %d results\n", ansiCyan, ansiReset, len(results))
	case "nsb_csv_complete":
		payload, ok := data.(nsbCSVCompletePayload)
		if !ok {
			debugWrongType("nsbCSVCompletePayload")
			break
		}
		session.nsbMutex.Lock()
		session.nsbHeaders = append([]string(nil), payload.Headers...)
		session.nsbRows = append([][]string(nil), payload.Rows...)
		session.nsbMutex.Unlock()
		fmt.Printf("%s[nsb-output]%s %s (%d rows)\n", ansiGreen, ansiReset, payload.File, len(payload.Rows))
		if payload.Status == "failed" {
			fmt.Printf("%s[nsb]%s 任务失败: 测试%s\n", ansiRed, ansiReset, payload.Message)
		} else if payload.Status == "partial" {
			fmt.Printf("%s[nsb]%s 任务结束: %s\n", ansiYellow, ansiReset, payload.Message)
		}
	case "speed_test_result":
		m, ok := data.(map[string]string)
		if !ok {
			debugWrongType("map[string]string")
			break
		}
		endpoint := m["endpoint"]
		if endpoint == "" {
			endpoint = m["ip"]
		}
		fmt.Printf("%s[speed]%s %s %s\n", ansiMagenta, ansiReset, endpoint, colorizeSpeedString(m["speed"]))
	case "compact_ipv4_progress":
		if cfg.showProgress {
			m, ok := data.(map[string]interface{})
			if !ok {
				debugWrongType("map[string]interface{}")
				break
			}
			current := asInt(m["current"])
			total := asInt(m["total"])
			setCLIProgress(session, "compact", current, total)
			maybePrintCLIProgress(session, "compact", current, total)
		}
	case "compact_ipv4_hit":
		if !debugMode {
			break
		}
		m, ok := data.(map[string]interface{})
		if !ok {
			debugWrongType("map[string]interface{}")
			break
		}
		fmt.Printf("%s[compact-hit]%s pass=%v %v\n", ansiMagenta, ansiReset, m["pass"], m["ip"])
	case "compact_ipv4_done":
		m, ok := data.(map[string]interface{})
		if !ok {
			debugWrongType("map[string]interface{}")
			break
		}
		fmt.Printf("%s[compact-done]%s 保留 %v 个子网 → %v\n", ansiGreen, ansiReset, m["count"], m["file"])
	}
}

func runOfficialCLI(cfg *cliConfig) error {
	if cfg.ipType != 4 && cfg.ipType != 6 {
		return errors.New("官方模式 -offiptype 仅支持 4 或 6")
	}
	if cfg.threads <= 0 {
		cfg.threads = 100
	}
	if cfg.port <= 0 {
		cfg.port = 443
	}
	if cfg.delay < 0 {
		cfg.delay = 0
	}
	if cfg.speedLimit < 0 {
		cfg.speedLimit = 0
	}
	if cfg.speedMin <= 0 {
		cfg.speedMin = 0.1
	}

	scanMode := cfg.scanMode
	if scanMode == "" {
		scanMode = scanModeTCPing
	}

	session := newCLISession(cfg)
	if err := session.runTaskSync(func(ctx context.Context, session *appSession) {
		runOfficialTask(ctx, session, cfg.ipType, cfg.threads, cfg.port, cfg.delay, scanMode)
	}); err != nil {
		return cliTaskError(err)
	}

	session.scanMutex.Lock()
	scanResults := append([]ScanResult(nil), session.scanResults...)
	session.scanMutex.Unlock()
	if len(scanResults) == 0 {
		return errors.New("官方模式未发现有效 IP")
	}

	dc := strings.TrimSpace(cfg.dc)
	if dc == "" {
		dc = pickBestDataCenter(scanResults)
		if dc == "" {
			fmt.Printf("%s[official]%s 无法确定数据中心，仅输出扫描结果\n", ansiYellow, ansiReset)
			return writeCLIExportAndMaybeUpload(cfg, applyCLIExportFilters(cfg, officialScanRows(scanResults, scanMode), cfg.speedMin), "official-scan")
		}
		fmt.Printf("%s[official]%s 自动选择数据中心: %s\n", ansiGreen, ansiReset, colorize(dc, ansiBold+ansiGreen))
	}

	session.testMutex.Lock()
	session.testResults = nil
	session.testMutex.Unlock()
	if err := session.runTaskSync(func(ctx context.Context, session *appSession) {
		runDetailedTest(ctx, session, dc, cfg.port, cfg.delay, scanMode)
	}); err != nil {
		return cliTaskError(err)
	}

	session.testMutex.Lock()
	results := append([]TestResult(nil), session.testResults...)
	session.testMutex.Unlock()
	if cfg.speedLimit <= 0 {
		fmt.Printf("%s[official]%s 官方测速已关闭（-offspeedlimit 0）\n", ansiYellow, ansiReset)
	} else if len(results) == 0 {
		fmt.Printf("%s[official]%s 没有可用的详细测试结果，跳过测速\n", ansiYellow, ansiReset)
	} else {
		sortOfficialTestResults(results)
		setCLIProgress(session, "speed", 0, cfg.speedLimit)
		fmt.Printf("%s[official]%s 开始测速：目标上限=%d，测速阈值=%.2fMB/s\n", ansiGreen, ansiReset, cfg.speedLimit, cfg.speedMin)
		results = runOfficialSpeedTests(context.Background(), session, results, cfg.port, cfg.speedLimit, cfg.speedMin)
	}
	return writeCLIExportAndMaybeUpload(cfg, applyCLIExportFilters(cfg, officialResultRows(scanResults, results, scanMode), cfg.speedMin), "official")
}

func runNSBCLI(cfg *cliConfig) error {
	if strings.TrimSpace(cfg.file) == "" && strings.TrimSpace(cfg.sourceURL) == "" {
		return errors.New("非标模式需要通过 -nsbfile 或 -nsbsourceurl 指定输入来源")
	}
	if cfg.threads <= 0 {
		cfg.threads = 100
	}
	if cfg.speedTest < 0 {
		cfg.speedTest = 0
	}
	if cfg.resultLimit <= 0 {
		return fmt.Errorf("-nsbresultlimit 必须是非 0 正整数")
	}
	if cfg.nsbSpeedLimit < 0 {
		cfg.nsbSpeedLimit = 0
	}
	if cfg.nsbSpeedMin < 0 {
		cfg.nsbSpeedMin = 0
	}
	if cfg.nsbFallbackPort <= 0 || cfg.nsbFallbackPort > 65535 {
		return errors.New("-nsbfallbackport 必须为 1-65535")
	}
	if cfg.delay < 0 {
		cfg.delay = 0
	}
	if strings.TrimSpace(cfg.outFile) == "" {
		cfg.outFile = "cfdata-results"
	}
	inputName := cfg.file
	content := ""
	var err error
	if strings.TrimSpace(cfg.file) != "" {
		content, err = getFileContent(cfg.file)
		if err != nil {
			return err
		}
	} else {
		parsedURL, parseErr := url.Parse(cfg.sourceURL)
		if parseErr != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
			return errors.New("-nsbsourceurl 必须是有效的 http/https 地址")
		}
		inputName = cfg.sourceURL
		content, err = getURLContent(cfg.sourceURL)
		if err != nil {
			return fmt.Errorf("获取非标网络输入失败: %w", err)
		}
	}

	session := newCLISession(cfg)
	if err := session.runTaskSync(func(ctx context.Context, session *appSession) {
		scanMode := cfg.scanMode
		if scanMode == "" {
			scanMode = scanModeTCPing
		}
		runNSBTask(ctx, session, inputName, content, cfg.outFile, cfg.threads, cfg.nsbFallbackPort, cfg.speedTest, speedTestURL, cfg.enableTLS, cfg.delay, cfg.resultLimit, cfg.nsbDC, cfg.nsbSpeedMin, cfg.nsbSpeedLimit, cfg.compactNSB, scanMode, func() string {
			if !cfg.relink {
				return ""
			}
			return cfg.relinkHost
		}())
	}); err != nil {
		return cliTaskError(err)
	}
	session.nsbMutex.Lock()
	rows := nsbPayloadRows(session.nsbHeaders, session.nsbRows)
	session.nsbMutex.Unlock()
	rows = applyCLIExportFilters(cfg, rows, cfg.nsbSpeedMin)
	if len(rows) == 0 {
		fmt.Printf("%s[nsb]%s 没有符合导出条件的结果\n", ansiYellow, ansiReset)
		return nil
	}
	return writeCLIExportAndMaybeUpload(cfg, rows, "nsb")
}

func cliTaskError(err error) error {
	if errors.Is(err, context.Canceled) {
		return errors.New("已有任务正在运行，请等待完成后再试")
	}
	return err
}

func pickBestDataCenter(scanResults []ScanResult) string {
	dcLatency := map[string]time.Duration{}
	for _, res := range scanResults {
		current, ok := dcLatency[res.DataCenter]
		if !ok || res.TCPDuration < current {
			dcLatency[res.DataCenter] = res.TCPDuration
		}
	}
	type item struct {
		dc      string
		latency time.Duration
	}
	items := make([]item, 0, len(dcLatency))
	for dc, latency := range dcLatency {
		items = append(items, item{dc: dc, latency: latency})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].latency < items[j].latency })
	if len(items) == 0 {
		return ""
	}
	return items[0].dc
}

func runOfficialSpeedTests(ctx context.Context, session *appSession, results []TestResult, port int, limit int, speedMinMB float64) []TestResult {
	updated, _ := runOfficialSpeedTestsCore(ctx, results, port, limit, speedMinMB, speedTestURL, func(current, total, qualifiedCount int, result TestResult) {
		setCLIProgress(session, "speed", min(qualifiedCount, limit), limit)
		fmt.Printf("%s[speed]%s %s %s:%d %s\n", ansiMagenta, ansiReset, renderCLIProgress(session, "speed"), result.IP, port, colorizeSpeedString(result.Speed))
		if speedMB, ok := parseSpeedMBForSort(result.Speed); ok && speedMB >= speedMinMB {
			fmt.Printf("%s[official]%s 达标 %d/%d\n", ansiGreen, ansiReset, qualifiedCount, limit)
		}
	}, func() {
		fmt.Printf("%s[official]%s %s\n", ansiYellow, ansiReset, speedRateLimitMessage)
	})
	return updated
}

func printCLIConfig(cfg *cliConfig) {
	fmt.Printf("%s %s\n", colorize("CFData-WEB 版本:", ansiBold+ansiGreen), appVersion)
	fmt.Printf("%s %s\n", colorize("调试等级:", ansiBold+ansiGreen), debugFlagValue{}.String())
	checkAndPrintUpdate("")

	line := func(label, value string) {
		fmt.Printf("  %s：%s\n", label, value)
	}
	boolLabel := func(v bool) string {
		if v {
			return "开启"
		}
		return "关闭"
	}
	isNSB := strings.EqualFold(strings.TrimSpace(cfg.mode), "nsb")

	scanLabel := "TCPing"
	if strings.EqualFold(strings.TrimSpace(cfg.scanMode), "httping") {
		scanLabel = "HTTPing"
	}
	speedURLLabel := speedTestURL
	if isAutoSpeedURL(speedTestURL) {
		speedURLLabel = "自动选择"
	}
	dnsLabel := "系统默认"
	if customDNSForced {
		dnsLabel = customDNSServer
	} else if strings.TrimSpace(customDNSServer) != "" && customDNSServer != defaultDNSServers {
		dnsLabel = "系统默认（回退 " + customDNSServer + "）"
	}
	ipVersionLabel := "IPv4"
	if cfg.ipType == 6 {
		ipVersionLabel = "IPv6"
	}
	outputRows := func() {
		line("开始行", strconv.Itoa(cfg.startRow))
		if cfg.endRow > 0 {
			line("结束行", strconv.Itoa(cfg.endRow))
			line("输出总量", fmt.Sprintf("%d行", cfg.endRow-cfg.startRow+1))
		} else {
			line("结束行", "不限")
			line("输出总量", "不限")
		}
	}

	fmt.Println(colorize("----------------------------------------", ansiCyan))
	fmt.Println(colorize("当前配置", ansiBold+ansiCyan))
	if isNSB {
		line("模式", "非标优选")
		line("扫描方式", scanLabel)
		line("输出文件名", cfg.outFile)
		line("文件格式", cfg.export.Format)
		line("CSV 精简", boolLabel(cfg.compactNSB))
		outputRows()
		if strings.TrimSpace(cfg.file) != "" {
			line("输入文件", cfg.file)
		} else {
			line("网络URL", cfg.sourceURL)
		}
		if cfg.nsbFallbackPort > 0 {
			line("备用端口", strconv.Itoa(cfg.nsbFallbackPort))
		} else {
			line("备用端口", "自动")
		}
		line("并发数量", strconv.Itoa(cfg.threads))
		line("扫描合格延迟", fmt.Sprintf("%dms", cfg.delay))
		line("扫描合格数量", strconv.Itoa(cfg.resultLimit))
		line("DNS", dnsLabel)
		dcLabel := strings.TrimSpace(cfg.nsbDC)
		if dcLabel == "" {
			dcLabel = "不限"
		}
		line("数据中心", dcLabel)
		line("TLS 模式", boolLabel(cfg.enableTLS))
		line("测速网址", speedURLLabel)
		line("自动测速", boolLabel(cfg.speedTest > 0 && cfg.nsbSpeedLimit > 0))
		line("测速并发", strconv.Itoa(cfg.speedTest))
		line("测速结果数量", strconv.Itoa(cfg.nsbSpeedLimit))
		line("测试合格速度", fmt.Sprintf("%.2fMB/s", cfg.nsbSpeedMin))
	} else {
		line("模式", "官方优选")
		line("扫描方式", scanLabel)
		line("IP类型", ipVersionLabel)
		line("输出文件名", cfg.outFile)
		line("文件格式", cfg.export.Format)
		outputRows()
		line("并发数量", strconv.Itoa(cfg.threads))
		line("扫描合格延迟", fmt.Sprintf("%dms", cfg.delay))
		line("DNS", dnsLabel)
		line("测试端口", strconv.Itoa(cfg.port))
		dcLabel := strings.TrimSpace(cfg.dc)
		if dcLabel == "" {
			dcLabel = "自动"
		}
		line("数据中心", dcLabel)
		line("测速网址", speedURLLabel)
		line("自动测速", boolLabel(cfg.speedLimit > 0))
		line("测速结果数量", strconv.Itoa(cfg.speedLimit))
		line("测试合格速度", fmt.Sprintf("%.2fMB/s", cfg.speedMin))
	}
	fmt.Println(colorize("----------------------------------------", ansiCyan))
}

func printCLIUsage() {
	printBanner()
	fmt.Fprintf(flag.CommandLine.Output(), "%s\n", colorize("CFData 命令行帮助", ansiBold+ansiGreen))
	fmt.Fprintf(flag.CommandLine.Output(), "版本: %s\n", appVersion)
	checkAndPrintUpdate("")
	fmt.Fprintf(flag.CommandLine.Output(), "\n")
	fmt.Fprintf(flag.CommandLine.Output(), "默认行为: 不带 -cli 时启动 Web 服务；-cli 进入交互菜单；-cli qs 按配置快速启动\n")
	fmt.Fprintf(flag.CommandLine.Output(), "CLI 用法: 程序名 -cli（菜单）或 -cli qs（快速启动）或 -cli -mode=official ...（直接执行）\n")
	fmt.Fprintf(flag.CommandLine.Output(), "\n")
	printCLIUsageGroup("通用参数", cliCommonFlags)
	printCLIUsageGroup("官方模式参数", cliOfficialFlags)
	printCLIUsageGroup("非标模式参数", cliNSBFlags)
}

func printCLIUsageGroup(title string, rows []cliFlagInfo) {
	fmt.Fprintf(flag.CommandLine.Output(), "%s\n", colorize("----------------------------------------", ansiCyan))
	fmt.Fprintf(flag.CommandLine.Output(), "%s\n", colorize(title, ansiBold+ansiCyan))
	for _, row := range rows {
		fmt.Fprintf(flag.CommandLine.Output(), "%s-%s%s\n", ansiBold, row.name, ansiReset)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s %s\n", colorize("说明:", ansiYellow), row.description)
		fmt.Fprintf(flag.CommandLine.Output(), "  %s %s\n", colorize("默认:", ansiYellow), colorizeCLIDefaultValue(row.defaultValue))
	}
}

func lookupCLIFlagDescription(rows []cliFlagInfo, name string) string {
	for _, row := range rows {
		if row.name == name {
			return row.description
		}
	}
	return ""
}

func colorize(text string, code string) string {
	if text == "" {
		return text
	}
	return code + text + ansiReset
}

func colorizeLatencyString(latency string) string {
	ms, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(latency), " ms"))
	if err != nil {
		return latency
	}
	return colorizeLatencyMS(ms)
}

func colorizeLatencyMS(ms int) string {
	text := fmt.Sprintf("%dms", ms)
	if ms <= 50 {
		return colorize(text, ansiGreen)
	}
	if ms <= 100 {
		return colorize(text, ansiBrightGreen)
	}
	if ms <= 200 {
		return colorize(text, ansiYellow)
	}
	if ms <= 250 {
		return colorize(text, ansiYellow)
	}
	if ms <= 3000 {
		return colorize(text, ansiYellow)
	}
	return colorize(text, ansiRed)
}

func colorizeLossRate(lossRate float64) string {
	text := fmt.Sprintf("%.0f%%", lossRate*100)
	if lossRate <= 0 {
		return colorize(text, ansiGreen)
	}
	if lossRate < 0.5 {
		return colorize(text, ansiYellow)
	}
	return colorize(text, ansiRed)
}

func colorizeSpeedString(speed string) string {
	if strings.Contains(speed, "MB/s") {
		value, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(speed, "MB/s")), 64)
		if err == nil {
			if value > 10 {
				return colorize(speed, ansiGreen)
			}
			return colorize(speed, ansiYellow)
		}
	}
	if strings.Contains(strings.ToLower(speed), "错误") || strings.Contains(speed, "失败") || strings.Contains(speed, "0MB/s") {
		return colorize(speed, ansiRed)
	}
	return speed
}

func colorizeCLIParamValue(value string, defaultValue string) string {
	if value == defaultValue {
		if value == "" {
			return colorize("<空>", ansiGreen)
		}
		return colorize(value, ansiGreen)
	}
	if value == "" {
		return colorize("<空>", ansiYellow)
	}
	if value == "true" {
		return colorize(value, ansiGreen)
	}
	if value == "false" {
		return colorize(value, ansiRed)
	}
	return colorize(value, ansiMagenta)
}

func colorizeCLIDefaultValue(value string) string {
	if value == "" {
		return colorize("<空>", ansiYellow)
	}
	return colorize(value, ansiGreen)
}

func setCLIProgress(session *appSession, phase string, current int, total int) {
	session.progressMutex.Lock()
	defer session.progressMutex.Unlock()
	if session.progressState == nil {
		session.progressState = map[string][2]int{}
	}
	state := session.progressState[phase]
	if total <= 0 {
		total = state[1]
	}
	if current < state[0] {
		current = state[0]
	}
	session.progressState[phase] = [2]int{current, total}
}

func maybePrintCLIProgress(session *appSession, phase string, current, total int) {
	if total <= 0 {
		return
	}
	session.progressMutex.Lock()
	if session.progressPrintTime == nil {
		session.progressPrintTime = map[string]time.Time{}
	}
	if session.progressPrintPercent == nil {
		session.progressPrintPercent = map[string]float64{}
	}
	now := time.Now()
	percent := float64(current) / float64(total) * 100
	lastTime := session.progressPrintTime[phase]
	lastPercent := session.progressPrintPercent[phase]

	shouldPrint := false
	switch {
	case lastTime.IsZero():
		shouldPrint = true
	case current >= total:
		shouldPrint = true
	case percent-lastPercent >= 5.0:
		shouldPrint = true
	case now.Sub(lastTime) >= 3*time.Second:
		shouldPrint = true
	}

	if shouldPrint {
		session.progressPrintTime[phase] = now
		session.progressPrintPercent[phase] = percent
	}
	session.progressMutex.Unlock()

	if !shouldPrint {
		return
	}
	fmt.Printf("%s[%s-progress]%s %s\n", ansiCyan, phase, ansiReset, colorize(fmt.Sprintf("[%d/%d %.2f%%]", current, total, percent), ansiCyan))
}

func renderCLIProgress(session *appSession, phase string) string {
	session.progressMutex.Lock()
	defer session.progressMutex.Unlock()
	if session.progressState == nil {
		return colorize("[0/0]", ansiCyan)
	}
	state, ok := session.progressState[phase]
	if !ok {
		return colorize("[0/0]", ansiCyan)
	}
	if state[1] <= 0 {
		return colorize(fmt.Sprintf("[%d/0]", state[0]), ansiCyan)
	}
	percent := float64(state[0]) / float64(state[1]) * 100
	return colorize(fmt.Sprintf("[%d/%d %.2f%%]", state[0], state[1], percent), ansiCyan)
}

func advanceCLIProgress(session *appSession, phase string) string {
	session.progressMutex.Lock()
	defer session.progressMutex.Unlock()
	if session.progressState == nil {
		session.progressState = map[string][2]int{}
	}
	state := session.progressState[phase]
	if state[1] > 0 && state[0] < state[1] {
		state[0]++
		session.progressState[phase] = state
	}
	if state[1] <= 0 {
		return colorize(fmt.Sprintf("[%d/0]", state[0]), ansiCyan)
	}
	percent := float64(state[0]) / float64(state[1]) * 100
	return colorize(fmt.Sprintf("[%d/%d %.2f%%]", state[0], state[1], percent), ansiCyan)
}

func asInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		value, err := strconv.Atoi(n)
		if err == nil {
			return value
		}
	}
	return 0
}

func writeCLIExportAndMaybeUpload(cfg *cliConfig, rows []cliResultRow, mode string) error {
	if len(rows) == 0 {
		return nil
	}
	content, err := formatCLIResults(rows, cfg.export)
	if err != nil {
		return err
	}
	filename := cfg.outFile
	if strings.TrimSpace(filename) == "" {
		filename = "cfdata-results." + cfg.export.Format
	}
	if ext := "." + cfg.export.Format; !strings.HasSuffix(strings.ToLower(filename), ext) {
		filename = strings.TrimSuffix(filename, filepath.Ext(filename)) + ext
	}
	filename = safeFilename(filename)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := writeUTF8BOM(file); err != nil {
		os.Remove(filename)
		return err
	}

	if _, err := io.WriteString(file, content); err != nil {
		return err
	}
	fmt.Printf("[%s-output] %s (%d rows, %s)\n", mode, filename, len(rows), cfg.export.Format)
	if cfg.export.GitHub {
		if err := uploadCLIExportToGitHub(cfg, content); err != nil {
			return err
		}
	}
	if cfg.export.EdgeTunnel {
		if err := uploadCLIExportToEdgetunnel(cfg, content); err != nil {
			return err
		}
	}
	return nil
}

func resolveCLITextSeparator(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	switch strings.ToLower(value) {
	case "空格", "space":
		return " "
	case "\\t", "tab":
		return "\t"
	}
	return value
}

func formatCLIResults(rows []cliResultRow, cfg cliExportConfig) (string, error) {
	customFields := parseCLICustomFields(cfg.Custom)
	fields := resolveCLIFields(cfg.Fields, cfg.Format, rows, customFields)
	rows = applyCLICustomFields(rows, customFields)
	if cfg.Format == "txt" {
		separator := resolveCLITextSeparator(cfg.Separator)
		var b strings.Builder
		for _, row := range rows {
			ipport := row["ipport"]
			if ip := strings.TrimSpace(row["ip"]); ip != "" {
				if cfg.V6Bracket && strings.Contains(ip, ":") {
					ipport = "[" + ip + "]:" + row["port"]
				} else {
					ipport = ip + ":" + row["port"]
				}
			} else if ipport == "" {
				ipport = row["ip"] + ":" + row["port"]
			}
			extras := make([]string, 0, len(fields))
			for _, field := range fields {
				if field == "ipport" {
					continue
				}
				if value := strings.TrimSpace(row[field]); value != "" {
					extras = append(extras, value)
				}
			}
			b.WriteString(ipport)
			if len(extras) > 0 {
				b.WriteString("#")
				b.WriteString(strings.Join(extras, separator))
			}
			b.WriteString("\n")
		}
		return b.String(), nil
	}

	var b strings.Builder
	writer := csv.NewWriter(&b)
	headers := make([]string, 0, len(fields))
	for _, field := range fields {
		headers = append(headers, cliFieldLabel(field, customFields))
	}
	if err := writer.Write(headers); err != nil {
		return "", err
	}
	for _, row := range rows {
		values := make([]string, 0, len(fields))
		for _, field := range fields {
			values = append(values, row[field])
		}
		if err := writer.Write(values); err != nil {
			return "", err
		}
	}
	writer.Flush()
	return b.String(), writer.Error()
}

func resolveCLIFields(spec, format string, rows []cliResultRow, customFields []cliCustomField) []string {
	spec = strings.TrimSpace(strings.ToLower(spec))
	customByKey := map[string]bool{}
	for _, field := range customFields {
		customByKey[field.Key] = true
	}
	appendCustomFields := func(fields []string) []string {
		seen := map[string]bool{}
		result := make([]string, 0, len(fields)+len(customFields))
		for _, field := range fields {
			if field == "" || seen[field] {
				continue
			}
			result = append(result, field)
			seen[field] = true
		}
		for _, field := range customFields {
			if !seen[field.Key] {
				result = append(result, field.Key)
			}
		}
		return result
	}
	if spec == "" || spec == "compact" {
		if format == "txt" {
			return appendCustomFields([]string{"ipport", "dc", "loc"})
		}
		if rowsAreOfficial(rows) {
			if rowsHaveField(rows, "speed") {
				return appendCustomFields([]string{"ip", "port", "latency", "speed", "dc", "region", "city"})
			}
			return appendCustomFields([]string{"ip", "port", "latency", "dc", "region", "city"})
		}
		return appendCustomFields([]string{"ip", "port", "tls", "latency", "speed", "outboundIP", "ipType", "originalInput", "dc", "loc", "region", "city", "asnNumber", "asnOrg"})
	}
	if spec == "ipport" {
		return appendCustomFields([]string{"ipport"})
	}
	if spec == "all" {
		fields := make([]string, 0, len(cliResultFields))
		for _, field := range cliResultFields {
			if field.Key == "ipport" {
				continue
			}
			for _, row := range rows {
				if strings.TrimSpace(row[field.Key]) != "" {
					fields = append(fields, field.Key)
					break
				}
			}
		}
		return appendCustomFields(fields)
	}
	parts := strings.Split(spec, ",")
	fields := make([]string, 0, len(parts))
	valid := map[string]string{"ipport": "ipport"}
	for _, field := range cliResultFields {
		valid[strings.ToLower(field.Key)] = field.Key
	}
	for _, part := range parts {
		field := strings.TrimSpace(part)
		fieldLower := strings.ToLower(field)
		if field != "" && (valid[fieldLower] != "" || customByKey[fieldLower]) {
			if customByKey[fieldLower] {
				field = fieldLower
			} else {
				field = valid[fieldLower]
			}
			fields = append(fields, field)
		}
	}
	if len(fields) == 0 {
		return appendCustomFields([]string{"ipport"})
	}
	return appendCustomFields(fields)
}

func rowsAreOfficial(rows []cliResultRow) bool {
	if len(rows) == 0 {
		return false
	}
	for _, row := range rows {
		if hasAnyCLIField(row, "tls", "outboundIP", "ipType", "loc", "asnNumber", "asnOrg", "visitScheme", "tlsVersion", "sni", "httpVersion", "warp", "gateway", "rbi", "kex", "timestamp") {
			return false
		}
	}
	return true
}

func rowsHaveField(rows []cliResultRow, field string) bool {
	for _, row := range rows {
		if strings.TrimSpace(row[field]) != "" {
			return true
		}
	}
	return false
}

func hasAnyCLIField(row cliResultRow, fields ...string) bool {
	for _, field := range fields {
		if strings.TrimSpace(row[field]) != "" {
			return true
		}
	}
	return false
}

func cliFieldLabel(key string, customFields []cliCustomField) string {
	for _, field := range customFields {
		if field.Key == key {
			return field.Label
		}
	}
	for _, field := range cliResultFields {
		if field.Key == key {
			return field.Label
		}
	}
	return key
}

func parseCLICustomFields(spec string) []cliCustomField {
	parts := strings.Split(spec, ",")
	fields := make([]cliCustomField, 0, len(parts))
	seen := map[string]int{}
	for _, part := range parts {
		item := strings.TrimSpace(part)
		key := ""
		if keyValue := strings.SplitN(item, "=", 2); len(keyValue) == 2 {
			key = strings.ToLower(strings.TrimSpace(keyValue[0]))
			item = strings.TrimSpace(keyValue[1])
		}
		labelValue := strings.SplitN(item, ":", 2)
		if len(labelValue) != 2 || strings.TrimSpace(labelValue[0]) == "" || strings.TrimSpace(labelValue[1]) == "" {
			continue
		}
		label := strings.TrimSpace(labelValue[0])
		if label == "" {
			label = key
		}
		if key == "" {
			key = strings.ToLower(label)
		}
		baseKey := key
		if count := seen[baseKey]; count > 0 {
			for {
				key = fmt.Sprintf("%s%d", baseKey, count)
				if seen[key] == 0 {
					break
				}
				count++
			}
		}
		fields = append(fields, cliCustomField{Key: key, Label: label, Value: strings.TrimSpace(labelValue[1])})
		seen[baseKey]++
		if key != baseKey {
			seen[key]++
		}
	}
	return fields
}

func applyCLICustomFields(rows []cliResultRow, fields []cliCustomField) []cliResultRow {
	if len(fields) == 0 {
		return rows
	}
	for _, row := range rows {
		for _, field := range fields {
			row[field.Key] = field.Value
		}
	}
	return rows
}

func officialScanRows(scanResults []ScanResult, scanMode string) []cliResultRow {
	rows := make([]cliResultRow, 0, len(scanResults))
	modeLabel := scanModeLabel(scanMode)
	for _, res := range scanResults {
		rows = append(rows, cliResultRow{"ip": res.IP, "port": strconv.Itoa(res.Port), "ipport": fmt.Sprintf("%s:%d", res.IP, res.Port), "dc": res.DataCenter, "dcCountry": res.DCCountry, "region": res.Region, "city": res.City, "latency": res.LatencyStr, "scanMode": modeLabel})
	}
	return rows
}

func officialResultRows(scanResults []ScanResult, testResults []TestResult, scanMode string) []cliResultRow {
	if len(testResults) == 0 {
		rows := officialScanRows(scanResults, scanMode)
		sortOfficialRows(rows)
		return rows
	}
	scanByIP := make(map[string]ScanResult, len(scanResults))
	for _, res := range scanResults {
		scanByIP[res.IP] = res
	}
	modeLabel := scanModeLabel(scanMode)
	rows := make([]cliResultRow, 0, len(testResults))
	for _, res := range testResults {
		scan := scanByIP[res.IP]
		port := scan.Port
		if port == 0 {
			port = res.Port
		}
		rows = append(rows, cliResultRow{"ip": res.IP, "port": strconv.Itoa(port), "ipport": fmt.Sprintf("%s:%d", res.IP, port), "dc": scan.DataCenter, "dcCountry": scan.DCCountry, "region": scan.Region, "city": scan.City, "latency": fmt.Sprintf("%dms", res.AvgLatency/time.Millisecond), "speed": res.Speed, "scanMode": modeLabel})
		if rows[len(rows)-1]["dc"] == "" {
			rows[len(rows)-1]["dc"] = res.DataCenter
			rows[len(rows)-1]["dcCountry"] = res.DCCountry
			rows[len(rows)-1]["region"] = res.Region
			rows[len(rows)-1]["city"] = res.City
		}
	}
	sortOfficialRows(rows)
	return rows
}

func sortOfficialRows(rows []cliResultRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		speedI, okI := parseSpeedMBForSort(rows[i]["speed"])
		speedJ, okJ := parseSpeedMBForSort(rows[j]["speed"])
		if okI != okJ {
			return okI
		}
		if okI && speedI != speedJ {
			return speedI > speedJ
		}
		latencyI := parseLatencyMSForSort(rows[i]["latency"])
		latencyJ := parseLatencyMSForSort(rows[j]["latency"])
		if latencyI != latencyJ {
			return latencyI < latencyJ
		}
		return rows[i]["ip"] < rows[j]["ip"]
	})
}

func parseSpeedMBForSort(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if !strings.Contains(value, "MB/s") {
		return 0, false
	}
	speed, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "MB/s")), 64)
	if err != nil {
		return 0, false
	}
	return speed, true
}

func parseLatencyMSForSort(value string) float64 {
	latency, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "ms")), 64)
	if err != nil {
		return math.MaxFloat64
	}
	return latency
}

func nsbPayloadRows(headers []string, rows [][]string) []cliResultRow {
	result := make([]cliResultRow, 0, len(rows))
	for _, row := range rows {
		item := cliResultRow{}
		for idx, header := range headers {
			if idx >= len(row) {
				continue
			}
			if key := cliFieldKeyFromHeader(header); key != "" {
				item[key] = row[idx]
			}
		}
		item["ipport"] = item["ip"] + ":" + item["port"]
		result = append(result, item)
	}
	return result
}

func filterCLIResultRowsByIPType(rows []cliResultRow, filter string) []cliResultRow {
	filter = normalizeIPTypeFilter(filter)
	if filter == "all" {
		return rows
	}
	filtered := make([]cliResultRow, 0, len(rows))
	for _, row := range rows {
		if strings.EqualFold(cliRowIPType(row), filter) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func cliRowIPType(row cliResultRow) string {
	if value := strings.TrimSpace(row["ipType"]); value != "" {
		return strings.ToLower(value)
	}
	if strings.Contains(row["ip"], ":") {
		return "ipv6"
	}
	return "ipv4"
}

func filterCLIResultRowsByQualified(rows []cliResultRow, filter string, speedMin float64) []cliResultRow {
	filter, ok := normalizeCLIQualifiedFilter(filter)
	if !ok || filter == "all" {
		return rows
	}
	filtered := make([]cliResultRow, 0, len(rows))
	for _, row := range rows {
		if getNSBSpeedStatus(row["speed"], speedMin) == filter {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func normalizeCLIQualifiedFilter(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all":
		return "all", true
	case "qualified", "合格":
		return "qualified", true
	case "unqualified", "不合格":
		return "unqualified", true
	}
	return "", false
}

func sliceCLIResultRowRange(rows []cliResultRow, start, end int) []cliResultRow {
	if start < 1 {
		start = 1
	}
	if start > len(rows) {
		return nil
	}
	if end <= 0 || end > len(rows) {
		end = len(rows)
	}
	if end < start {
		return nil
	}
	return rows[start-1 : end]
}

func applyCLIExportFilters(cfg *cliConfig, rows []cliResultRow, speedMin float64) []cliResultRow {
	rows = filterCLIResultRowsByIPType(rows, cfg.outIPType)
	rows = filterCLIResultRowsByQualified(rows, cfg.outQualified, speedMin)
	return sliceCLIResultRowRange(rows, cfg.startRow, cfg.endRow)
}

func normalizeIPTypeFilter(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "all", "全部", "全部展示":
		return "all"
	case "4", "ipv4":
		return "ipv4"
	case "6", "ipv6":
		return "ipv6"
	default:
		return ""
	}
}

func cliFieldKeyFromHeader(header string) string {
	header = normalizeNSBHeaderForCLI(header)
	for _, field := range cliResultFields {
		if field.Label == header {
			return field.Key
		}
	}
	return ""
}

func normalizeNSBHeaderForCLI(header string) string {
	switch strings.TrimSpace(header) {
	case "IP":
		return "IP地址"
	case "端口":
		return "端口号"
	default:
		return strings.TrimSpace(header)
	}
}

func uploadCLIExportToGitHub(cfg *cliConfig, content string) error {
	parts := strings.Split(strings.TrimSpace(cfg.export.GHRepo), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("-ghrepo 必须是 owner/repo")
	}
	if strings.TrimSpace(cfg.export.GHToken) == "" {
		return fmt.Errorf("启用 -github 时需要 -ghtoken 或 -ghtokenfile")
	}
	params := githubUploadRequest{Token: cfg.export.GHToken, Owner: parts[0], Repo: parts[1], Branch: cfg.export.GHBranch, Path: cfg.export.GHPath, Message: cfg.export.GHMessage, Content: content}
	downloadURL, err := uploadGitHubContentWithRetry(context.Background(), params, func(attempt, total int, err error) {
		if err == nil {
			fmt.Printf("%s[github]%s upload attempt %d/%d\n", ansiYellow, ansiReset, attempt, total)
			return
		}
		fmt.Printf("%s[github]%s upload attempt %d/%d failed: %s\n", ansiRed, ansiReset, attempt, total, err.Error())
	})
	if err != nil {
		return err
	}
	fmt.Printf("%s[github]%s uploaded %s\n", ansiGreen, ansiReset, downloadURL)
	return nil
}

func uploadCLIExportToEdgetunnel(cfg *cliConfig, content string) error {
	if strings.TrimSpace(cfg.export.ETHost) == "" {
		return fmt.Errorf("启用 -edgetunnel 时需要 -ethost")
	}
	if strings.TrimSpace(cfg.export.ETPassword) == "" {
		return fmt.Errorf("启用 -edgetunnel 时需要 -etpassword")
	}
	mode := cfg.export.ETMode
	if mode == "" {
		mode = "overwrite"
	}
	params := edgetunnelUploadRequest{
		Host:     cfg.export.ETHost,
		Password: cfg.export.ETPassword,
		Content:  content,
		Mode:     mode,
	}
	err := uploadToEdgetunnel(context.Background(), params, func(msg string) {
		fmt.Printf("%s[edgetunnel]%s %s\n", ansiYellow, ansiReset, msg)
	})
	if err != nil {
		return err
	}
	fmt.Printf("%s[edgetunnel]%s uploaded\n", ansiGreen, ansiReset)
	return nil
}
