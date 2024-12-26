package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
)

func main() {
	//operation()
	op()
}

type Stat struct {
	Left  int
	Total int
}

type User struct {
	Id        bson.ObjectId `bson:"_id,omitempty"`
	Name      string
	PassWord  string
	Age       int
	Interests []string
	Stat      []Stat
}

func op() {
	db, err := mgo.Dial("127.0.0.1:27017")
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()
	db.SetMode(mgo.Monotonic, true)
	database := db.DB("medical_industry_medical")
	c := database.C("medical_device_register_pro")
	for i := 0; i < 100; i++ {
		c.Insert(&User{
			Name: "w",
			Age:  i + 10,
		})
	}

	var users []User
	if err := c.Find(nil).Sort("-age").Limit(10).All(&users); err != nil {
		return
	} else {
		log.Println(users)
	}

	c.Update(bson.M{"age": 100}, bson.M{"$push": bson.M{"interests": "PHP", "stat": bson.M{"left": 5, "total": 10}}})
	c.Update(bson.M{"age": 100}, bson.M{"$push": bson.M{"interests": "GO", "stat": bson.M{"left": 50, "total": 70}}})
	c.Update(bson.M{"age": 100}, bson.M{"$push": bson.M{"stat": bson.M{"left": 500, "total": 700}}})
	c.Find(bson.M{"age": 100}).All(&users)
	log.Println(users)

	//c.Update(bson.M{"age": 100}, bson.M{"$pull": bson.M{"interests": "PHP"}})
	//c.Find(bson.M{"age": 100}).All(&users)
	//log.Println("$pull resp: ", users)

	c.Update(bson.M{"age": 100}, bson.M{"$unset": bson.M{"interests": ""}})
	c.Find(bson.M{"age": 100}).All(&users)
	log.Println("$pull resp: ", users)

	c.Update(bson.M{"age": 100}, bson.M{"$pullAll": bson.M{"interests": bson.M{}}})
	c.Find(bson.M{"age": 100}).All(&users)
	log.Println("$pull all resp: ", users)

	if err := c.DropCollection(); err != nil {
		log.Println("drop index medical_device_register_pro error", err)
		return
	}
	if err := database.DropDatabase(); err != nil {
		log.Println("drop database medical_industry_medical error", err)
		return
	}

}

func operation() {
	//
	db, err := mgo.Dial("127.0.0.1:27017")
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()
	db.SetMode(mgo.Monotonic, true)
	database := db.DB("medical_industry_medical")
	c := database.C("medical_device_register_pro")
	//插入
	c.Insert(&User{
		Id:       bson.NewObjectId(),
		Name:     "JK_CHENG",
		PassWord: "123132",
		Age:      2,
		//Interests []string{}
	}, &User{
		Id:       bson.NewObjectId(),
		Name:     "JK_WEI",
		PassWord: "qwer",
		Age:      5,
		//Interests []string{}
	}, &User{
		Id:       bson.NewObjectId(),
		Name:     "JK_HE",
		PassWord: "6666",
		Age:      7,
		//Interests []string{}
	})
	log.Println(c)
	var users []User
	c.Find(bson.M{"Name": bson.M{"$regex": bson.RegEx{"JK", "im"}}}).All(&users) //模糊查询
	log.Println("fuzzy query:", users)

	e := c.Find(nil).All(&users) //查询全部数据
	log.Println("find all", e, " len: ", len(users), users)

	c.FindId(users[0].Id).All(&users) //通过ID查询
	log.Println("iq query: ", users)

	c.Find(bson.M{"name": "JK_WEI"}).All(&users) //单条件查询(=)
	log.Println("name condition query: ", users)

	c.Find(bson.M{"name": bson.M{"$ne": "JK_WEI"}}).All(&users) //单条件查询(!=)
	log.Println("name ne query: ", users)

	c.Find(bson.M{"age": bson.M{"$gt": 5}}).All(&users) //单条件查询(>)
	log.Println("age gt query: ", users)

	c.Find(bson.M{"age": bson.M{"$gte": 5}}).All(&users) //单条件查询(>=)
	log.Println("age get query: ", users)

	c.Find(bson.M{"age": bson.M{"$lt": 5}}).All(&users) //单条件查询(<)
	log.Println("age lt query: ", users)

	c.Find(bson.M{"age": bson.M{"$lte": 5}}).All(&users) //单条件查询(<=)
	log.Println("age lte query: ", users)

	c.Find(bson.M{"name": bson.M{"$in": []string{"JK_WEI", "JK_HE"}}}).All(&users) //单条件查询(in)
	log.Println("name in query: ", users)

	c.Find(bson.M{"$or": []bson.M{bson.M{"name": "JK_WEI"}, bson.M{"age": 7}}}).All(&users) //多条件查询(or)
	log.Println("name or age multi query: ", users)

	c.Update(bson.M{"_id": users[0].Id}, bson.M{"$set": bson.M{"name": "JK_HOWIE", "age": 61}}) //修改字段的值($set)
	c.FindId(users[0].Id).All(&users)
	log.Println("update query: ", users)

	c.Find(bson.M{"name": "JK_CHENG", "age": 2}).All(&users) //多条件查询(and)
	log.Println("name and age query: ", users)

	c.Update(bson.M{"_id": users[0].Id}, bson.M{"$inc": bson.M{"age": -6}}) //字段增加值($inc)
	c.FindId(users[0].Id).All(&users)
	log.Println("age min value query: ", users)

	c.Update(bson.M{"_id": users[0].Id}, bson.M{"$push": bson.M{"interests": "PHP"}}) //从数组中增加一个元素($push)
	c.FindId(users[0].Id).All(&users)
	log.Println("push element query: ", users)

	c.Update(bson.M{"_id": users[0].Id}, bson.M{"$pull": bson.M{"interests": "PHP"}}) //从数组中删除一个元素($pull)
	c.FindId(users[0].Id).All(&users)
	log.Println("pull element query: ", users)

	c.Remove(bson.M{"name": "JK_CHENG"}) //删除
	c.Find(bson.M{"name": "JK_CHENG"}).All(&users)
	log.Println("delete query: ", users)

	if err := c.DropCollection(); err != nil {
		log.Println("drop index medical_device_register_pro error", err)
		return
	}
	if err := database.DropDatabase(); err != nil {
		log.Println("drop database medical_industry_medical error", err)
		return
	}
}
