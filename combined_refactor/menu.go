package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	promptOK = iota
	promptBack
	promptQuit
)

var menuInputReader = bufio.NewReader(os.Stdin)

func onlyCLIArgProvided() bool {
	only := true
	flag.Visit(func(f *flag.Flag) {
		if f.Name != "cli" {
			only = false
		}
	})
	return only
}

func readMenuLine() (string, int) {
	line, err := menuInputReader.ReadString('\n')
	trimmed := strings.TrimSpace(line)
	if trimmed == "" && err != nil {
		return "", promptQuit
	}
	return trimmed, promptOK
}

func displayOrEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(空)"
	}
	return value
}

func askLine(label, currentDisplay, hint string) (string, int) {
	fmt.Println()
	fmt.Printf("  %s  当前: %s\n", label, currentDisplay)
	if strings.TrimSpace(hint) != "" {
		fmt.Printf("    提示: %s\n", hint)
	}
	fmt.Print("    直接回车=保持当前，输入 b=返回菜单 > ")
	line, outcome := readMenuLine()
	if outcome != promptOK {
		return "", outcome
	}
	if line == "b" {
		return "", promptBack
	}
	return line, promptOK
}

func askChoice(label, hint string, options []string, currentIdx int) (int, int) {
	fmt.Println()
	if currentIdx < 0 || currentIdx >= len(options) {
		currentIdx = 0
	}
	fmt.Printf("  %s  当前: %s\n", label, options[currentIdx])
	if strings.TrimSpace(hint) != "" {
		fmt.Printf("    提示: %s\n", hint)
	}
	for i, opt := range options {
		fmt.Printf("    %d = %s\n", i+1, opt)
	}
	fmt.Print("    直接回车=保持当前，输入 b=返回菜单 > ")
	for {
		line, outcome := readMenuLine()
		if outcome != promptOK {
			return 0, outcome
		}
		if line == "b" {
			return 0, promptBack
		}
		if line == "" {
			return currentIdx, promptOK
		}
		n, err := strconv.Atoi(line)
		if err == nil && n >= 1 && n <= len(options) {
			return n - 1, promptOK
		}
		fmt.Printf("    无效输入，请输入 1-%d 或直接回车 > ", len(options))
	}
}

func askInt(label, currentDisplay, hint string, current, min, max int) (int, int) {
	fmt.Println()
	fmt.Printf("  %s  当前: %s\n", label, currentDisplay)
	if strings.TrimSpace(hint) != "" {
		fmt.Printf("    提示: %s\n", hint)
	}
	for {
		fmt.Print("    直接回车=保持当前，输入 b=返回菜单 > ")
		line, outcome := readMenuLine()
		if outcome != promptOK {
			return 0, outcome
		}
		if line == "b" {
			return 0, promptBack
		}
		if line == "" {
			return current, promptOK
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < min || (max > 0 && n > max) {
			var rangeHint string
			if max > 0 {
				rangeHint = fmt.Sprintf("%d-%d", min, max)
			} else {
				rangeHint = fmt.Sprintf("≥%d", min)
			}
			fmt.Printf("    无效输入，请输入整数（%s）\n", rangeHint)
			continue
		}
		return n, promptOK
	}
}

func askFloatValue(label, currentDisplay, hint string, current, min float64) (float64, int) {
	fmt.Println()
	fmt.Printf("  %s  当前: %s\n", label, currentDisplay)
	if strings.TrimSpace(hint) != "" {
		fmt.Printf("    提示: %s\n", hint)
	}
	for {
		fmt.Print("    直接回车=保持当前，输入 b=返回菜单 > ")
		line, outcome := readMenuLine()
		if outcome != promptOK {
			return 0, outcome
		}
		if line == "b" {
			return 0, promptBack
		}
		if line == "" {
			return current, promptOK
		}
		f, err := strconv.ParseFloat(line, 64)
		if err != nil || f < min {
			fmt.Printf("    无效输入，请输入数值（≥%g）\n", min)
			continue
		}
		return f, promptOK
	}
}

func askTimeValue(label string) (int, int, int) {
	fmt.Println()
	fmt.Printf("  %s\n", label)
	fmt.Println("    格式: 小时:分钟，如 9:30 或 09:30（00:00-23:59）")
	for {
		fmt.Print("    输入 b=返回菜单 > ")
		line, outcome := readMenuLine()
		if outcome != promptOK {
			return 0, 0, outcome
		}
		if line == "b" {
			return 0, 0, promptBack
		}
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			fmt.Println("    格式错误，请按 9:30 形式输入")
			continue
		}
		hour, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		minute, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 != nil || err2 != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			fmt.Println("    时间无效，小时需在 0-23，分钟需在 0-59")
			continue
		}
		return hour, minute, promptOK
	}
}

