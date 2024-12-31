package models

import (
	"fmt"
	"github.com/astaxie/beego"
	"gopkg.in/mgo.v2/bson"
	"strings"
	"time"
	"yulong-hids/web/models/wmongo"
		mgo "gopkg.in/mgo.v2"
	"yulong-hids/web/utils"
)

type User struct {
	ID         bson.ObjectId `bson:"_id,omitempty"      json:"_id,omitempty"`
	Name       string        `bson:"name"       json:"name"`
	Password   string        `bson:"password"       json:"password"`
	NickName   string        `bson:"nickname"       json:"nickname"`
	Remark     string        `bson:"remark"       json:"remark"`
		Status     int        `bson:"status"       json:"status"`
	baseModel  `bson:",inline"`
	Uptime     time.Time `bson:"uptime"   json:"uptime,omitempty"`
	ModifyTime time.Time `bson:"modifytime"   json:"modifytime,omitempty"`
}

func NewUser() User {
	mdl := User{}
	mdl.collectionName = "user"
	return mdl
}

func (c *User) Update() bool {
	mConn := wmongo.Conn()
	defer mConn.Close()

	collections := mConn.DB("").C("user")

	selector := bson.M{"name": c.Name}
	err := collections.Update(selector, &c)

	if err == mgo.ErrNotFound {
		err = collections.Insert(c)
	} else if err != nil {
		beego.Error("User Insert Error:", err)
		return false
	}

	return true
}

func (c *User) Save() bool {
	mConn := wmongo.Conn()
	defer mConn.Close()
	id := bson.NewObjectId()
	collections := mConn.DB("").C(c.collectionName)
	c.ID = id
	c.Uptime = time.Now()
	c.ModifyTime = time.Now()
	if err := collections.Insert(&c); err != nil {
		beego.Error("Task Insert Error", err)
		return false
	}
	return true
}

func (c *User) EditByID(id string, key string, value string) bool {

	if strings.Trim(value, " ") == "" {
		return false
	}

	mConn := wmongo.Conn()
	defer mConn.Close()
	mid := bson.ObjectIdHex(id)

	vresult := utils.KeyType(key, value)

	data := bson.M{"$set": bson.M{fmt.Sprintf("%s", key): vresult,"modifyTime":time.Now()}}
	err := mConn.DB("").C(c.collectionName).UpdateId(mid, data)
	if err == nil {
		return true
	}
	beego.Error("User ChangeById(model.UpdateId) Error", err)

	return false
}

func (c *User) DelOne(id string) bool {
	return c.Remove(bson.M{"_id":id}) == nil
}


