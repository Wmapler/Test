package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego"
	"gopkg.in/mgo.v2/bson"
	"log"
	"net/http"
	"time"
	"yulong-hids/server/models"
	"yulong-hids/web/models/wmongo"
	"yulong-hids/web/utils"
)

// WebApiController web api for agent
type WebApiController struct {
	beego.Controller
}

// Info Get agent will get publickey content and serverlist here
func (c *WebApiController) Info() {

	conn := wmongo.Conn()
	defer conn.Close()

	type DockerData struct {
		ContainerId string `json:"containerId"`
		ImageId     string `json:"imageId"`
		ImageName   string `json:"imageName"`
		Mounts      string `json:"mounts"`
		Ports       string `json:"ports"`
		Status      string `json:"status"`
	}
	type LoginData struct {
		Status   string `json:"status"`
		Remote   string `json:"remote"`
		Username string `json:"username"`
		Time     string `json:"time"`
	}
	type FileData struct {
		Path string `json:"path"`
		Hash string `json:"hash"`
		User string `json:"user"`
	}
	type ConnectionData struct {
		Dir      string `json:"dir"`
		Protocol string `json:"protocol"`
		Remote   string `json:"remote"`
		Local    string `json:"local"`
		Pid      string `json:"pid"`
		Name     string `json:"name"`
	}
	type OutputInfo struct {
		Ip  string `json:"ip"`
		Mac string `json:"mac"`

		Docker     []DockerData  `json:"docker"`
		Login      []interface{} `json:"login"`
		File       []interface{} `json:"file"`
		Connection []interface{} `json:"connection"`
	}

	var docker []models.DataInfo
	if err := conn.DB("agent").C("info").Find(bson.M{"type": "docker", "system": "linux"}).All(&docker); err != nil {
		c.Data["json"] = []map[string]interface{}{}
	} else {
		var results = make([]OutputInfo, 0, len(docker))

		var indexed = []string{
			fmt.Sprintf("monitor%04d_%02d", time.Now().Year(), time.Now().Month()),
			//fmt.Sprintf("monitor%04d_%02d", time.Now().Year(), (time.Now().Month()+11)%12),
		}

		for _, doc := range docker {
			var outOne OutputInfo
			outOne.Ip = doc.IP

			for _, datum := range doc.Data {
				outOne.Mac = datum["mac"]
				outOne.Docker = append(outOne.Docker, DockerData{
					ContainerId: datum["containerID"],
					ImageId:     datum["imageId"],
					ImageName:   datum["imageName"],
					Mounts:      datum["mounts"],
					Ports:       datum["ports"],
					Status:      datum["status"],
				})
			}

			outOne.Login = esData(indexed, "loginlog", doc.IP)
			outOne.File = esData(indexed, "file", doc.IP)
			outOne.Connection = esData(indexed, "connection", doc.IP)

			results = append(results, outOne)
		}

		c.Data["json"] = results
	}

	c.ServeJSON()
	return

}

func esData(indexes []string, mode, ip string) []interface{} {
	termquery := bson.M{"term": bson.M{"ip": ip}}
	var query = bson.M{"query": bson.M{"bool": bson.M{"must": []bson.M{termquery}}}}
	query["from"] = 0
	query["size"] = 10
	query["sort"] = []bson.M{bson.M{"time": bson.M{"order": "desc"}}}

	// request es web api
	var result interface{}
	es := utils.NewSession()
	qbytes, _ := json.Marshal(query)
	result = es.NewSearch(indexes, mode, qbytes)
	if result == nil {
		return []interface{}{}
	}

	result = result.(bson.M)["hits"]
	result = result.(map[string]interface{})["hits"]

	itemlist := result.([]interface{})
	var datalist []interface{}
	for _, item := range itemlist {
		var data interface{}
		data = item.(map[string]interface{})["_source"]
		datalist = append(datalist, data)
	}

	return datalist
}

func (c *WebApiController) Abnormal() {

	conn := wmongo.Conn()
	defer conn.Close()

	type result struct {
		Ip          string              `json:"ip"`
		Message     string              `json:"message"`
		Time        string              `json:"time"`
		RuleMessage []map[string]string `json:"ruleMessage"`
	}
	var results = make([]result, 0)

	type abnormal struct {
		Ip     string    `json:"ip"`
		Body   string    `json:"body"`
		Uptime time.Time `json:"uptime"`
	}

	type rule struct {
		Ip          string    `json:"ip"`
		Type        string    `json:"type"`
		Info        string    `json:"info"`
		Description string    `json:"description"`
		Time        time.Time `json:"time"`
	}
	var abnormals []abnormal
	if err := conn.DB("agent").C("predict").Find(bson.M{"type": "predict"}).All(&abnormals); err == nil {
		for _, ab := range abnormals {
			var rst = result{
				Ip:      ab.Ip,
				Message: ab.Body,
				Time:    ab.Uptime.String(),
			}

			var rules []rule
			if err := conn.DB("agent").C("notice").Find(bson.M{"ip": ab.Ip}).All(&rules); err == nil {
				rst.RuleMessage = make([]map[string]string, 0)
				for _, r := range rules {
					rst.RuleMessage = append(rst.RuleMessage, map[string]string{
						"type":        r.Type,
						"info":        r.Info,
						"description": r.Description,
						"time":        r.Time.String(),
					})
				}
			} else {
				log.Println("Warn", "agent notice find error, ip: ", ab.Ip)
			}

			results = append(results, rst)
		}
	} else {
		log.Println("Warn", "agent predict find error")
	}

	c.Data["json"] = results
	c.ServeJSON()
	return

}

