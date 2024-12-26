package models

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/group/edwards25519"
	"io"
	"log"
	"time"
	"yulong-hids/agent/common"
)

var AuthData = new(AuthInfo)

type AuthInfo struct {
	Sk kyber.Scalar `json:"sk"`

	Password string `json:"password"`

	Pk kyber.Point `json:"pk"`
	G  kyber.Point `json:"g"`
}

func InitAuth() {
	var curve = edwards25519.NewBlakeSHA256Ed25519()
	// Get the base of the curve.
	G := curve.Point().Base()

	// Pick a random k from allowed set.
	Sk := curve.Scalar().Pick(curve.RandomStream()) // secret key
	// r = k * G (a.k.a the same operation as r = g^k)
	Pk := curve.Point().Mul(Sk, G) // public key

	// init auth
	AuthData.G = G
	AuthData.Sk = Sk
	AuthData.Pk = Pk
	//authInfo.Rr = r

	sum256 := sha256.Sum256([]byte(string(time.Now().UnixNano())))

	AuthData.Password = hex.EncodeToString(sum256[:])

	log.Println("init and print once: G", G.String(), "Pk", Pk.String(), "Sk", Sk.String())
	log.Println("init and print once: password: ", AuthData.Password)
}

type RegisterRequest struct {
	Pk       string `json:"pk"`
	G        string `json:"g"`
	Password string `json:"password"`
	Ip       string `json:"ip"`
}

func RegRequest() *RegisterRequest {
	request := new(RegisterRequest)
	pkBin, _ := AuthData.Pk.MarshalBinary()
	gBin, _ := AuthData.G.MarshalBinary()
	request.Pk = hex.EncodeToString(pkBin)
	request.G = hex.EncodeToString(gBin)
	request.Password = AuthData.Password
	request.Ip = common.LocalIP
	return request
}

type AuthRequest struct {
	Ip                 string `json:"ip"`
	AuthenticationType int    `json:"authentication_type"`

	Pk string `json:"pk"`
	Z  string `json:"z"`
	R  string `json:"r"`

	Password string `json:"password"`
}

func ZkpRequest() *AuthRequest {
	request := new(AuthRequest)
	var curve = edwards25519.NewBlakeSHA256Ed25519()
	var rnd = make([]byte, 32)
	io.ReadFull(rand.Reader, rnd)
	r := curve.Scalar().SetBytes(rnd)

	R := curve.Point().Mul(r, AuthData.G)
	C := Sum256([]byte(R.String()), []byte(AuthData.Pk.String()))
	z := curve.Scalar().Add(r, curve.Scalar().Mul(curve.Scalar().SetBytes(C), AuthData.Sk))

	Rbin, _ := R.MarshalBinary()
	request.R = hex.EncodeToString(Rbin)

	zbin, _ := z.MarshalBinary()
	request.Z = hex.EncodeToString(zbin)

	PkBin, _ := AuthData.Pk.MarshalBinary()
	request.Pk = hex.EncodeToString(PkBin)

	request.AuthenticationType = 1
	request.Ip = common.LocalIP
	return request
}

func PasswordRequest() *AuthRequest {
	request := new(AuthRequest)

	request.Ip = common.LocalIP
	request.AuthenticationType = 0
	request.Password = AuthData.Password

	return request
}

func Sum256(s, r []byte) []byte {
	hash := sha1.New()
	hash.Write(s)
	hash.Write(r)
	return hash.Sum(nil)
}
