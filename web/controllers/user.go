package controllers

import (
	"encoding/json"
	"yulong-hids/web/models"
	"yulong-hids/web/settings"
		"github.com/astaxie/beego"
	"gopkg.in/mgo.v2/bson"
)

// TaskController /task
type UserController struct {
	BaseController
}

// Get method
func (c *UserController) Get() {

	taskid := c.GetString("uid")
	var res interface{}

	paginator := c.InitPaginator()
	start, limit := paginator.ToParameter()

	if taskid == "" {
		cli := models.NewUser()
		res = cli.GetSortedTop(bson.M{"status":0}, start, limit, "-time")
	} else {
		cli := models.NewTaskResult()
		res = cli.GetSortedTop(bson.M{"task_id": bson.ObjectIdHex(taskid)}, start, limit, "-time")
	}

	c.Data["json"] = res
	c.ServeJSON()
	return
}

// Post method
func (c *UserController) Post() {
	var j = models.NewUser()
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &j); err != nil {
		beego.Error("JSON Unmarshal error:", err)
		c.Data["json"] = models.NewErrorInfo(settings.AddUserFailure)
		c.ServeJSON()
		return
	}
	if res := j.Save(); res {
		json := bson.M{
			"status": 1,
			"msg":    "添加用户成功",
			"Data":   j,
		}
		c.Data["json"] = json
	} else {
		c.Data["json"] = models.NewErrorInfo(settings.AddUserFailure)
	}
	c.ServeJSON()
	return
}


// Edit HTTP method POST
func (c *UserController) Edit() {

	cli := models.NewUser()
	j := models.EditCfgForm{}

	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &j); err != nil {
		beego.Debug("Config edit error:", err)
		c.Data["json"] = models.NewErrorInfo(settings.EditCfgFailure)
		c.ServeJSON()
		return
	}

	res := cli.EditByID(j.Id, j.Key, j.Input)
	c.Data["json"] = bson.M{"status": res}
	c.ServeJSON()
	return
}

// Del HTTP method DELETE
func (c *UserController) Del() {
	beego.Info("del method enter...")
	cli := models.NewUser()
	j := models.EditCfgForm{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &j); err != nil {
		beego.Debug("Config edit error:", err)
		c.Data["json"] = models.NewErrorInfo(settings.EditCfgFailure)
		c.ServeJSON()
		return
	}
	res := cli.DelOne(j.Id)
	c.Data["json"] = bson.M{"status": res}
	c.ServeJSON()
	return
}
