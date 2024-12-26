package safecheck

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
	"yulong-hids/server/models"
)

type score struct {
	CHistoryRule *mgo.Collection // 规则命中存储地址
	CScore       *mgo.Collection // 打分存储地址
	CInfo        *mgo.Collection // docker 数据存储地址
}

type CalcScore struct {
	Type   string
	Time   time.Time
	IP     string
	System string

	Score int64
}

func Score() {
	log.Println("start score thread at: ", time.Now().String())

	ticker := time.NewTicker(time.Minute * 60)
	defer ticker.Stop()

	for {

		for {
			select {
			case t := <-ticker.C:
				fmt.Println("ticker start, current time: ", t)
				go scoreRun()
			}
		}

	}

}

func scoreRun() {
	ScoreRunning(models.DB)
}

func ScoreRunning(database *mgo.Database) {
	s := new(score)
	s.CHistoryRule = database.C("history_rule")
	s.CScore = database.C("score")
	s.CInfo = database.C("info")

	// 获取docker镜像统计信息
	var ips []string
	if err := s.CInfo.Find(bson.M{"type": "docker"}).Distinct("ip", &ips); err != nil {
		log.Println("score running for distinct ip for docker")
		return
	}

	type dataInfo struct {
		Data   []map[string]string
		Uptime time.Time
		IP     string
		Type   string
	}

	var PercentDefine = []float64{60.0, 40.0}
	for _, ip := range ips {

		log.Println("score ip: ", ip)

		// current ip  docker stat
		var score = 0.0
		var d *dataInfo
		var aliveCount, totalCount = 0, 0
		s.CInfo.Find(bson.M{"type": "docker", "ip": ip}).One(d)
		log.Printf("get docker info： %+v\n", d)
		if d == nil {
			score += PercentDefine[0]
		} else {

			for _, data := range d.Data {
				totalCount++
				if data["status"] == "1" {
					aliveCount++
				}
			}
			score += PercentDefine[0] * float64(aliveCount) / float64(totalCount)
		}

		// current
		var historyRule []models.HistoryRule
		s.CHistoryRule.Find(bson.M{"ip": ip, "system": "linux"}).All(&historyRule)
		log.Printf("get history rule %+v\n", historyRule)
		if len(historyRule) == 0 {
			score += PercentDefine[1]
		} else {
			var ratio float64 = 0.0
			for _, rule := range historyRule {
				ratio += float64(rule.Total-rule.Hit) / float64(rule.Total)
			}
			score += PercentDefine[1] * ratio / float64(len(historyRule))
		}

		log.Printf("calc score for ip: %s, alive: %d, total: %d, rule len: %d,  score: %d\n", ip, aliveCount, totalCount, len(historyRule), int64(score))

		s.CScore.Upsert(bson.M{"type": "score", "ip": ip, "system": "linux"}, bson.M{"$set": bson.M{"score": int64(score), "uptime": time.Now()}})

		// 需要清理超过60分钟的历史规则统计数据
		//count, _ := s.CHistoryRule.Find(bson.M{
		//	"ip":     ip,
		//	"system": "linux",
		//	"uptime": bson.M{"$lte": time.Now().Add(time.Minute * time.Duration(-60))},
		//}).Count()
		//if count > 0 {
		//	log.Println("to deleted query history rule count: ", count)
		//	if err := s.CHistoryRule.Remove(bson.M{
		//		"ip":     ip,
		//		"system": "linux",
		//		"uptime": bson.M{"$lte": time.Now().Add(time.Minute * time.Duration(-60))},
		//	}); err != nil {
		//		log.Println("history rule remove for ip: ", ip, err.Error())
		//		continue
		//	}
		//}

	}

}
