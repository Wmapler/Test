package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"gopkg.in/mgo.v2/bson"
	//"io/ioutil"
	"log"
	"time"
	"yulong-hids/server/action"
	"yulong-hids/server/models"
	"yulong-hids/server/safecheck"
	"crypto/tls"
	"github.com/smallnest/rpcx/protocol"
	"github.com/smallnest/rpcx/server"
)

const authToken string = "67080fc75bb8ee4a168026e5b21bf6fc"

type authing struct {
	Ip     string    `bson:"ip"`
	Value  string    `bson:"value"`
	Uptime time.Time `bson:"Uptime"`
	Cut    int       `bson:"cut"`
}

type Watcher int

// GetInfo agent 提交主机信息获取配置信息
func (w *Watcher) GetInfo(ctx context.Context, info *action.ComputerInfo, result *action.ClientConfig) error {
	action.ComputerInfoSave(*info)
	config := action.GetAgentConfig(info.IP)
	log.Println("getconfig:", info.IP)
	*result = config
	return nil
}

// PutInfo 接收处理agent传输的信息
func (w *Watcher) PutInfo(ctx context.Context, datainfo *models.DataInfo, result *int) error {
	//保证数据正常
	if len(datainfo.Data) == 0 {
		return nil
	}

	// auth check the cut off op
	ip := datainfo.IP
	var a authing
	if err := models.DB.C("authing").Find(bson.M{"ip": ip}).One(&a); err != nil {
		log.Println("err finding authing e:", err, "ip:", ip)
		return err
	}
	if a.Cut == 1 {
		log.Println("PutInfo status cutoff now, ip: ", ip)
		return nil
	} else {
		if err := models.DB.C("authing").Update(bson.M{"ip": ip}, bson.M{"$set": bson.M{"uptime": time.Now()}}); err != nil {
			log.Println("err updating authing Uptime e:", err, "ip:", ip)
			return err
		}
	}

	datainfo.Uptime = time.Now()
	log.Println("putinfo:", datainfo.IP, datainfo.Type)
	err := action.ResultSave(*datainfo)
	if err != nil {
		log.Println(err)
	}
	err = action.ResultStat(*datainfo)
	if err != nil {
		log.Println(err)
	}
	safecheck.ScanChan <- *datainfo
	*result = 1
	return nil
}

func (w *Watcher) Register(ctx context.Context, request *models.RegisterRequest, result *int) error {
	body, _ := json.Marshal(request)
	if _, err := models.DB.C("authing").Upsert(bson.M{"ip": request.Ip}, bson.M{"$set": bson.M{"value": string(body), "uptime": time.Now(), "cut": 0}}); err != nil {
		log.Println("upsert authing error:", err, "ip:", request.Ip)
		return err
	} else {

		type score struct {
			Score int    `json:"score"`
			Ip    string `json:"ip"`
		}

		var s score
		if err := models.DB.C("score").Find(bson.M{"type": "score", "system": "linux", "ip": request.Ip}).One(&s); err != nil {
			log.Println("invalid score for ip:", request.Ip, " default auth type setting: zkp")
			*result = 1
		} else {
			log.Println("ip； ", request.Ip, " registered, score: ", s.Score)
			if s.Score >= 90 {
				*result = 0
			} else {
				*result = 1
			}
		}
		return nil
	}
}

func Sum256(s, r []byte) []byte {
	hash := sha1.New()
	hash.Write(s)
	hash.Write(r)
	return hash.Sum(nil)
}

var curve = edwards25519.NewBlakeSHA256Ed25519()

