package test

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"testing"
	"time"
	"yulong-hids/server/models"
	"yulong-hids/server/safecheck"
)

func init() {
	log.Println("init...")
}

func initData(database *mgo.Database) {
	log.Println("init data for collection info & history_rule")
	info := database.C("info")
	info.Insert(
		bson.M{
			"ip":     "172.29.70.59",
			"type":   "docker",
			"uptime": time.Now(),
			"data": []map[string]string{
				{
					"containerID": "f95b4336856a",
					"image":       "yulong-hids-ids_web",
					"ip":          "172.29.70.59",
					"mac":         "00:16:3e:06:e7:fb",
					"names":       "ids_web",
					"status":      "0",
				}, {
					"containerID": "73b3e47c3274",
					"image":       "yulong-hids-ids_server",
					"ip":          "172.29.70.59",
					"mac":         "00:16:3e:06:e7:fb",
					"names":       "ids_server",
					"status":      "0"},
				{
					"containerID": "29c31a7434f0",
					"image":       "docker.elastic.co/elasticsearch/elasticsearch:5.5.1",
					"ip":          "172.29.70.59",
					"mac":         "00:16:3e:06:e7:fb",
					"names":       "yulong-hids-ids_elasticsearch-1",
					"status":      "0"},
				{
					"containerID": "1af0249c02a4",
					"image":       "mongo:latest",
					"ip":          "172.29.70.59",
					"mac":         "00:16:3e:06:e7:fb",
					"names":       "ids_mongodb",
					"status":      "1",
				},
			},
		},
	)

	hr := database.C("history_rule")
	hr.Insert(
		models.HistoryRule{
			Type:   "processlist",
			IP:     "172.29.70.59",
			Uptime: time.Now(),
			System: "linux",
			Hit:    1,
			Total:  5,
		},
		models.HistoryRule{
			Type:   "crontab",
			IP:     "172.29.70.59",
			Uptime: time.Now(),
			System: "linux",
			Hit:    3,
			Total:  5,
		},
		models.HistoryRule{
			Type:   "listening",
			IP:     "172.29.70.59",
			Uptime: time.Now(),
			System: "linux",
			Hit:    2,
			Total:  5,
		},
		models.HistoryRule{
			Type:   "listening",
			IP:     "172.29.70.59",
			Uptime: time.Now().Add(time.Duration(-100) * time.Minute),
			System: "linux",
			Hit:    2,
			Total:  5,
		},
	)

}

func TestScore(t *testing.T) {
	log.Println("TestScore start")
	var db *mgo.Session
	var err error
	db, err = mgo.Dial("127.0.0.1:27017")
	if err != nil {
		log.Panic(err)
	}
	db.SetMode(mgo.Monotonic, true)
	database := db.DB("medical_industry_medical")
	initData(database)
	safecheck.ScoreRunning(database)

	if names, err := database.CollectionNames(); err != nil {
		return
	} else {
		for _, name := range names {
			log.Println("drop collection:", name)
			database.C(name).DropCollection()
		}
	}
	database.DropDatabase()

}
