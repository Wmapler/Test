package models

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"

	"github.com/muesli/clusters"
	"github.com/muesli/kmeans"
)

// 定义数据结构
type UsageData struct {
	Name string
	CPU  float64
	Mem  float64
}

// 将 UsageData 转换为 KMeans 库所需的 Observation
func toObservations(data []UsageData) clusters.Observations {
	obs := make(clusters.Observations, len(data))
	for i, d := range data {
		obs[i] = clusters.Coordinates{d.CPU, d.Mem}
	}
	return obs
}

// 使用 KMeans 聚类并检测异常值
func Predict(data []UsageData, numClusters int) string {
	// 将数据转换为 Observations 格式
	observations := toObservations(data)

	// 创建 KMeans 聚类器
	km := kmeans.New()

	// 聚类，获取簇分配结果
	clusterResults, err := km.Partition(observations, numClusters)
	if err != nil {
		log.Fatal(err)
	}

	var body string
	// 打印聚类结果
	for i, c := range clusterResults {
		body += fmt.Sprintf("Cluster %d: 中心点 (CPU=%.2f%%, 内存=%.2f%%)\n", i, c.Center[0], c.Center[1])

		// 对每个簇内的数据点进行异常检测
		for j, o := range c.Observations {
			// 获取数据点的 CPU 和内存使用率
			coordinates := o.Coordinates()
			cpuUsage := coordinates[0] // 通过方法获取 CPU 和 Mem 值
			memUsage := coordinates[1]

			// 获取对应的进程名称
			processName := data[j].Name // 从原始数据中获取进程名称

			// 计算该数据点到簇中心的距离
			cpuDistance := cpuUsage - c.Center[0]
			memDistance := memUsage - c.Center[1]
			distance := math.Sqrt(cpuDistance*cpuDistance + memDistance*memDistance)

			// 设定一个简单的阈值，超过一定距离的点可以被视为异常
			threshold := 3.0 // 这个阈值可以根据数据情况调整
			if distance > threshold {
				body += fmt.Sprintf("异常检测: 进程/容器 %s (CPU=%.2f%%, 内存=%.2f%%), 距离簇中心=%.2f, 状态=异常\n", processName, cpuUsage, memUsage, distance)
			} else {
				body += fmt.Sprintf("正常检测: 进程/容器 %s (CPU=%.2f%%, 内存=%.2f%%), 距离簇中心=%.2f, 状态=正常\n", processName, cpuUsage, memUsage, distance)
			}
		}
	}
	return body
}

// 从 CSV 文件读取数据
func readCSVAndDetect(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	_, _ = reader.Read() // 跳过头部

	var data []UsageData

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		cpu, _ := strconv.ParseFloat(record[1], 64)
		mem, _ := strconv.ParseFloat(record[2], 64)
		data = append(data, UsageData{Name: record[0], CPU: cpu, Mem: mem})
	}

	// 使用KMeans检测异常
	Predict(data, 2) // 将进程分为2个簇
}

func main1() {
	// 读取 CSV 文件并检测异常
	readCSVAndDetect("usage_stats.csv")
}
