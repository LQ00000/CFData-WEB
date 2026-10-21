package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/oschwald/maxminddb-golang"
)

const (
	asnDBFile = "GeoLite2-ASN.mmdb"
	asnDBURL  = "https://jsd.onmicrosoft.cn/gh/seketiti/GeoLiet2@release/GeoLite2-ASN.mmdb"
)

type asnRecord struct {
	AutonomousSystemNumber       uint   `maxminddb:"autonomous_system_number"`
	AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
}

var asnLookupState = struct {
	sync.Mutex
	reader *maxminddb.Reader
	err    error
	loaded bool
}{}

func lookupASN(ipText string) (string, string) {
	reader, err := getASNReader()
	if err != nil || reader == nil {
		return "", ""
	}
	ip := net.ParseIP(ipText)
	if ip == nil {
		return "", ""
	}
	var record asnRecord
	if err := reader.Lookup(ip, &record); err != nil {
		return "", ""
	}
	if record.AutonomousSystemNumber == 0 && record.AutonomousSystemOrganization == "" {
		return "", ""
	}
	return fmt.Sprintf("AS%d", record.AutonomousSystemNumber), record.AutonomousSystemOrganization
}

func getASNReader() (*maxminddb.Reader, error) {
	asnLookupState.Lock()
	defer asnLookupState.Unlock()
	if asnLookupState.loaded {
		return asnLookupState.reader, asnLookupState.err
	}
	asnLookupState.loaded = true
	if _, err := os.Stat(asnDBFile); os.IsNotExist(err) {
		sendLog("本地 GeoLite2-ASN.mmdb 不存在，正在下载...")
		content, err := getURLBytesViaProxy(context.Background(), asnDBURL)
		if err != nil {
			asnLookupState.err = err
			sendLog("下载 ASN 数据库失败: " + err.Error())
			return nil, err
		}
		if err := atomicWriteFile(asnDBFile, content, 0644); err != nil {
			asnLookupState.err = err
			sendLog("保存 ASN 数据库失败: " + err.Error())
			return nil, err
		}
	}
	reader, err := maxminddb.Open(asnDBFile)
	if err != nil {
		asnLookupState.err = err
		sendLog("打开 ASN 数据库失败: " + err.Error())
		return nil, err
	}
	asnLookupState.reader = reader
	return reader, nil
}

func resetASNReader() {
	asnLookupState.Lock()
	defer asnLookupState.Unlock()
	if asnLookupState.reader != nil {
		asnLookupState.reader.Close()
	}
	asnLookupState.reader = nil
	asnLookupState.err = nil
	asnLookupState.loaded = false
}
