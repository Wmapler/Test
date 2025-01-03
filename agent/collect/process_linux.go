package collect

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"yulong-hids/agent/common"
)

type PsEoResult struct {
	ID      string
	CpuPerc string
	MemPerc string
	Command string
}

func GetProcessList() (resultData []map[string]string) {
	var dirs []string
	var err error
	dirs, err = dirsUnder("/proc")
	if err != nil || len(dirs) == 0 {
		return
	}

	var psResult = pseoResult()
	//log.Printf("ps result: %v\n", psResult)

	for _, v := range dirs {
		pid, err := strconv.Atoi(v)
		if err != nil {
			continue
		}
		statusInfo := getStatus(pid)
		command := getcmdline(pid)
		m := make(map[string]string)
		m["pid"] = v
		m["ppid"] = statusInfo["PPid"]
		m["name"] = statusInfo["Name"]
		m["command"] = command

		if _, ok := psResult[v]; ok {
			m["cpuPerc"] = psResult[v].CpuPerc
			m["memPerc"] = psResult[v].MemPerc
		} else {
			continue
		}

		resultData = append(resultData, m)
	}
	return
}

func pseoResult() map[string]PsEoResult {
	var result = make(map[string]PsEoResult)
	psStrings := common.Cmdexec("ps -eo pid,%cpu,%mem,comm")
	psString := strings.Split(psStrings, "\n")
	//log.Printf("psStrings: %v\n", psStrings)

	for i := 1; i < len(psString); i++ {
		ps := strings.Fields(psString[i])
		if len(ps) < 4 {
			continue
		}
		pid := ps[0]
		cpuPerc := ps[1]
		memPerc := ps[2]
		command := ps[3]
		result[pid] = PsEoResult{pid, cpuPerc, memPerc, command}
	}

	return result
}

func getcmdline(pid int) string {
	cmdlineFile := fmt.Sprintf("/proc/%d/cmdline", pid)
	cmdlineBytes, e := ioutil.ReadFile(cmdlineFile)
	if e != nil {
		return ""
	}
	cmdlineBytesLen := len(cmdlineBytes)
	if cmdlineBytesLen == 0 {
		return ""
	}
	for i, v := range cmdlineBytes {
		if v == 0 {
			cmdlineBytes[i] = 0x20
		}
	}
	return strings.TrimSpace(string(cmdlineBytes))
}
func getStatus(pid int) (status map[string]string) {
	status = make(map[string]string)
	statusFile := fmt.Sprintf("/proc/%d/status", pid)
	var content []byte
	var err error
	content, err = ioutil.ReadFile(statusFile)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, ":") {
			kv := strings.SplitN(line, ":", 2)
			status[kv[0]] = strings.TrimSpace(kv[1])
		}
	}
	return
}

func dirsUnder(dirPath string) ([]string, error) {
	fs, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return []string{}, err
	}

	sz := len(fs)
	if sz == 0 {
		return []string{}, nil
	}
	ret := make([]string, 0, sz)
	for i := 0; i < sz; i++ {
		if fs[i].IsDir() {
			name := fs[i].Name()
			if name != "." && name != ".." {
				ret = append(ret, name)
			}
		}
	}
	return ret, nil
}