func sectionHeader(title string) {
	fmt.Println()
	fmt.Println(colorize("---------- "+title+" ----------", ansiBold+ansiCyan))
}

func boolChoiceIndex(value bool) int {
	if value {
		return 0
	}
	return 1
}

func runCLIMenu(cfg *cliConfig) {
	configPath := defaultCLIConfigPath()
	if strings.TrimSpace(cfg.export.ConfigFile) != "" {
		configPath = cfg.export.ConfigFile
	}
	fileCfg, created, err := loadOrCreateCLIConfig(configPath)
	if err != nil {
		fmt.Printf("[config] 加载配置文件失败: %v\n", err)
		return
	}
	if created {
		fmt.Printf("[config] 已生成默认配置文件: %s\n", configPath)
		fmt.Println("[config] 建议先用菜单 2 或 3 设置参数并保存，之后可用菜单 1 直接启动")
	}
	provided := map[string]bool{"cli": true}
	for {
		status, _ := scheduleStatus()
		fmt.Println(colorize("========== CFDATA-WEB ==========", ansiBold+ansiCyan))
		fmt.Printf("  版本: %s\n", colorize(appVersion, ansiBold+ansiGreen))
		fmt.Println("  1. 启动（按照设置的参数）")
		fmt.Println("  2. 启动（自定义参数）")
		fmt.Println("  3. 修改配置参数")
		fmt.Printf("  4. 定时任务（%s）\n", status)
		fmt.Println("  5. 帮助")
		fmt.Println("  6. 退出")
		fmt.Print("请选择> ")
		line, outcome := readMenuLine()
		if outcome != promptOK {
			return
		}
		switch strings.TrimSpace(line) {
		case "1":
			runWithFileConfig(cfg, fileCfg, configPath, provided)
		case "2":
			wcfg := fileCfg
			result := runSettingsWizard(&wcfg)
			if result == promptQuit {
				return
			}
			if result == promptBack {
				continue
			}
			saveIdx, o := askChoice("是否保存配置到配置文件？（保存后可用菜单 1 按此配置直接启动）", "", []string{"保存", "不保存"}, 0)
			if o == promptQuit {
				return
			}
			if o == promptBack {
				continue
			}
			if saveIdx == 0 {
				if err := writeCLIConfigTemplate(configPath, wcfg); err != nil {
					fmt.Printf("[config] 保存失败: %v\n", err)
				} else {
					fmt.Printf("[config] 已保存: %s\n", configPath)
					fileCfg = wcfg
				}
			}
			startIdx, o := askChoice("是否立即启动？", "", []string{"立即启动", "返回菜单"}, 0)
			if o == promptQuit {
				return
			}
			if o == promptBack {
				continue
			}
			if startIdx == 0 {
				runWithFileConfig(cfg, wcfg, configPath, provided)
			}
		case "3":
			wcfg := fileCfg
			result := runSettingsWizard(&wcfg)
			if result == promptQuit {
				return
			}
			if result == promptBack {
				continue
			}
			saveIdx, o := askChoice("保存修改到配置文件？", "", []string{"保存", "放弃修改"}, 0)
			if o == promptQuit {
				return
			}
			if o == promptBack {
				continue
			}
			if saveIdx == 0 {
				if err := writeCLIConfigTemplate(configPath, wcfg); err != nil {
					fmt.Printf("[config] 保存失败: %v\n", err)
				} else {
					fmt.Printf("[config] 已保存: %s\n", configPath)
					fileCfg = wcfg
				}
			}
		case "4":
			runScheduleMenu()
		case "5":
			printCLIUsage()
		case "6":
			return
		default:
			fmt.Println("请输入 1-6 之间的数字")
		}
	}
}

