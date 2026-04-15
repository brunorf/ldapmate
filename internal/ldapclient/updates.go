package ldapclient

import (
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
)

func (b *LDAPBackend) UpdateUser(userDN, fullname, shell string, aliases []string, groups []string, password string) error {
	// 1. Update basic attributes
	mod := ldap.NewModifyRequest(userDN, nil)
	mod.Replace("cn", []string{fullname})
	sn := fullname
	if i := strings.LastIndex(fullname, " "); i != -1 {
		sn = fullname[i+1:]
	}
	mod.Replace("sn", []string{sn})
	mod.Replace("loginShell", []string{shell})

	// 2. Update password if provided
	if password != "" {
		mod.Replace("userPassword", []string{b.hashPassword(password)})
	}

	// 3. Update aliases (keeping main UID)
	parts := strings.Split(userDN, ",")
	rdnUID := strings.Split(parts[0], "=")[1]

	uidMap := map[string]bool{rdnUID: true}
	var newUIDList []string
	newUIDList = append(newUIDList, rdnUID)

	for _, a := range aliases {
		a = strings.TrimSpace(a)
		if a != "" && !uidMap[a] {
			uidMap[a] = true
			newUIDList = append(newUIDList, a)
		}
	}

	mod.Replace("uid", newUIDList)

	if err := b.Conn.Modify(mod); err != nil {
		return fmt.Errorf("error updating user info: %v", err)
	}

	// 4. Sync Groups (Diff)
	sr := ldap.NewSearchRequest(
		"ou=Groups,"+b.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(member=%s)", userDN), []string{"cn"}, nil,
	)
	res, err := b.Conn.Search(sr)
	if err != nil {
		return fmt.Errorf("error fetching current groups: %v", err)
	}

	currentGroupsMap := make(map[string]bool)
	for _, entry := range res.Entries {
		currentGroupsMap[entry.GetAttributeValue("cn")] = true
	}

	newGroupsMap := make(map[string]bool)
	for _, g := range groups {
		newGroupsMap[g] = true
	}

	// Remove from groups
	for g := range currentGroupsMap {
		if !newGroupsMap[g] {
			b.RemoveFromGroup(userDN, g)
		}
	}

	// Add to groups
	for g := range newGroupsMap {
		if !currentGroupsMap[g] {
			b.AddToGroup(userDN, g)
		}
	}

	return nil
}

func (b *LDAPBackend) RemoveFromGroup(userDN, groupCN string) error {
	sr := ldap.NewSearchRequest(
		"ou=Groups,"+b.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(cn=%s)", groupCN), []string{"dn"}, nil,
	)
	res, err := b.Conn.Search(sr)
	if err != nil || len(res.Entries) == 0 {
		return nil
	}

	mod := ldap.NewModifyRequest(res.Entries[0].DN, nil)
	mod.Delete("member", []string{userDN})
	return b.Conn.Modify(mod)
}

func (b *LDAPBackend) AddToGroup(userDN, groupCN string) error {
	sr := ldap.NewSearchRequest(
		"ou=Groups,"+b.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(cn=%s)", groupCN), []string{"dn"}, nil,
	)
	res, err := b.Conn.Search(sr)
	if err != nil || len(res.Entries) == 0 {
		return nil
	}

	mod := ldap.NewModifyRequest(res.Entries[0].DN, nil)
	mod.Add("member", []string{userDN})
	return b.Conn.Modify(mod)
}

func (b *LDAPBackend) UpdateGroupInfo(groupDN, description string) error {
	mod := ldap.NewModifyRequest(groupDN, nil)
	mod.Replace("description", []string{description})
	return b.Conn.Modify(mod)
}

func (b *LDAPBackend) UpdateGroupMembers(groupDN string, newMembers []string) error {
	sr := ldap.NewSearchRequest(
		groupDN, ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
		"(objectClass=*)", []string{"member"}, nil,
	)
	res, err := b.Conn.Search(sr)
	if err != nil || len(res.Entries) == 0 {
		return fmt.Errorf("group not found: %v", err)
	}

	currentMembers := res.Entries[0].GetAttributeValues("member")
	currentMap := make(map[string]bool)
	for _, m := range currentMembers {
		currentMap[m] = true
	}

	newMap := make(map[string]bool)
	for _, m := range newMembers {
		newMap[m] = true
	}

	var toAdd []string
	for m := range newMap {
		if !currentMap[m] {
			toAdd = append(toAdd, m)
		}
	}

	var toRemove []string
	for m := range currentMap {
		if !newMap[m] {
			toRemove = append(toRemove, m)
		}
	}

	if len(toAdd) == 0 && len(toRemove) == 0 {
		return nil
	}

	mod := ldap.NewModifyRequest(groupDN, nil)
	if len(toAdd) > 0 {
		mod.Add("member", toAdd)
	}
	if len(toRemove) > 0 {
		mod.Delete("member", toRemove)
	}

	return b.Conn.Modify(mod)
}

func (b *LDAPBackend) GetUserGroups(userDN string) ([]Group, error) {
	sr := ldap.NewSearchRequest(
		"ou=Groups,"+b.BaseDN, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(member=%s)", userDN), []string{"cn", "description", "gidNumber"}, nil,
	)
	res, err := b.Conn.Search(sr)
	if err != nil {
		return nil, err
	}

	var groups []Group
	for _, entry := range res.Entries {
		groups = append(groups, Group{
			CN:   entry.GetAttributeValue("cn"),
			GID:  entry.GetAttributeValue("gidNumber"),
			Desc: entry.GetAttributeValue("description"),
		})
	}
	return groups, nil
}
