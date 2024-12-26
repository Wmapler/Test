package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
	"yulong-hids/agent/collect"
	"yulong-hids/agent/common"
	"yulong-hids/agent/models"
	"yulong-hids/agent/monitor"

	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/share"
)

var err error

type dataInfo struct {
	IP     string              // 客户端的IP地址
	Type   string              // 传输的数据类型
	System string              // 操作系统
	Data   []map[string]string // 数据内容
}

// Agent agent客户端结构
type Agent struct {
	ServerNetLoc string         // 服务端地址 IP:PORT
	Client       client.XClient // RPC 客户端
	ServerList   []string       // 存活服务端集群列表
	PutData      dataInfo       // 要传输的数据
	Reply        int            // RPC Server 响应结果
	Mutex        *sync.Mutex    // 安全操作锁
	IsDebug      bool           // 是否开启debug模式，debug模式打印传输内容和报错信息
	ctx          context.Context
}

var httpClient = &http.Client{
	Timeout:   time.Second * 10,
	Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
}

func (a *Agent) init() {
	a.ServerList, err = a.getServerList()
	if err != nil {
		a.log("GetServerList error:", err)
		panic(1)
	}
	a.ctx = context.WithValue(context.Background(), share.ReqMetaDataKey, make(map[string]string))
	a.log("Available server node:", a.ServerList)
	if len(a.ServerList) == 0 {
		time.Sleep(time.Second * 30)
		a.log("No server node available")
		panic(1)
	}
	a.newClient()
	if a.Client == nil {
		a.log("Failed to initialize RPC client")
		panic("Failed to initialize RPC client")
	}
	common.Readonly(func() {
		if common.LocalIP == "" {
			a.log("Can not get local address")
			panic(1)
		}
	})
	//if common.LocalIP == "" {
	//	a.log("Can not get local address")
	//	panic(1)
	//}
	a.Mutex = new(sync.Mutex)
	common.Readonly(func() {
		err := a.Client.Call(a.ctx, "GetInfo", &common.ServerInfo, &common.Config)
		if err != nil {
			a.log("RPC Client Call Error:", err.Error())
			panic(1)
		}
		a.log("Common Client Config:", common.Config)
	})
}

func (a *Agent) register() {
	request := models.RegRequest()
	var response int
	err := a.Client.Call(a.ctx, "Register", request, &response)
	if err != nil {
		common.Readonly(func() {
			a.log("Register error:", err, " ip: ", common.LocalIP)
			panic(1)
		})
	} else {
		a.log("Register success, response type:", response)
		if response == 0 {
			// password auth
			passwordRequest := models.PasswordRequest()
			var passResponse int
			err := a.Client.Call(a.ctx, "Auth", passwordRequest, &passResponse)
			if err != nil {
				a.log("Auth password error:", err)
				panic(1)
			} else {
				a.log("Auth password success:", passResponse)
			}
		} else if response == 1 {
			// zkp auth
			zkpRequest := models.ZkpRequest()
			var zkpResponse int
			err := a.Client.Call(a.ctx, "Auth", zkpRequest, &zkpResponse)
			if err != nil {
				a.log("Auth zkp error:", err)
				panic(1)
			} else {
				a.log("Auth zkp success:", zkpResponse)
			}
		} else {
			a.log("Register error, and invalid response:", response)
			panic(1)
		}
	}
}

// Run 启动agent
func (a *Agent) Run() {
	models.InitAuth()

	// agent 初始化
	// 请求Web API，获取Server地址，初始化RPC客户端，获取客户端IP等
	a.init()

	// 每隔一段时间更新初始化配置
	a.configRefresh()

	// register auth
	a.register()

	// 开启各个监控流程 文件监控，网络监控，进程监控
	a.monitor()

	// 每隔一段时间获取系统信息
	// 监听端口，服务信息，用户信息，开机启动项，计划任务，登录信息，进程列表等
	a.getInfo()
}

func (a *Agent) newClient() {
	var servers []*client.KVPair
	for _, server := range a.ServerList {
		common.Writeonly(func() {
			common.ServerIPList = append(common.ServerIPList, strings.Split(server, ":")[0])
			s := client.KVPair{Key: server}
			servers = append(servers, &s)
			if common.LocalIP == "" {
				a.setLocalIP(server)
				common.ServerInfo = collect.GetComInfo()
				a.log("Host Information:", common.ServerInfo)
			}
		})
	}
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}
	option := client.DefaultOption
	option.TLSConfig = conf
	serverd, _ := client.NewMultipleServersDiscovery(servers)
	a.Client = client.NewXClient("Watcher", FAILMODE, client.RandomSelect, serverd, option)
	a.Client.Auth(AUTH_TOKEN)
}

