package config

import (
	"encoding/json"
	"os"
)

type AppConfig struct {
	LDAPHost        string            `json:"ldap_host"`
	LDAPBase        string            `json:"ldap_base"`
	AdminDN         string            `json:"admin_dn"`
	SecretKey       string            `json:"secret_key"`
	CategoryRanges  map[string][2]int `json:"category_ranges"`
	DefaultCategory string            `json:"default_category"`
	ServerPort      string            `json:"server_port"`
}

var Cfg AppConfig
var Loaded bool

func LoadConfig(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &Cfg)
	if err == nil {
		Loaded = true
	}
	return err
}

func CreateDefaultConfig(path string) error {
	Cfg = AppConfig{
		LDAPHost:  "localhost",
		LDAPBase:  "dc=example,dc=com",
		AdminDN:   "cn=admin,dc=example,dc=com",
		SecretKey: "sua_chave_secreta_aqui",
		CategoryRanges: map[string][2]int{
			"admin":   {10000, 10999},
			"service": {11000, 19999},
			"prof":    {20000, 29999},
			"aluno":   {30000, 89999},
			"guest":   {90000, 99999},
		},
		DefaultCategory: "aluno",
		ServerPort:      ":8080",
	}
	b, err := json.MarshalIndent(Cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
