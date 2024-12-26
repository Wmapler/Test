package controllers

import (
	"yulong-hids/web/models"
	"github.com/astaxie/beego"
	"gopkg.in/mgo.v2/bson"
)

// LoginController /login
type LoginController struct {
	beego.Controller
}

// Get the first page in this project
func (c *LoginController) Get() {
	c.Ctx.Output.Header("is-login-page", "true")
	c.Data["Style"] = "login-style"
	c.TplName = "login.tpl"
}

// Post HTTP method POST
func (c *LoginController) Post() {
	json := map[string]bool{"status": false}
	username := c.GetString("username")
	passwd := c.GetString("password")
	beego.Info(username)
	beego.Info(passwd)
	var j = models.NewUser()
	if (j.FindOne(bson.M{"name": username, "password": passwd,"status":0}) != nil) {
		c.Ctx.SetCookie("user", username)
		beego.Warn("User Login :", username)
		json["status"] = true
	}
	c.Data["json"] = json
	c.ServeJSON()
	return
}
