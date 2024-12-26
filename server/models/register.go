package models

type RegisterRequest struct {
	Pk       string `json:"pk"`
	G        string `json:"g"`
	Password string `json:"password"`
	Ip       string `json:"ip"`
}

type AuthRequest struct {
	Ip                 string `json:"ip"`
	AuthenticationType int    `json:"authentication_type"`

	Pk string `json:"pk"`
	Z  string `json:"z"`
	R  string `json:"r"`

	Password string `json:"password"`
}
