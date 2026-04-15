package ldapclient

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"dirmate/internal/config"
	"github.com/go-ldap/ldap/v3"
)

var ranges = map[string][2]int{
	"admin":   {10000, 10999},
	"service": {11000, 19999},
	"prof":    {20000, 29999},
	"aluno":   {30000, 89999},
	"guest":   {90000, 99999},
}

func (b *LDAPBackend) getNextUID(category string) int {
	rng, ok := config.Cfg.CategoryRanges[category]
	if !ok {
		rng = config.Cfg.CategoryRanges[config.Cfg.DefaultCategory]
	}
	start, end := rng[0], rng[1]

	sr := ldap.NewSearchRequest(
		b.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=posixAccount)", []string{"uidNumber"}, nil,
	)

	res, err := b.Conn.Search(sr)
	if err != nil {
		return start
	}

	maxUid := start - 1
	for _, entry := range res.Entries {
		uid, err := strconv.Atoi(entry.GetAttributeValue("uidNumber"))
		if err == nil && uid >= start && uid <= end {
			if uid > maxUid {
				maxUid = uid
			}
		}
	}

	if maxUid >= start {
		return maxUid + 1
	}
	return start
}

func (b *LDAPBackend) hashPassword(password string) string {
	if strings.HasPrefix(password, "$") {
		return "{CRYPT}" + password
	}
	salt := make([]byte, 4)
	rand.Read(salt)
	h := sha1.New()
	h.Write([]byte(password))
	h.Write(salt)
	return "{SSHA}" + base64.StdEncoding.EncodeToString(append(h.Sum(nil), salt...))
}

func (b *LDAPBackend) GetUser(userDN string) (*User, error) {
	sr := ldap.NewSearchRequest(
		userDN, ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=posixAccount)", []string{"uid", "cn", "sn", "homeDirectory", "loginShell", "uidNumber", "gidNumber"}, nil,
	)
	res, err := b.Conn.Search(sr)
	if err != nil || len(res.Entries) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	entry := res.Entries[0]
	uidNum, _ := strconv.Atoi(entry.GetAttributeValue("uidNumber"))
	shell := entry.GetAttributeValue("loginShell")
	if shell == "" {
		shell = "/bin/bash"
	}

	parts := strings.Split(userDN, ",")
	rdnUID := strings.Split(parts[0], "=")[1]

	allUIDs := entry.GetAttributeValues("uid")
	var aliases []string
	for _, u := range allUIDs {
		if u != rdnUID {
			aliases = append(aliases, u)
		}
	}

	return &User{
		UID:       rdnUID,
		CN:        entry.GetAttributeValue("cn"),
		UIDNumber: uidNum,
		GIDNumber: entry.GetAttributeValue("gidNumber"),
		DN:        entry.DN,
		Shell:     shell,
		Category:  b.getCategoryFromUID(uidNum),
		Aliases:   aliases,
	}, nil
}

func (b *LDAPBackend) GetSSHKeys(userDN string) ([]string, error) {
	sr := ldap.NewSearchRequest(
		userDN, ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=ldapPublicKey)", []string{"sshPublicKey"}, nil,
	)
	res, err := b.Conn.Search(sr)
	if err != nil || len(res.Entries) == 0 {
		return nil, nil // No keys or objectClass doesn't match
	}
	return res.Entries[0].GetAttributeValues("sshPublicKey"), nil
}

func (b *LDAPBackend) AddSSHKey(userDN, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("empty key")
	}
	mod := ldap.NewModifyRequest(userDN, nil)
	mod.Add("sshPublicKey", []string{key})
	return b.Conn.Modify(mod)
}

func (b *LDAPBackend) RemoveSSHKey(userDN, key string) error {
	mod := ldap.NewModifyRequest(userDN, nil)
	mod.Delete("sshPublicKey", []string{strings.TrimSpace(key)})
	return b.Conn.Modify(mod)
}

func (b *LDAPBackend) GetGroup(cn string) (*Group, error) {
	sr := ldap.NewSearchRequest(
		"ou=Groups,"+b.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(cn=%s)", cn), []string{"cn", "gidNumber", "description", "member"}, nil,
	)
	res, err := b.Conn.Search(sr)
	if err != nil || len(res.Entries) == 0 {
		return nil, fmt.Errorf("group not found")
	}
	entry := res.Entries[0]
	desc := entry.GetAttributeValue("description")
	if desc == "" {
		desc = entry.GetAttributeValue("cn")
	}
	return &Group{
		CN:      entry.GetAttributeValue("cn"),
		GID:     entry.GetAttributeValue("gidNumber"),
		Desc:    desc,
		Members: entry.GetAttributeValues("member"),
	}, nil
}

// And the rest: User CRUD and Group CRUD will be implemented in handlers by calling proper ldap modify.
// Actually, it's better to implement the AddUser etc here keeping the abstraction clean.

func (b *LDAPBackend) CreateGroup(name, gid, desc string, members []string) error {
	req := ldap.NewAddRequest(fmt.Sprintf("cn=%s,ou=Groups,%s", name, b.BaseDN), nil)
	req.Attribute("objectClass", []string{"top", "groupOfNames", "posixGroup"})
	req.Attribute("cn", []string{name})
	req.Attribute("gidNumber", []string{gid})
	if desc == "" {
		desc = name
	}
	req.Attribute("description", []string{desc})
	if len(members) == 0 {
		return fmt.Errorf("at least one member required")
	}
	req.Attribute("member", members)
	return b.Conn.Add(req)
}

func (b *LDAPBackend) DeleteGroup(cn string) error {
	return b.Conn.Del(ldap.NewDelRequest(fmt.Sprintf("cn=%s,ou=Groups,%s", cn, b.BaseDN), nil))
}

func (b *LDAPBackend) DeleteUser(dn string) error {
	return b.Conn.Del(ldap.NewDelRequest(dn, nil))
}

func (b *LDAPBackend) AddUser(username, fullname, password, category string, aliases []string, shell string) error {
	uidNum := b.getNextUID(category)
	dn := fmt.Sprintf("uid=%s,ou=People,%s", username, b.BaseDN)

	req := ldap.NewAddRequest(dn, nil)
	req.Attribute("objectClass", []string{"inetOrgPerson", "posixAccount", "ldapPublicKey", "shadowAccount"})
	req.Attribute("cn", []string{fullname})
	sn := fullname
	if i := strings.LastIndex(fullname, " "); i != -1 {
		sn = fullname[i+1:]
	}
	req.Attribute("sn", []string{sn})

	uidAttr := []string{username}
	for _, a := range aliases {
		a = strings.TrimSpace(a)
		if a != "" && a != username {
			uidAttr = append(uidAttr, a)
		}
	}
	req.Attribute("uid", uidAttr)
	req.Attribute("uidNumber", []string{strconv.Itoa(uidNum)})
	req.Attribute("gidNumber", []string{"10000"})
	req.Attribute("homeDirectory", []string{"/home/" + username})
	req.Attribute("loginShell", []string{shell})
	req.Attribute("userPassword", []string{b.hashPassword(password)})

	return b.Conn.Add(req)
}
