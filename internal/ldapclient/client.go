package ldapclient

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/go-ldap/ldap/v3"
)

type LDAPBackend struct {
	Server   string
	BindDN   string
	Password string
	BaseDN   string
	Conn     *ldap.Conn
}

type User struct {
	UID       string
	CN        string
	UIDNumber int
	GIDNumber string
	DN        string
	Shell     string
	Category  string
	Aliases   []string
	Groups    []string
}

type Group struct {
	CN      string
	GID     string
	Desc    string
	Members []string
}

func NewLDAPBackend(server, bindDN, password, baseDN string) *LDAPBackend {
	return &LDAPBackend{
		Server:   server,
		BindDN:   bindDN,
		Password: password,
		BaseDN:   baseDN,
	}
}

func (b *LDAPBackend) Connect() (bool, error) {
	conn, err := ldap.DialURL(fmt.Sprintf("ldap://%s", b.Server))
	if err != nil {
		return false, err
	}

	err = conn.Bind(b.BindDN, b.Password)
	if err != nil {
		conn.Close()
		return false, err
	}

	b.Conn = conn
	return true, nil
}

func (b *LDAPBackend) getCategoryFromUID(uid int) string {
	switch {
	case uid >= 10000 && uid <= 10999:
		return "admin"
	case uid >= 11000 && uid <= 19999:
		return "service"
	case uid >= 20000 && uid <= 29999:
		return "prof"
	case uid >= 30000 && uid <= 89999:
		return "aluno"
	case uid >= 90000 && uid <= 99999:
		return "guest"
	}
	return "unknown"
}

func (b *LDAPBackend) ListUsers() ([]User, error) {
	searchRequest := ldap.NewSearchRequest(
		"ou=People,"+b.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=posixAccount)",
		[]string{"uid", "cn", "uidNumber", "gidNumber", "homeDirectory", "loginShell"},
		nil,
	)

	sr, err := b.Conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	var users []User
	for _, entry := range sr.Entries {
		uidNumStr := entry.GetAttributeValue("uidNumber")
		uidNum, _ := strconv.Atoi(uidNumStr)

		shell := entry.GetAttributeValue("loginShell")
		if shell == "" {
			shell = "/bin/bash"
		}

		dn := entry.DN
		parts := strings.Split(dn, ",")
		rdnUID := ""
		if len(parts) > 0 && strings.HasPrefix(parts[0], "uid=") {
			rdnUID = strings.TrimPrefix(parts[0], "uid=")
		}

		allUIDs := entry.GetAttributeValues("uid")
		var aliases []string
		primaryUID := rdnUID
		if primaryUID == "" && len(allUIDs) > 0 {
			primaryUID = allUIDs[0]
		}

		for _, u := range allUIDs {
			if u != primaryUID {
				aliases = append(aliases, u)
			}
		}

		users = append(users, User{
			UID:       primaryUID,
			CN:        entry.GetAttributeValue("cn"),
			UIDNumber: uidNum,
			GIDNumber: entry.GetAttributeValue("gidNumber"),
			DN:        entry.DN,
			Shell:     shell,
			Category:  b.getCategoryFromUID(uidNum),
			Aliases:   aliases,
		})
	}

	categoryWeight := map[string]int{
		"admin":   1,
		"prof":    2,
		"aluno":   3,
		"service": 4,
		"guest":   5,
	}

	sort.Slice(users, func(i, j int) bool {
		w1 := categoryWeight[users[i].Category]
		if w1 == 0 {
			w1 = 99
		}
		w2 := categoryWeight[users[j].Category]
		if w2 == 0 {
			w2 = 99
		}

		if w1 != w2 {
			return w1 < w2
		}
		return users[i].UIDNumber < users[j].UIDNumber
	})

	return users, nil
}
func (b *LDAPBackend) ListGroups() ([]Group, error) {

	searchRequest := ldap.NewSearchRequest(
		"ou=Groups,"+b.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=posixGroup)",
		[]string{"cn", "gidNumber", "description", "member"},
		nil,
	)

	sr, err := b.Conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	var groups []Group
	for _, entry := range sr.Entries {
		cn := entry.GetAttributeValue("cn")
		if cn == "ldap_users" {
			continue
		}

		desc := entry.GetAttributeValue("description")
		if desc == "" {
			desc = cn
		}

		groups = append(groups, Group{
			CN:      cn,
			GID:     entry.GetAttributeValue("gidNumber"),
			Desc:    desc,
			Members: entry.GetAttributeValues("member"),
		})
	}
	return groups, nil
}
