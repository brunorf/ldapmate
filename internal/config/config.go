package config

import (
_ "embed"
"encoding/json"
"log"
)

//go:embed config.json
var configFile []byte

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

func init() {
err := json.Unmarshal(configFile, &Cfg)
if err != nil {
log.Fatalf("Erro ao carregar configuração embutida (config.json): %v", err)
}
}