func (a Agent) getServerList() ([]string, error) {
	var serlist []string
	var url string
	if TESTMODE {
		url = "http://" + a.ServerNetLoc + ":8001" + SERVER_API
	} else {
		url = "http://" + a.ServerNetLoc + ":8001" + SERVER_API
		// url = "https://" + a.ServerNetLoc + ":8001" + SERVER_API
	}
	a.log("Web API:", url)
	request, _ := http.NewRequest("GET", url, nil)
	request.Close = true
	resp, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(result), &serlist)
	if err != nil {
		return nil, err
	}
	return serlist, nil
}

func (a Agent) setLocalIP(ip string) {
	conn, err := net.Dial("tcp", ip)
	if err != nil {
		a.log("Net.Dial:", ip)
		a.log("Error:", err)
		panic(1)
	}
	defer conn.Close()
	//common.Writeonly(func() {
		common.LocalIP = strings.Split(conn.LocalAddr().String(), ":")[0]
	//})
}
func (a *Agent) configRefresh() {
	ticker := time.NewTicker(time.Second * time.Duration(CONFIGR_REF_INTERVAL))
	go func() {
		for _ = range ticker.C {
			ch := make(chan bool)
			go func() {
				a.Mutex.Lock()
				defer a.Mutex.Unlock()
				common.Readonly(func() {
					err = a.Client.Call(a.ctx, "GetInfo", &common.ServerInfo, &common.Config)
					if err != nil {
						a.log("RPC Client Call:", err.Error())
						return
					}
				})
				ch <- true
			}()
			// Server集群列表获取
			select {
			case <-ch:
				serverList, err := a.getServerList()
				if err != nil {
					a.log("RPC Client Call:", err.Error())
					break
				}
				if len(serverList) == 0 {
					a.log("No server node available")
					break
				}
				if len(serverList) == len(a.ServerList) {
					for i, server := range serverList {
						// TODO 可能会产生问题
						if server != a.ServerList[i] {
							a.ServerList = serverList
							// 防止正在传输重置client导致数据丢失
							a.Mutex.Lock()
							a.Client.Close()
							a.newClient()
							a.Mutex.Unlock()
							break
						}
					}
				} else {
					a.log("Server nodes from old to new:", a.ServerList, "->", serverList)
					a.ServerList = serverList
					a.Mutex.Lock()
					a.Client.Close()
					a.newClient()
					a.Mutex.Unlock()
				}
			case <-time.NewTicker(time.Second * 3).C:
				break
			}
		}
	}()
}

func (a *Agent) monitor() {
	resultChan := make(chan map[string]string, 16)

	go monitor.StartNetSniff(resultChan) // connection

	//go monitor.StartProcessMonitor(resultChan) // process
	go monitor.StartFileMonitor(resultChan) // file

	go func(result chan map[string]string) {
		var resultdata []map[string]string
		var data map[string]string
		for {
			data = <-result
			data["time"] = fmt.Sprintf("%d", time.Now().Unix())
			a.log("Monitor data: ", data)
			source := data["source"]
			delete(data, "source")
			resultdata = append(resultdata, data)
			a.Mutex.Lock()
			common.Readonly(func() {
				a.PutData = dataInfo{common.LocalIP, source, runtime.GOOS, resultdata}
				//a.PutData = dataInfo{common.LocalIP, source, runtime.GOOS, append(resultdata, data)}
			})
			a.put()
			a.Mutex.Unlock()
		}
	}(resultChan)
}

func (a *Agent) getInfo() {
	historyCache := make(map[string][]map[string]string)
	for {
		size := 0
		common.Readonly(func() {
			size = len(common.Config.MonitorPath)
			a.log("MonitorPath current value:", common.Config.MonitorPath)
		})
		if len(size) == 0 {
			time.Sleep(time.Second)
			a.log("Failed to get the configuration information")
			continue
		}
		allData := collect.GetAllInfo()
		for k, v := range allData {
			if len(v) == 0 || a.mapComparison(v, historyCache[k]) {
				a.log("GetInfo Data:", k, "No change", " len: ", len(v))
				continue
			} else {
				a.Mutex.Lock()
				common.Readonly(func() {
					a.PutData = dataInfo{common.LocalIP, k, runtime.GOOS, v}
				})
				a.put()
				if k != "service" {
					a.log("Data details:", k, a.PutData)
				}
				a.Mutex.Unlock()
				historyCache[k] = v
			}
		}
		common.Readonly(func() {
			if common.Config.Cycle == 0 {
				common.Config.Cycle = 1
			}
		})
		time.Sleep(time.Second)
		//time.Sleep(time.Second * time.Duration(common.Config.Cycle) * 60)
	}
}

func (a Agent) put() {
	_, err := a.Client.Go(a.ctx, "PutInfo", &a.PutData, &a.Reply, nil)
	if err != nil {
		a.log("PutInfo error:", err.Error())
	}
}

func (a Agent) mapComparison(new []map[string]string, old []map[string]string) bool {
	if len(new) == len(old) {
		for i, v := range new {
			for k, value := range v {
				if value != old[i][k] {
					return false
				}
			}
		}
		return true
	}
	return false
}

func (a Agent) log(info ...interface{}) {
	if a.IsDebug {
		log.Println(info...)
	}
}
