//go:build linux
// +build linux

package collect

import (
	"bytes"
	"encoding/json"
	"fmt"
	//"github.com/smallnest/rpcx/log"
	"log"
	"net"
	"strings"
	"os/exec"
	"yulong-hids/agent/common"
)

var macString string = ""

func MacString() string {

	if len(macString) != 0 {
		return macString
	}

	if interfaces, err := net.Interfaces(); err != nil {
		//log.Warnf("invalid net.Interfaces return, %+v", err)
		log.Printf("invalid net.Interfaces return, %+v", err)
		return ""
	} else {
		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			if strings.HasPrefix(iface.Name, "docker") || strings.HasPrefix(iface.Name, "veth") {
				continue
			}
			if bytes.Compare(iface.HardwareAddr, nil) == 0 {
				continue
			}
			macString = iface.HardwareAddr.String()

			if addrs, err := iface.Addrs(); err != nil {
				continue
			} else {
				for _, addr := range addrs {
					if ip, ok := addr.(*net.IPNet); ok {
						if ip.IP.IsLoopback() {
							continue
						}
						if ip.IP.To4() == nil {
							continue
						} else {
							ip := ip.IP.To4()

							if ip != nil && ip.String() == common.LocalIP {
								macString = iface.HardwareAddr.String()
								return macString
							}

						}

					}
				}
			}

		}
	}

	return ""
}

type PsResult struct {
	ID     string `json:"ID"`
	Image  string `json:"Image"`
	Names  string `json:"Names"`
	Status string `json:"Status"`

	Command      string `json:"Command"`
	CreatedAt    string `json:"CreatedAt"`
	Labels       string `json:"Labels"`
	LocalVolumes string `json:"LocalVolumes"`
	Mounts       string `json:"Mounts"`
	Networks     string `json:"Networks"`
	Ports        string `json:"Ports"`
	RunningFor   string `json:"RunningFor"`
	Size         string `json:"Size"`
	State        string `json:"State"`
}

type StatResult struct {
	ID        string `json:"ID"`
	BlockIO   string `json:"BlockIO"`
	CpuPerc   string `json:"CPUPerc"`
	Container string `json:"Container"`
	MemPerc   string `json:"MemPerc"`
	MemUsage  string `json:"MemUsage"`
	Name      string `json:"Name"`
	NetIO     string `json:"NetIO"`
	Pids      string `json:"PIDs"`
}

type Mount struct {
	Type        string `json:"Type"`
	Name        string `json:"Name"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Driver      string `json:"Driver"`
	Mode        string `json:"Mode"`
	RW          bool   `json:"RW"`
	Propagation string `json:"Propagation"`
}
type NetworkSettings struct {
	Ports map[string]string `json:"Ports"`
}

type InspectResult struct {
	ID              string          `json:"Id"`
	Image           string          `json:"Image"`
	Mounts          []Mount         `json:"Mounts"`
	NetworkSettings NetworkSettings `json:"NetworkSettings"`
}

// GetDocker docker info
func GetDocker() (resultData []map[string]string) {
	var cmdexecStrings string
	//cmdexecStrings := common.Cmdexec("docker ps -a --format json")
	c := exec.Command("sh", "-c", "docker ps -a --format '{{json .}}'")
	out, _ := c.CombinedOutput()
	cmdexecStrings = string(out)
	//cmdexecStrings := common.Cmdexec("docker ps -a --format '{{json .}}'")
	cmdexecString := strings.Split(cmdexecStrings, "\n")
	//log.Infof("docker info: %s, len: %d \n", cmdexecStrings, len(cmdexecStrings))
	//log.Printf("docker info: %s, len: %d \n", cmdexecStrings, len(cmdexecStrings))
	if len(cmdexecString) == 0 {
		return
	}

	var statsStrings string
	//statsStrings := common.Cmdexec("docker stats -a --no-stream --format json")
	s := exec.Command("sh", "-c", "docker stats -a --no-stream --format '{{json .}}'")
	outs, _ := s.CombinedOutput()
	statsStrings = string(outs)
	//statsStrings := common.Cmdexec("docker stats -a --no-stream --format '{{json .}}'")
	statsString := strings.Split(statsStrings, "\n")

	var cpuPerc = make(map[string]StatResult, len(cmdexecString))
	for _, stats := range statsString {
		var cpu StatResult
		if err := json.Unmarshal([]byte(stats), &cpu); err != nil {
			continue
		}
		cpuPerc[cpu.ID] = cpu
	}
	//log.Infof("cpu stat: %v\n", cpuPerc)
	//log.Printf("cpu stat: %v\n", cpuPerc)

	for _, cmdString := range cmdexecString {
		var psResult PsResult
		if err := json.Unmarshal([]byte(cmdString), &psResult); err != nil {
			//log.Infof("json unmashall string: %s, error, err: %v\n", cmdString, err)
			//log.Printf("json unmashall string: %s, error, err: %v\n", cmdString, err)
			continue
		} else {
			cid := psResult.ID
			inspectCommand := fmt.Sprintf("docker inspect %s", cid)
			inspectString := common.Cmdexec(inspectCommand)
			var inspectResults []InspectResult
			if err := json.Unmarshal([]byte(inspectString), &inspectResults); err != nil {
				//log.Warnf("invalid unmarshal body, err: %+v", err)
				//log.Printf("invalid unmarshal body, err: %+v", err)
				continue
			} else {
				if len := len(inspectResults); len == 0 {
					//log.Infof("inspect result is empty, cid: %s, inspectString: %s\n", cid, inspectString)
					//log.Printf("inspect result is empty, cid: %s, inspectString: %s\n", cid, inspectString)
					continue
				} else {
					var inspectResult = inspectResults[0]

					t := map[string]string{
						"ip":  common.LocalIP,
						"mac": MacString(),

						"containerID":   inspectResult.ID,
						"containerName": psResult.Names, // containerNames
						"imageId":       inspectResult.Image,
						"imageName":     psResult.Image, // ImageNames

						// mounts
						"mounts": mountString(inspectResult.Mounts),
						// ports
						"ports": portString(inspectResult.NetworkSettings.Ports),

						"status": ontShot(psResult.Status),
					}
					if _, ok := cpuPerc[psResult.ID]; ok {
						t["cpuPerc"] = trimChar(cpuPerc[psResult.ID].CpuPerc, "%")
						t["memPerc"] = trimChar(cpuPerc[psResult.ID].MemPerc, "%")
					} else {
						t["cpuPerc"] = "0.00"
						t["memPerc"] = "0.00"
					}
					resultData = append(resultData, t)
				}

			}

		}
	}
	//log.Infof("docker collected result: %+v", resultData)
	log.Printf("docker collected result: %+v", resultData)
	return resultData
}

func trimChar(str, chars string) string {
	return strings.Replace(str, chars, "", -1)
}

func mountString(mounts []Mount) string {
	if len := len(mounts); len == 0 {
		return ""
	}
	var mstring = make([]string, 0, len(mounts))
	for _, mount := range mounts {
		mstring = append(mstring, fmt.Sprintf("%s:%s", mount.Source, mount.Destination))
	}
	return strings.Join(mstring, ",")

}

func portString(ports map[string]string) string {
	if len := len(ports); len == 0 {
		return ""
	}

	var pstring = make([]string, 0, len(ports))
	for key, value := range ports {
		pstring = append(pstring, fmt.Sprintf("%s:%s", key, value))
	}

	return strings.Join(pstring, ",")

}

func ontShot(shot string) string {
	if strings.HasPrefix(shot, "Up") || strings.HasPrefix(shot, "up") {
		return "1"
	} else {
		return "0"
	}

}