func authZkp(request *models.AuthRequest, m map[string]string) error {
	PkString := request.Pk
	PkBin, _ := hex.DecodeString(PkString)
	Pk := curve.Point()
	_ = Pk.UnmarshalBinary(PkBin)

	RString := request.R
	Rbin, _ := hex.DecodeString(RString)
	R := curve.Point()
	_ = R.UnmarshalBinary(Rbin)

	zString := request.Z
	zBin, _ := hex.DecodeString(zString)
	z := curve.Scalar()
	_ = z.UnmarshalBinary(zBin)

	GString := m["g"]
	Gbin, _ := hex.DecodeString(GString)
	G := curve.Point()
	_ = G.UnmarshalBinary(Gbin)

	log.Println("calc Pk", Pk.String(), "R:", R.String(), "G:", G.String(), "z:", z.String())

	zG := curve.Point().Mul(z, G)

	C := Sum256([]byte(R.String()), []byte(Pk.String()))
	zGp := curve.Point().Add(R, curve.Point().Mul(curve.Scalar().SetBytes(C), Pk))

	if zG.String() != zGp.String() {
		log.Println("invalid zkp prove: ", zG.String(), "zGp: ", zGp.String())
		return errors.New("validate zkp auth failed")
	}
	log.Println("zkp prove, zG: ", zG.String(), "zGp: ", zGp.String())
	return nil
}

func authPassword(request *models.AuthRequest, m map[string]string) error {
	Password := request.Password
	validatePassword := m["password"]

	if Password != validatePassword {
		log.Println("invalid password verified")
		return errors.New("validate password auth failed")
	}
	log.Println("password verified")
	return nil
}

func (w *Watcher) Auth(ctx context.Context, request *models.AuthRequest, result *int) error {

	authType := request.AuthenticationType
	log.Println("auth begin: type:", authType, " request:", *request)

	var one authing
	if err := models.DB.C("authing").Find(bson.M{"ip": request.Ip}).One(&one); err != nil {
		log.Println("auth info is invalid:", request.Ip, " error:", err)
		return err
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(one.Value), &m); err != nil {
		log.Println("json decode value error:", err, " value:", one.Value)
		return err
	}
	if authType == 1 {
		if err := authZkp(request, m); err != nil {
			log.Println("auth zkp error, e: ", err)
			return err
		} else {
			log.Println("auth zkp success, ip: ", request.Ip)
			*result = 0
		}
	} else if authType == 0 {
		if err := authPassword(request, m); err != nil {
			log.Println("auth password error, e: ", err)
			return err
		} else {
			log.Println("auth password success, ip: ", request.Ip)
			*result = 0
		}
	} else {
		log.Println("auth type invalid, authType:", authType, " request:", *request)
	}

	return nil
}

func auth(ctx context.Context, req *protocol.Message, token string) error {
	if token == authToken {
		return nil
	}
	return errors.New("invalid token")
}

func init() {
	log.Println(models.Config)
	// 从数据库获取证书和RSA私钥
	//ioutil.WriteFile("cert.pem", []byte(models.Config.Cert), 0666)
	//ioutil.WriteFile("private.pem", []byte(models.Config.Private), 0666)
	//certData, _ := ioutil.ReadFile("cert.pem")
	//log.Println("cert.pem内容:", string(certData))
	//privateData, _ := ioutil.ReadFile("private.pem")
	//log.Println("private.pem内容:", string(privateData))
	// 启动心跳线程
	go models.Heartbeat()
	// 启动推送任务线程
	go action.TaskThread()
	// 启动安全检测线程
	go safecheck.ScanMonitorThread()
	// 启动客户端健康检测线程
	go safecheck.HealthCheckThread()
	// ES异步写入线程
	go models.InsertThread()
	go safecheck.Score()
	go safecheck.Predict()

}
func main() {
	//cert, err := tls.LoadX509KeyPair("cert.pem", "private.pem")
	//if err != nil {
		//log.Println("cert error!")
		//return
	//}
	cert, err := tls.LoadX509KeyPair("cert.pem", "private.pem")
	if err != nil {
    		log.Println("cert error!", err)
    		return
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	s := server.NewServer(server.WithTLSConfig(config))
	s.AuthFunc = auth
	s.RegisterName("Watcher", new(Watcher), "")
	log.Println("RPC Server started")
	err = s.Serve("tcp", ":33433")
	if err != nil {
		log.Println(err.Error())
	}
}