func runWithFileConfig(cfg *cliConfig, fileCfg cliFileConfig, configPath string, provided map[string]bool) {
	if err := applyCLIExportResolution(cfg, fileCfg, configPath, provided); err != nil {
		fmt.Printf("配置解析失败: %v\n", err)
		return
	}
	if err := validateCLIOutputConfig(cfg); err != nil {
		fmt.Printf("配置校验失败: %v\n", err)
		return
	}
	if cfg.noColor {
		disableANSIColors()
	}
	cfg.configResolved = true
	speedTestWorkers = cfg.speedTest
	if err := runCLI(cfg); err != nil {
		fmt.Printf("CLI 执行失败: %v\n", err)
	}
	fmt.Println()
}

func runSettingsWizard(fileCfg *cliFileConfig) int {
	fmt.Println(colorize("========== 参数设置向导 ==========", ansiBold+ansiCyan))
	fmt.Println("逐项设置参数：每项显示当前值与格式说明，直接回车保持不变，输入 b 返回菜单。")
	var o int

	sectionHeader("基本设置")
	modeIdx := 0
	if strings.EqualFold(strings.TrimSpace(fileCfg.Mode), "nsb") {
		modeIdx = 1
	}
	idx, o := askChoice("模式", "官方优选=内置 CF IP 池扫描；非标优选=自定义目标扫描",
		[]string{"官方优选（official）", "非标优选（nsb）"}, modeIdx)
	if o != promptOK {
		return o
	}
	if idx == 0 {
		fileCfg.Mode = "official"
	} else {
		fileCfg.Mode = "nsb"
	}
	isNSB := idx == 1

	scanIdx := 0
	if strings.EqualFold(strings.TrimSpace(fileCfg.ScanMode), "httping") {
		scanIdx = 1
	}
	idx, o = askChoice("扫描方式", "TCPing=仅 TCP 握手延迟；HTTPing=HTTP TTFB 全链路延迟，数值偏高属正常，两者数据不可互相比较",
		[]string{"TCPing", "HTTPing"}, scanIdx)
	if o != promptOK {
		return o
	}
	if idx == 0 {
		fileCfg.ScanMode = "tcping"
	} else {
		fileCfg.ScanMode = "httping"
	}

	dnsCurrent := "系统默认"
	if strings.TrimSpace(fileCfg.DNS) != "" && fileCfg.DNS != defaultDNSServers {
		dnsCurrent = fileCfg.DNS
	}
	v, o := askLine("DNS", dnsCurrent, "格式: IP[:端口]，多个用逗号分隔（如 223.5.5.5,8.8.8.8:53）；系统 DNS 优先、失败时回退；输入 0=恢复系统默认")
	if o != promptOK {
		return o
	}
	if v == "0" {
		fileCfg.DNS = ""
	} else if v != "" {
		fileCfg.DNS = v
	}

	progIdx := boolChoiceIndex(fileCfg.Progress)
	idx, o = askChoice("进度日志", "运行时是否输出扫描/测速进度", []string{"开启", "关闭"}, progIdx)
	if o != promptOK {
		return o
	}
	fileCfg.Progress = idx == 0

	colorIdx := boolChoiceIndex(fileCfg.NoColor)
	idx, o = askChoice("无颜色输出", "终端出现乱码（如 cmd、部分 SSH 客户端）时建议开启", []string{"开启", "关闭"}, colorIdx)
	if o != promptOK {
		return o
	}
	fileCfg.NoColor = idx == 0

	debugIdx := 0
	switch v := fmt.Sprintf("%v", fileCfg.Debug); v {
	case "true", "error":
		debugIdx = 1
	case "all":
		debugIdx = 2
	}
	idx, o = askChoice("调试等级", "error=记录程序错误与下载/API 异常；all=额外包含测速失败等全部明细，日志写入 debug 目录",
		[]string{"关闭", "error", "all"}, debugIdx)
	if o != promptOK {
		return o
	}
	switch idx {
	case 0:
		fileCfg.Debug = false
	case 1:
		fileCfg.Debug = "error"
	case 2:
		fileCfg.Debug = "all"
	}

	skipGeoIdx := boolChoiceIndex(fileCfg.SkipGeo)
	idx, o = askChoice("跳过代理检测", "开启=运行前不再提示代理/VPN 环境警告，直接开始测试；选择后后续启动仅提示已跳过",
		[]string{"开启", "关闭"}, skipGeoIdx)
	if o != promptOK {
		return o
	}
	fileCfg.SkipGeo = idx == 0

	sectionHeader("扫描与测速")
	if isNSB {
		useURL := strings.TrimSpace(fileCfg.SourceURL) != "" && strings.TrimSpace(fileCfg.File) == ""
		idx, o = askChoice("输入方式", "非标优选的扫描目标来源，二选一",
			[]string{"输入文件（本机文件路径）", "网络URL（http/https 文本地址）"}, boolChoiceIndex(!useURL))
		if o != promptOK {
			return o
		}
		if idx == 0 {
			for {
				v, o := askLine("输入文件", displayOrEmpty(fileCfg.File), "格式: 本机文件路径，如 ip.txt（每行 IP[ 端口]）")
				if o != promptOK {
					return o
				}
				if v == "" {
					if strings.TrimSpace(fileCfg.File) != "" {
						break
					}
					fmt.Println("    不能为空，请输入文件路径")
					continue
				}
				fileCfg.File = v
				break
			}
			fileCfg.SourceURL = ""
		} else {
			for {
				v, o := askLine("网络URL", displayOrEmpty(fileCfg.SourceURL), "格式: http(s):// 地址，返回 IP 列表文本")
				if o != promptOK {
					return o
				}
				if v == "" {
					if strings.TrimSpace(fileCfg.SourceURL) != "" {
						break
					}
					fmt.Println("    不能为空，请输入网络URL")
					continue
				}
				fileCfg.SourceURL = v
				break
			}
			fileCfg.File = ""
		}

		fallbackDisplay := "自动（随 TLS 使用 443/80）"
		if fileCfg.NSBFallbackPort > 0 {
			fallbackDisplay = strconv.Itoa(fileCfg.NSBFallbackPort)
		}
		n, o := askInt("备用端口", fallbackDisplay, "输入缺省端口；0=随 TLS 自动（开启 TLS 用 443，否则 80）", fileCfg.NSBFallbackPort, 0, 65535)
		if o != promptOK {
			return o
		}
		fileCfg.NSBFallbackPort = n

		n, o = askInt("并发数量", strconv.Itoa(fileCfg.NSBThreads), "同时扫描的 IP 数量，越大越快但占用带宽越多", fileCfg.NSBThreads, 1, 0)
		if o != promptOK {
			return o
		}
		fileCfg.NSBThreads = n
	} else {
		n, o := askInt("并发数量", strconv.Itoa(fileCfg.Threads), "同时扫描的 IP 数量，越大越快但占用带宽越多", fileCfg.Threads, 1, 0)
		if o != promptOK {
			return o
		}
		fileCfg.Threads = n
	}

	delayCurrent := fileCfg.Delay
	if isNSB {
		delayCurrent = fileCfg.NSBDelay
	}
	delayMin := 1
	delayHint := "TCPing/HTTPing 延迟超过该值视为不合格（毫秒）"
	if isNSB {
		delayMin = 0
		delayHint = "TCPing/HTTPing 延迟超过该值视为不合格（毫秒）；0=不筛延迟"
	}
	n, o := askInt("扫描合格延迟", fmt.Sprintf("%dms", delayCurrent), delayHint, delayCurrent, delayMin, 0)
	if o != promptOK {
		return o
	}
	if isNSB {
		fileCfg.NSBDelay = n
	} else {
		fileCfg.Delay = n
	}

	if isNSB {
		n, o = askInt("扫描合格数量", strconv.Itoa(fileCfg.ResultLimit), "延迟测试结果达到该数量后停止扫描", fileCfg.ResultLimit, 1, 0)
		if o != promptOK {
			return o
		}
		fileCfg.ResultLimit = n
	} else {
		ipIdx := 0
		if fileCfg.IPType == 6 {
			ipIdx = 1
		}
		idx, o = askChoice("IP类型", "官方优选扫描的 IP 池版本", []string{"IPv4", "IPv6"}, ipIdx)
		if o != promptOK {
			return o
		}
		if idx == 0 {
			fileCfg.IPType = 4
		} else {
			fileCfg.IPType = 6
		}

		n, o = askInt("测试端口", strconv.Itoa(fileCfg.TestPort), "官方扫描目标端口，1-65535", fileCfg.TestPort, 1, 65535)
		if o != promptOK {
			return o
		}
		fileCfg.TestPort = n
	}

	dcCurrent := "自动"
	if isNSB {
		dcCurrent = "不限"
	}
	if strings.TrimSpace(fileCfg.DC) != "" {
		dcCurrent = fileCfg.DC
	}
	if isNSB {
		if strings.TrimSpace(fileCfg.NSBDC) != "" {
			dcCurrent = fileCfg.NSBDC
		}
		v, o := askLine("数据中心", dcCurrent, "格式: 数据中心代码（如 HKG/NRT）；输入 0=不限制结果数据中心")
		if o != promptOK {
			return o
		}
		if v == "0" {
			fileCfg.NSBDC = ""
		} else if v != "" {
			fileCfg.NSBDC = v
		}
	} else {
		v, o := askLine("数据中心", dcCurrent, "格式: 数据中心代码（如 HKG/NRT）；输入 0=自动选择最低延迟数据中心")
		if o != promptOK {
			return o
		}
		if v == "0" {
			fileCfg.DC = ""
		} else if v != "" {
			fileCfg.DC = v
		}
	}

	if isNSB {
		tlsIdx := boolChoiceIndex(fileCfg.TLS)
		idx, o = askChoice("TLS 模式", "开启=HTTPS 连接（缺省端口 443）；关闭=HTTP（缺省端口 80）", []string{"开启", "关闭"}, tlsIdx)
		if o != promptOK {
			return o
		}
		fileCfg.TLS = idx == 0
	}

	speedURL := fileCfg.URL
	if isNSB {
		speedURL = fileCfg.NSBURL
	}
	urlCurrent := "自动选择"
	if !isAutoSpeedURL(speedURL) && strings.TrimSpace(speedURL) != "" {
		urlCurrent = speedURL
	}
	v, o = askLine("测速网址", urlCurrent, "输入 0 或直接回车=自动选择内置测速源；或输入完整 http(s) 下载地址")
	if o != promptOK {
		return o
	}
	if v == "0" {
		speedURL = autoSpeedURLValue
	} else if v != "" {
		speedURL = v
	}
	if isNSB {
		fileCfg.NSBURL = speedURL
	} else {
		fileCfg.URL = speedURL
	}

	if isNSB {
		autoOn := fileCfg.SpeedTest > 0 && fileCfg.NSBSpeedLimit > 0
		idx, o = askChoice("自动测速", "扫描合格后是否进行下载测速（需测速并发与测速结果数量均大于 0）",
			[]string{"开启", "关闭"}, boolChoiceIndex(autoOn))
		if o != promptOK {
			return o
		}
		if idx == 0 {
			if fileCfg.SpeedTest <= 0 {
				fileCfg.SpeedTest = 1
			}
			if fileCfg.NSBSpeedLimit <= 0 {
				fileCfg.NSBSpeedLimit = 5
			}
		} else {
			fileCfg.SpeedTest = 0
		}
		if fileCfg.SpeedTest > 0 && fileCfg.NSBSpeedLimit > 0 {
			n, o = askInt("测速并发", strconv.Itoa(fileCfg.SpeedTest), "同时测速的 IP 数量；多 IP 并发会降低测速准确性，需要准确速度建议 1", fileCfg.SpeedTest, 1, 0)
			if o != promptOK {
				return o
			}
			fileCfg.SpeedTest = n
			n, o = askInt("测速结果数量", strconv.Itoa(fileCfg.NSBSpeedLimit), "测速达标结果达到该数量后停止", fileCfg.NSBSpeedLimit, 1, 0)
			if o != promptOK {
				return o
			}
			fileCfg.NSBSpeedLimit = n
			f, o := askFloatValue("测试合格速度", fmt.Sprintf("%.2fMB/s", fileCfg.NSBSpeedMin), "下载速度低于该值视为不合格（MB/s）", fileCfg.NSBSpeedMin, 0)
			if o != promptOK {
				return o
			}
			fileCfg.NSBSpeedMin = f
		}
	} else {
		idx, o = askChoice("自动测速", "扫描完成后是否对合格结果下载测速；关闭=测速结果数量为 0",
			[]string{"开启", "关闭"}, boolChoiceIndex(fileCfg.SpeedLimit > 0))
		if o != promptOK {
			return o
		}
		if idx == 0 {
			if fileCfg.SpeedLimit <= 0 {
				fileCfg.SpeedLimit = 5
			}
		} else {
			fileCfg.SpeedLimit = 0
		}
		if fileCfg.SpeedLimit > 0 {
			n, o = askInt("测速结果数量", strconv.Itoa(fileCfg.SpeedLimit), "测速达标结果达到该数量后停止", fileCfg.SpeedLimit, 1, 0)
			if o != promptOK {
				return o
			}
			fileCfg.SpeedLimit = n
			f, o := askFloatValue("测试合格速度", fmt.Sprintf("%.2fMB/s", fileCfg.SpeedMin), "下载速度低于该值视为不合格（MB/s）", fileCfg.SpeedMin, 0)
			if o != promptOK {
				return o
			}
			fileCfg.SpeedMin = f
		}
	}

	sectionHeader("输出配置")
	v, o = askLine("输出文件名", displayOrEmpty(fileCfg.Out), "格式: 文件名，如 ip.csv / ip.txt（扩展名按文件格式自动修正）")
	if o != promptOK {
		return o
	}
	if v != "" {
		fileCfg.Out = v
	}

	formatIdx := 1
	if strings.EqualFold(strings.TrimSpace(fileCfg.Format), "csv") {
		formatIdx = 0
	}
	idx, o = askChoice("文件格式", "本地输出与导出的文件格式", []string{"csv", "txt"}, formatIdx)
	if o != promptOK {
		return o
	}
	if idx == 0 {
		fileCfg.Format = "csv"
	} else {
		fileCfg.Format = "txt"
	}

	if isNSB {
		idx, o = askChoice("CSV 精简", "非标模式本地 CSV 是否只保留精简字段", []string{"开启", "关闭"}, boolChoiceIndex(fileCfg.Compact))
		if o != promptOK {
			return o
		}
		fileCfg.Compact = idx == 0
	}

	startDisplay := strconv.Itoa(fileCfg.OutStartRow)
	n, o = askInt("开始行", startDisplay, "筛选后从第几行开始导出（从 1 计）", fileCfg.OutStartRow, 1, 0)
	if o != promptOK {
		return o
	}
	fileCfg.OutStartRow = n
	for {
		endDisplay := "不限"
		if fileCfg.OutEndRow > 0 {
			endDisplay = strconv.Itoa(fileCfg.OutEndRow)
		}
		n, o = askInt("结束行", endDisplay, "导出到第几行（含）；0=不限（导出到末尾）", fileCfg.OutEndRow, 0, 0)
		if o != promptOK {
			return o
		}
		if n > 0 && n < fileCfg.OutStartRow {
			fmt.Println("    结束行不能小于开始行，请重新输入")
			continue
		}
		fileCfg.OutEndRow = n
		break
	}
	totalText := "不限"
	if fileCfg.OutEndRow > 0 {
		totalText = fmt.Sprintf("%d行", fileCfg.OutEndRow-fileCfg.OutStartRow+1)
	}
	fmt.Printf("    当前输出总量: %s\n", totalText)

	v, o = askLine("导出字段", fileCfg.Fields, "可选: compact / all / ipport 或逗号分隔字段 key，如 ipport,latency,dc,loc")
	if o != promptOK {
		return o
	}
	if v != "" {
		fileCfg.Fields = v
	}

	qualIdx := 0
	switch strings.TrimSpace(fileCfg.OutQualified) {
	case "qualified":
		qualIdx = 1
	case "unqualified":
		qualIdx = 2
	}
	idx, o = askChoice("合格筛选", "只影响导出与上传内容；未测速的记录不会计入「仅合格/仅不合格」",
		[]string{"全部结果", "仅合格", "仅不合格"}, qualIdx)
	if o != promptOK {
		return o
	}
	switch idx {
	case 0:
		fileCfg.OutQualified = "all"
	case 1:
		fileCfg.OutQualified = "qualified"
	case 2:
		fileCfg.OutQualified = "unqualified"
	}

	ipFilterIdx := 0
	switch strings.ToLower(strings.TrimSpace(fileCfg.OutIPType)) {
	case "ipv4":
		ipFilterIdx = 1
	case "ipv6":
		ipFilterIdx = 2
	}
	idx, o = askChoice("IP类型筛选", "只影响导出与上传内容", []string{"全部", "仅 IPv4", "仅 IPv6"}, ipFilterIdx)
	if o != promptOK {
		return o
	}
	switch idx {
	case 0:
		fileCfg.OutIPType = "all"
	case 1:
		fileCfg.OutIPType = "ipv4"
	case 2:
		fileCfg.OutIPType = "ipv6"
	}

	if strings.EqualFold(strings.TrimSpace(fileCfg.Format), "txt") {
		v, o = askLine("TXT 分隔符", fileCfg.Separator, "多字段分隔符；输入 空格=空格、tab=制表符，或任意字符（如 |）")
		if o != promptOK {
			return o
		}
		if v != "" {
			fileCfg.Separator = v
		}
	}

	sectionHeader("GitHub 上传")
	ghIdx := boolChoiceIndex(fileCfg.GitHub)
	idx, o = askChoice("GitHub 上传", "导出后是否自动上传结果到 GitHub 仓库", []string{"启用", "不启用"}, ghIdx)
	if o != promptOK {
		return o
	}
	fileCfg.GitHub = idx == 0
	if fileCfg.GitHub {
		for {
			v, o := askLine("GitHub 仓库", displayOrEmpty(fileCfg.GHRepo), "格式: owner/repo，如 cfdata/cfdata")
			if o != promptOK {
				return o
			}
			if v == "" {
				if strings.TrimSpace(fileCfg.GHRepo) != "" {
					break
				}
				fmt.Println("    不能为空，请输入仓库地址")
				continue
			}
			fileCfg.GHRepo = v
			break
		}
		v, o := askLine("GitHub 分支", displayOrEmpty(fileCfg.GHBranch), "默认 main")
		if o != promptOK {
			return o
		}
		if v != "" {
			fileCfg.GHBranch = v
		}
		if strings.TrimSpace(fileCfg.GHBranch) == "" {
			fileCfg.GHBranch = "main"
		}
		pathDisplay := "自动（results/ip.文件格式）"
		if strings.TrimSpace(fileCfg.GHPath) != "" {
			pathDisplay = fileCfg.GHPath
		}
		v, o = askLine("GitHub 目标路径", pathDisplay, "文件不存在会新建、存在会覆盖；输入 0=恢复自动")
		if o != promptOK {
			return o
		}
		if v == "0" {
			fileCfg.GHPath = ""
		} else if v != "" {
			fileCfg.GHPath = v
		}
		v, o = askLine("提交信息", fileCfg.GHMessage, "git commit message")
		if o != promptOK {
			return o
		}
		if v != "" {
			fileCfg.GHMessage = v
		}
		tokenMode := 0
		if strings.TrimSpace(fileCfg.GHTokenFile) != "" {
			tokenMode = 1
		}
		idx, o = askChoice("Token 方式", "强烈建议使用仅限制指定仓库读写权限的 token",
			[]string{"直接输入 Token（写入配置文件）", "从文件读取 Token（更安全）"}, tokenMode)
		if o != promptOK {
			return o
		}
		if idx == 0 {
			tokenDisplay := "未设置"
			if strings.TrimSpace(fileCfg.GHToken) != "" {
				tokenDisplay = "已设置（不显示）"
			}
			v, o := askLine("GitHub Token", tokenDisplay, "输入后回车保存；输入 0=清除已保存的 Token")
			if o != promptOK {
				return o
			}
			if v == "0" {
				fileCfg.GHToken = ""
			} else if v != "" {
				fileCfg.GHToken = v
				fileCfg.GHTokenFile = ""
			}
		} else {
			for {
				v, o := askLine("Token 文件路径", displayOrEmpty(fileCfg.GHTokenFile), "格式: 本机文件路径，文件内容为 Token")
				if o != promptOK {
					return o
				}
				if v == "" {
					if strings.TrimSpace(fileCfg.GHTokenFile) != "" {
						break
					}
					fmt.Println("    不能为空，请输入文件路径")
					continue
				}
				fileCfg.GHTokenFile = v
				fileCfg.GHToken = ""
				break
			}
		}
	}

	sectionHeader("edgetunnel 上传")
	etIdx := boolChoiceIndex(fileCfg.EdgeTunnel)
	idx, o = askChoice("edgetunnel 上传", "导出后是否自动上传到 edgetunnel 订阅", []string{"启用", "不启用"}, etIdx)
	if o != promptOK {
		return o
	}
	fileCfg.EdgeTunnel = idx == 0
	if fileCfg.EdgeTunnel {
		for {
			v, o := askLine("edgetunnel 主机地址", displayOrEmpty(fileCfg.ETHost), "格式: https://example.com")
			if o != promptOK {
				return o
			}
			if v == "" {
				if strings.TrimSpace(fileCfg.ETHost) != "" {
					break
				}
				fmt.Println("    不能为空，请输入主机地址")
				continue
			}
			fileCfg.ETHost = v
			break
		}
		for {
			v, o := askLine("edgetunnel 密码", displayOrEmpty(fileCfg.ETPassword), "登录 edgetunnel 管理页的密码")
			if o != promptOK {
				return o
			}
			if v == "" {
				if strings.TrimSpace(fileCfg.ETPassword) != "" {
					break
				}
				fmt.Println("    不能为空，请输入密码")
				continue
			}
			fileCfg.ETPassword = v
			break
		}
		modeIdx := 1
		if strings.EqualFold(strings.TrimSpace(fileCfg.ETMode), "overwrite") {
			modeIdx = 0
		}
		idx, o = askChoice("edgetunnel 上传模式", "覆盖=替换全部订阅内容；追加=保留原有内容并追加新结果",
			[]string{"覆盖（overwrite）", "追加（append）"}, modeIdx)
		if o != promptOK {
			return o
		}
		if idx == 0 {
			fileCfg.ETMode = "overwrite"
		} else {
			fileCfg.ETMode = "append"
		}
	}

	fmt.Println(colorize("========== 参数设置完成 ==========", ansiBold+ansiCyan))
	return promptOK
}
