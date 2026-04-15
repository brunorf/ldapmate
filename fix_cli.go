package main

import (
	"io/ioutil"
	"strings"
)

func main() {
	c, _ := ioutil.ReadFile("/home/bruno/Projetos/ldap/ldapmate/cmd/web/cli.go")
	s := string(c)
	s = strings.Replace(s, `func handleUserCommand`, `
func handleUserCommand`, 1)
	ioutil.WriteFile("/home/bruno/Projetos/ldap/ldapmate/cmd/web/cli.go", []byte(s), 0644)
}
