package main

import "encoding/json"

type wsRequest struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type startTaskRequest struct {
	IPType             int     `json:"ipType"`
	Threads            int     `json:"threads"`
	Port               int     `json:"port"`
	Delay              int     `json:"delay"`
	ScanMode           string  `json:"scanMode"`
	AutoSpeed          bool    `json:"autoSpeed"`
	OfficialTargetDC   string  `json:"officialTargetDC"`
	OfficialSpeedPort  int     `json:"officialSpeedPort"`
	OfficialSpeedURL   string  `json:"officialSpeedURL"`
	OfficialSpeedMin   float64 `json:"officialSpeedMin"`
	OfficialSpeedLimit int     `json:"officialSpeedLimit"`
}

type startTestRequest struct {
	DC       string `json:"dc"`
	Port     int    `json:"port"`
	Delay    int    `json:"delay"`
	ScanMode string `json:"scanMode"`
}

type startSpeedTestRequest struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
	URL  string `json:"url"`
}

type startOfficialSpeedBatchRequest struct {
	Port       int          `json:"port"`
	URL        string       `json:"url"`
	SpeedMin   float64      `json:"speedMin"`
	SpeedLimit int          `json:"speedLimit"`
	Results    []TestResult `json:"results"`
	SkipTested bool         `json:"skipTested"`
}

type startNSBTaskRequest struct {
	FileName     string  `json:"fileName"`
	FileContent  string  `json:"fileContent"`
	SourceURL    string  `json:"sourceURL"`
	OutFile      string  `json:"outFile"`
	MaxThreads   int     `json:"maxThreads"`
	FallbackPort int     `json:"fallbackPort"`
	SpeedTest    int     `json:"speedTest"`
	SpeedURL     string  `json:"speedURL"`
	EnableTLS    bool    `json:"enableTLS"`
	Delay        int     `json:"delay"`
	ResultLimit  int     `json:"resultLimit"`
	DC           string  `json:"dc"`
	SpeedMin     float64 `json:"speedMin"`
	SpeedLimit   int     `json:"speedLimit"`
	Compact      bool    `json:"compact"`
	ScanMode     string  `json:"scanMode"`
	RelinkHost   string  `json:"relinkHost"`
}

type startNSBReallinkRequest struct {
	Results    []nsbScanMessage `json:"results"`
	RelinkHost string           `json:"relinkHost"`
	OutFile    string           `json:"outFile"`
	SpeedTest  int              `json:"speedTest"`
	SpeedURL   string           `json:"speedURL"`
	EnableTLS  bool             `json:"enableTLS"`
	SpeedMin   float64          `json:"speedMin"`
	SpeedLimit int              `json:"speedLimit"`
	Compact    bool             `json:"compact"`
	ScanMode   string           `json:"scanMode"`
}

type startNSBSpeedBatchRequest struct {
	Results    []nsbScanMessage `json:"results"`
	SpeedTest  int              `json:"speedTest"`
	SpeedURL   string           `json:"speedURL"`
	EnableTLS  bool             `json:"enableTLS"`
	SpeedMin   float64          `json:"speedMin"`
	SpeedLimit int              `json:"speedLimit"`
	SkipTested bool             `json:"skipTested"`
	Compact    bool             `json:"compact"`
}

type edgetunnelUploadRequest struct {
	Host     string `json:"host"`
	Password string `json:"password"`
	Content  string `json:"content"`
	Mode     string `json:"mode"`
	Silent   bool   `json:"silent"`
}

type githubUploadRequest struct {
	Token   string `json:"token"`
	Owner   string `json:"owner"`
	Repo    string `json:"repo"`
	Branch  string `json:"branch"`
	Path    string `json:"path"`
	Message string `json:"message"`
	Content string `json:"content"`
	Silent  bool   `json:"silent"`
}
