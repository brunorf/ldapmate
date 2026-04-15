package main

import (
	"fmt"
	"net/http"
	"strings"
)

func editUserHandler(w http.ResponseWriter, r *http.Request) {
	ldap := getLDAP(r)
	if ldap == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	dn := r.URL.Query().Get("dn")

	if r.Method == "POST" {
		fullname := r.FormValue("fullname")
		password := r.FormValue("password")
		shell := r.FormValue("loginShell")
		aliasesStr := r.FormValue("aliases")

		var aliases []string
		if aliasesStr != "" {
			for _, a := range strings.Split(aliasesStr, ",") {
				aliases = append(aliases, strings.TrimSpace(a))
			}
		}

		r.ParseForm()
		groups := r.Form["groups"]

		err := ldap.UpdateUser(dn, fullname, shell, aliases, groups, password)
		session, _ := store.Get(r, "ldap-session")
		if err != nil {
			session.AddFlash(fmt.Sprintf("Erro ao atualizar: %v", err))
		} else {
			session.AddFlash("Usuário atualizado com sucesso")
		}
		session.Save(r, w)
		http.Redirect(w, r, "/users", http.StatusFound)
		return
	}

	user, _ := ldap.GetUser(dn)

	groups, _ := ldap.ListGroups()

	// Get current user groups
	var userGroupNames []string
	userGroups, err := ldap.GetUserGroups(dn)

	if err == nil {
		for _, g := range userGroups {
			userGroupNames = append(userGroupNames, g.CN)
		}
	}

	renderTemplateLocal(w, r, "form_user.html", map[string]interface{}{
		"User":            user,
		"AvailableGroups": groups,
		"UserGroups":      userGroupNames,
	})
}

func editGroupHandler(w http.ResponseWriter, r *http.Request) {
	ldap := getLDAP(r)
	if ldap == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	cn := parts[len(parts)-1]

	group, err := ldap.GetGroup(cn)
	if err != nil {
		session, _ := store.Get(r, "ldap-session")
		session.AddFlash("Grupo não encontrado.")
		session.Save(r, w)
		http.Redirect(w, r, "/groups", http.StatusFound)
		return
	}

	if r.Method == "POST" {
		desc := r.FormValue("description")

		r.ParseForm()
		members := r.Form["members"]

		ldap.UpdateGroupInfo("cn="+cn+",ou=Groups,"+ldap.BaseDN, desc)
		err := ldap.UpdateGroupMembers("cn="+cn+",ou=Groups,"+ldap.BaseDN, members)

		session, _ := store.Get(r, "ldap-session")
		if err != nil {
			session.AddFlash(fmt.Sprintf("Erro ao atualizar: %v", err))
		} else {
			session.AddFlash("Grupo atualizado com sucesso!")
		}
		session.Save(r, w)
		http.Redirect(w, r, "/groups", http.StatusFound)
		return
	}

	users, _ := ldap.ListUsers()

	renderTemplateLocal(w, r, "form_group.html", map[string]interface{}{
		"EditMode":       true,
		"Group":          group,
		"AvailableUsers": users,
		"GroupMembers":   group.Members,
	})
}
