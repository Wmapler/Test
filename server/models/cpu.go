package models

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type ResourceInfo struct {
	Type     string  // "process" 或 "container"
	ID       string  // PID 或容器 ID
	Name     string  // 进程名或容器名
	CPUUsage float64 // CPU 使用率
	MemUsage float64 // 内存使用率
	Label    int     // 标记，0：正常，1：异常
}

// 获取系统中所有进程的 CPU 和内存使用情况
func getProcessInfo() ([]ResourceInfo, error) {
	cmd := exec.Command("ps", "-eo", "pid,comm,%cpu,%mem")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out.String(), "\n")
	var processList []ResourceInfo
	for i, line := range lines {
		if i == 0 || line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		pid := fields[0]
		name := fields[1]
		cpuUsage, _ := strconv.ParseFloat(fields[2], 64)
		memUsage, _ := strconv.ParseFloat(fields[3], 64)

		label := 0
		if cpuUsage > 80.0 || memUsage > 80.0 {
			label = 1 // 标记为异常
		}

		processList = append(processList, ResourceInfo{
			Type:     "process",
			ID:       pid,
			Name:     name,
			CPUUsage: cpuUsage,
			MemUsage: memUsage,
			Label:    label,
		})
	}
	return processList, nil
}

// 获取所有 Docker 容器的 CPU 和内存使用情况
func getContainerInfo() ([]ResourceInfo, error) {
	cmd := exec.Command("docker", "stats", "--no-stream", "--format", "{{.ID}} {{.Name}} {{.CPUPerc}} {{.MemUsage}}")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out.String(), "\n")
	var containerList []ResourceInfo
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		id := fields[0]
		name := fields[1]
		cpuUsageStr := strings.TrimSuffix(fields[2], "%")
		cpuUsage, _ := strconv.ParseFloat(cpuUsageStr, 64)

		memUsageStr := fields[3]
		memUsageParts := strings.Split(memUsageStr, "/")
		if len(memUsageParts) < 1 {
			continue
		}
		memUsageValue := strings.Trim(memUsageParts[0], "MiB")
		memUsage, _ := strconv.ParseFloat(memUsageValue, 64)

		label := 0
		if cpuUsage > 80.0 || memUsage > 80.0 {
			label = 1 // 标记为异常
		}

		containerList = append(containerList, ResourceInfo{
			Type:     "container",
			ID:       id,
			Name:     name,
			CPUUsage: cpuUsage,
			MemUsage: memUsage,
			Label:    label,
		})
	}
	return containerList, nil
}

// 收集指定时间内的资源使用数据
func collectData(interval time.Duration, duration time.Duration) ([]ResourceInfo, error) {
	var allData []ResourceInfo
	endTime := time.Now().Add(duration)
	for time.Now().Before(endTime) {
		processData, err := getProcessInfo()
		if err != nil {
			log.Println("获取进程信息出错:", err)
		}
		containerData, err := getContainerInfo()
		if err != nil {
			log.Println("获取容器信息出错:", err)
		}
		allData = append(allData, processData...)
		allData = append(allData, containerData...)

		time.Sleep(interval)
	}
	return allData, nil
}

// 将数据保存到 CSV 文件
func saveDataToCSV(data []ResourceInfo, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入 CSV 头部
	writer.Write([]string{"Type", "ID", "Name", "CPUUsage", "MemUsage", "Label"})

	for _, info := range data {
		writer.Write([]string{
			info.Type,
			info.ID,
			info.Name,
			fmt.Sprintf("%.2f", info.CPUUsage),
			fmt.Sprintf("%.2f", info.MemUsage),
			strconv.Itoa(info.Label),
		})
	}
	return nil
}

func main2() {
	interval := 10 * time.Second // 采样间隔
	duration := 1 * time.Minute  // 采样持续时间

	log.Println("开始收集数据...")
	data, err := collectData(interval, duration)
	if err != nil {
		log.Fatal("数据收集出错:", err)
	}

	filename := "resource_usage_data.csv"
	err = saveDataToCSV(data, filename)
	if err != nil {
		log.Fatal("保存数据到 CSV 文件出错:", err)
	}

	log.Printf("数据收集完成，已保存到 %s", filename)
}
