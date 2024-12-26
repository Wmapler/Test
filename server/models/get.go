package models

import (
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// 获取进程信息
func getProcessStats() (map[string]map[string]float64, error) {
	cmd := exec.Command("ps", "-eo", "pid,%cpu,%mem,comm")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]map[string]float64)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		cpu, _ := strconv.ParseFloat(fields[1], 64)
		mem, _ := strconv.ParseFloat(fields[2], 64)
		name := fields[3]

		stats[name] = map[string]float64{
			"cpu": cpu,
			"mem": mem,
		}
	}

	return stats, nil
}

// 获取 Docker 容器信息
func getDockerStats() (map[string]map[string]float64, error) {
	cmd := exec.Command("docker", "stats", "--no-stream", "--format", "{{.Name}} {{.CPUPerc}} {{.MemUsage}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]map[string]float64)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		name := fields[0]
		cpu, _ := strconv.ParseFloat(strings.TrimSuffix(fields[1], "%"), 64)
		memInfo := strings.Split(fields[2], "/")
		memUsage, _ := strconv.ParseFloat(strings.TrimSpace(memInfo[0]), 64)

		stats[name] = map[string]float64{
			"cpu": cpu,
			"mem": memUsage,
		}
	}

	return stats, nil
}

// 保存数据至 CSV 文件
func saveToCSV(stats map[string]map[string]float64, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入头部
	writer.Write([]string{"Name", "CPU_Usage", "Memory_Usage"})

	for name, usage := range stats {
		writer.Write([]string{name, fmt.Sprintf("%f", usage["cpu"]), fmt.Sprintf("%f", usage["mem"])})
	}

	return nil
}

func main3() {
	processStats, err := getProcessStats()
	if err != nil {
		fmt.Println("Error getting process stats:", err)
		return
	}

	dockerStats, err := getDockerStats()
	if err != nil {
		fmt.Println("Error getting Docker stats:", err)
		return
	}

	// 合并进程和 Docker 数据
	for k, v := range dockerStats {
		processStats[k] = v
	}

	err = saveToCSV(processStats, "usage_stats.csv")
	if err != nil {
		fmt.Println("Error saving to CSV:", err)
	}
}
