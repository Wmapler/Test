package monitor

import (
	"strings"
	"time"
	"yulong-hids/agent/common"
)

// StartNetSniff 开始网络行为监控
func StartNetSniff(resultChan chan map[string]string) {
	var resultant map[string]string
	for {
		time.Sleep(time.Minute)
		cmdStrings := common.Cmdexec("netstat -tunp")
		cmdExecString := strings.Split(cmdStrings, "\n")
		if len(cmdExecString) == 0 {
			continue
		}

		for _, cmdStr := range cmdExecString {
			fields := strings.Fields(cmdStr)
			if len(fields) == 0 {
				continue
			}

			if len(fields) < 7 {
				continue
			}

			if fields[5] != "ESTABLISHED" {
				continue
			}

			resultant = map[string]string{
				"source":   "connection",
				"dir":      "out",
				"protocol": "",
				"remote":   "",
				"local":    "",
				"pid":      "",
				"name":     "",
			}

			src, dst := fields[3], fields[4]

			srcIp := strings.Split(src, ":")[0]
			dstIp := strings.Split(dst, ":")[0]

			if strings.Contains(src, "127.0.0.1") && strings.Contains(dst, "127.0.0.1") {
				continue
			}
			if common.InArray(common.ServerIPList, srcIp, false) || common.InArray(common.ServerIPList, dstIp, false) {
				continue
			}
			if srcIp == "127.0.0.1" {
				strings.Replace(src, "127.0.0.1", common.LocalIP, -1)
			}
			if dstIp == "127.0.0.1" {
				strings.Replace(dst, "127.0.0.1", common.LocalIP, -1)
				resultant["dir"] = "in"
			}

			resultant["protocol"] = fields[0]
			resultant["local"] = src
			resultant["remote"] = dst

			splits := strings.SplitN(fields[6], "/", 2)

			if len(splits) > 1 {
				resultant["pid"] = splits[0]
				resultant["name"] = splits[1]
			}

			resultChan <- resultant
		}
	}
}