// Score get agent score value
func (c *WebApiController) Score() {

	type score struct {
		Score int    `json:"score"`
		Ip    string `json:"ip"`
	}

	var scores []score
	conn := wmongo.Conn()
	defer conn.Close()
	if err := conn.DB("agent").C("score").Find(bson.M{"type": "score", "system": "linux"}).All(&scores); err != nil {
		c.Ctx.Output.SetStatus(503)
	} else {
		c.Data["json"] = scores
	}
	c.ServeJSON()
	return
}

func (c *WebApiController) Auth() {
	type score struct {
		Score    int    `json:"score"`
		Ip       string `json:"ip"`
		AuthType int    `json:"authType"`
	}

	var scores []score
	conn := wmongo.Conn()
	defer conn.Close()
	if err := conn.DB("agent").C("score").Find(bson.M{"type": "score", "system": "linux"}).All(&scores); err != nil {
		c.Ctx.Output.SetStatus(503)
	} else {
		for i := range scores {
			if scores[i].Score >= 90 {
				scores[i].AuthType = 0
			} else {
				scores[i].AuthType = 1
			}
		}
		log.Printf("score: %+v\n", scores)

		c.Data["json"] = scores
	}
	c.ServeJSON()
	return
}

func (c *WebApiController) ActiveAuth() {
	ip := c.GetString("ip")
	url := fmt.Sprintf("http://%s:10001/require_auth", ip)
	get, _ := http.NewRequest("GET", url, nil)
	cli := http.Client{}
	if getResponse, err := cli.Do(get); err != nil {
		log.Println("require auth handler error: ", err)
		c.Data["json"] = false
	} else {
		defer getResponse.Body.Close()
		if getResponse.StatusCode != 200 {
			log.Println("auth status code error, code:", getResponse.StatusCode)
			c.Data["json"] = false
		} else {
			log.Println("auth success, status", getResponse.StatusCode)
			c.Data["json"] = true
		}
	}

	c.ServeJSON()
	return

}

func (c *WebApiController) CutOff() {
	ip := c.GetString("ip")
	conn := wmongo.Conn()
	defer conn.Close()

	if cnt, err := conn.DB("agent").C("authing").Find(bson.M{"ip": ip}).Count(); err != nil {
		log.Println("cutoff find ip: ", ip, " count error, error: ", err)
		return
	} else {
		if cnt == 0 {
			log.Println("cutoff find ip: ", ip, " count == 0 and ignore this cut off op")
			c.Data["json"] = map[string]bool{
				"data": false,
			}
			return
		} else {
			if _, err := conn.DB("agent").C("authing").Upsert(bson.M{"ip": ip}, bson.M{"$set": bson.M{"cut": 1}}); err != nil {
				c.Data["json"] = map[string]bool{
					"data": false,
				}
			} else {
				c.Data["json"] = map[string]bool{
					"data": true,
				}
			}
			c.ServeJSON()
			return

		}
	}

}

func (c *WebApiController) Recover() {
	ip := c.GetString("ip")
	conn := wmongo.Conn()
	defer conn.Close()

	if cnt, err := conn.DB("agent").C("authing").Find(bson.M{"ip": ip}).Count(); err != nil {
		log.Println("recover find ip: ", ip, " count error, error: ", err)
		return
	} else {
		if cnt == 0 {
			log.Println("recover find ip: ", ip, " count == 0 and ignore this recover op")
			c.Data["json"] = map[string]bool{
				"data": false,
			}
			return
		} else {
			if _, err := conn.DB("agent").C("authing").Upsert(bson.M{"ip": ip}, bson.M{"$set": bson.M{"cut": 0}}); err != nil {
				c.Data["json"] = map[string]bool{
					"data": false,
				}
			} else {
				c.Data["json"] = map[string]bool{
					"data": true,
				}
			}
			c.ServeJSON()
			return

		}
	}

}

func (c *WebApiController) ClientStatus() {

	type status struct {
		Ip   string    `bson:"ip"`
		Time time.Time `bson:"uptime"`
		Cut  int       `bson:"cut"`
	}

	var sts []status

	conn := wmongo.Conn()
	defer conn.Close()
	if err := conn.DB("agent").C("authing").Find(bson.M{}).All(&sts); err != nil {
		c.Ctx.Output.SetStatus(503)
		return
	} else {

		type r struct {
			Ip     string    `json:"ip"`
			Status int       `json:"status"`
			Uptime time.Time `json:"uptime"`
			Now    time.Time `json:"now"`
		}

		var rst []r = make([]r, 0)
		for _, s := range sts {
			var status = 0
			if s.Cut == 1 {
				status = 1
			} else {
				if time.Now().Unix()-s.Time.Unix() > 150 {
					status = 2
				}
			}
			rst = append(rst, r{
				Ip:     s.Ip,
				Status: status,
				Uptime: s.Time,
			})
		}

		c.Data["json"] = rst
	}
	c.ServeJSON()
	return

}
