package wmongo

import (
	"github.com/astaxie/beego"
	"gopkg.in/mgo.v2"
)

var session *mgo.Session

// Conn return mongodb session.
func Conn() *mgo.Session {
	return session.Copy()
}

/*
func Close() {
	session.Close()
}
*/

func init() {
	url := beego.AppConfig.String("mongodb::url")
	if len(url) == 0 {
		url = "192.168.49.132:27017"
	}

	sess, err := mgo.Dial(url)
	if err != nil {
		beego.Error("Mongodb url:", url)
		beego.Error("Mongodb session connect", err)
	}

	session = sess
	session.SetMode(mgo.Monotonic, true)
}
