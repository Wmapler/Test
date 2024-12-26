package safecheck

import (
	"fmt"
	"gopkg.in/mgo.v2/bson"
	"log"
	"strconv"
	"time"
	"yulong-hids/server/models"
)

func Predict() {
	log.Println("start Predict thread at: ", time.Now().String())

	ticker := time.NewTicker(time.Minute * 10)
	defer ticker.Stop()

	for {

		for {
			select {
			case t := <-ticker.C:
				fmt.Println("predict ticker start, current time: ", t)
				go predictRunning()
			}
		}

	}
}

func predictRunning() {
	infoDB := models.DB.C("info")

	var ips []string
	if err := infoDB.Find(bson.M{"type": "processlist"}).Distinct("ip", &ips); err != nil {
		log.Println("error in predict finding distinct type: processlist", err)
		return
	}

	type dataInfo struct {
		Data   []map[string]string
		Uptime time.Time
		IP     string
		Type   string
	}

	for _, ip := range ips {
		fmt.Println("predict ip: ", ip)
		var collectedData = make([]models.UsageData, 0)
		var dockerData dataInfo
		if err := infoDB.Find(bson.M{"type": "docker", "ip": ip}).One(&dockerData); err != nil {
			log.Println("error in predict finding docker data, type: docker ", err)
		} else {
			for _, v := range dockerData.Data {
				name := v["containerName"]
				memPerc, _ := strconv.ParseFloat(v["memPerc"], 64)
				cpuPerc, _ := strconv.ParseFloat(v["cpuPerc"], 64)
				collectedData = append(collectedData, models.UsageData{
					CPU:  cpuPerc,
					Mem:  memPerc,
					Name: name,
				})
			}
		}
		var processData dataInfo
		if err := infoDB.Find(bson.M{"type": "processlist", "ip": ip}).One(&processData); err != nil {
			log.Println("error in predict find processlist data, type: processlist", err)
		} else {
			for _, v := range processData.Data {
				name := v["name"]
				memPerc, _ := strconv.ParseFloat(v["memPerc"], 64)
				cpuPerc, _ := strconv.ParseFloat(v["cpuPerc"], 64)
				collectedData = append(collectedData, models.UsageData{
					CPU:  cpuPerc,
					Mem:  memPerc,
					Name: name,
				})
			}
		}
		body := models.Predict(collectedData, 2)
		models.DB.C("predict").Upsert(bson.M{"ip": ip, "type": "predict"}, bson.M{"$set": bson.M{"body": body, "uptime": time.Now()}})

	}

}
