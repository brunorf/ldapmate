package main

import (
	"ldapmate/templates"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

func renderTemplate(w http.ResponseWriter, r *http.Request, tmplFile string, data map[string]interface{}) {
	session, _ := store.Get(r, "ldap-session")
	val := TemplateData{
		Session: session,
		Flashes: getFlashes(w, r, session),
		Data:    data,
	}
	tmpl, err := template.ParseFS(templates.FS, "base.html", tmplFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Template error: %v", err), 500)
		return
	}
	tmpl.ExecuteTemplate(w, "base.html", val)
}

func addUserHandler(w http.ResponseWriter, r *http.Request) {
	ldap := getLDAP(r)
	if ldap == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if r.Method == "POST" {
		fullname := r.FormValue("fullname")
		username := r.FormValue("username")
		password := r.FormValue("password")
		generatedPass := false
		if password == "" {
			password = generateRandomPassword(12)
			generatedPass = true
		}
		category := r.FormValue("category")
		shell := r.FormValue("loginShell")
		aliasesStr := r.FormValue("aliases")

		aliases := strings.Split(aliasesStr, ",")
		for i := range aliases {
			aliases[i] = strings.TrimSpace(aliases[i])
		}

		err := ldap.AddUser(username, fullname, password, category, aliases, shell)
		session, _ := store.Get(r, "ldap-session")
		if err != nil {
			session.AddFlash(fmt.Sprintf("Erro ao adicionar usuário: %v", err))
		} else {
			if generatedPass {
				session.AddFlash(fmt.Sprintf("Usuário criado com sucesso! Senha: %s", password))
			} else {
				session.AddFlash("Usuário criado com sucesso!")
			}
		}
		session.Save(r, w)
		http.Redirect(w, r, "/users", http.StatusFound)
		return
	}

	groups, _ := ldap.ListGroups()
	renderTemplate(w, r, "form_user.html", map[string]interface{}{
		"AvailableGroups": groups,
	})
}

func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	ldap := getLDAP(r)
	if ldap != nil {
		err := ldap.DeleteUser(r.URL.Query().Get("dn"))
		session, _ := store.Get(r, "ldap-session")
		if err != nil {
			session.AddFlash(fmt.Sprintf("Erro ao deletar: %v", err))
		} else {
			session.AddFlash("Deletado com sucesso!")
		}
		session.Save(r, w)
	}
	http.Redirect(w, r, "/users", http.StatusFound)
}

func addGroupHandler(w http.ResponseWriter, r *http.Request) {
	ldap := getLDAP(r)
	if ldap == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if r.Method == "POST" {
		name := r.FormValue("name")
		gid := r.FormValue("gid")
		desc := r.FormValue("description")

		r.ParseForm()
		members := r.Form["members"]

		err := ldap.CreateGroup(name, gid, desc, members)
		session, _ := store.Get(r, "ldap-session")
		if err != nil {
			session.AddFlash(fmt.Sprintf("Erro: %v", err))
		} else {
			session.AddFlash("Grupo criado com sucesso!")
		}
		session.Save(r, w)
		http.Redirect(w, r, "/groups", http.StatusFound)
		return
	}

	users, _ := ldap.ListUsers()
	renderTemplate(w, r, "form_group.html", map[string]interface{}{
		"EditMode":       false,
		"AvailableUsers": users,
		"SuggestedGID":   "10001",
	})
}

func deleteGroupHandler(w http.ResponseWriter, r *http.Request) {
	ldap := getLDAP(r)
	if ldap != nil {
		parts := strings.Split(r.URL.Path, "/")
		cn := parts[len(parts)-1]
		if cn == "ldap_users" {
			session, _ := store.Get(r, "ldap-session")
			session.AddFlash("Não é permitido apagar o grupo base ldap_users.")
			session.Save(r, w)
		} else {
			err := ldap.DeleteGroup(cn)
			session, _ := store.Get(r, "ldap-session")
			if err != nil {
				session.AddFlash(fmt.Sprintf("Erro ao deletar grupo: %v", err))
			} else {
				session.AddFlash("Grupo removido!")
			}
			session.Save(r, w)
		}
	}
	http.Redirect(w, r, "/groups", http.StatusFound)
}

func manageKeysHandler(w http.ResponseWriter, r *http.Request) {
	ldap := getLDAP(r)
	if ldap == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	dn := r.URL.Query().Get("dn")
	if r.Method == "POST" {
		key := r.FormValue("key")
		err := ldap.AddSSHKey(dn, key)
		session, _ := store.Get(r, "ldap-session")
		if err != nil {
			session.AddFlash(fmt.Sprintf("Erro: %v", err))
		} else {
			session.AddFlash("Chave adicionada!")
		}
		session.Save(r, w)
		http.Redirect(w, r, "/manage_keys?dn="+dn, http.StatusFound)
		return
	}

	user, _ := ldap.GetUser(dn)
	keys, _ := ldap.GetSSHKeys(dn)

	renderTemplate(w, r, "manage_keys.html", map[string]interface{}{
		"User": user,
		"Keys": keys,
	})
}

func deleteKeyHandler(w http.ResponseWriter, r *http.Request) {
	ldap := getLDAP(r)
	if ldap != nil {
		dn := r.URL.Query().Get("dn")
		key := r.FormValue("key")
		err := ldap.RemoveSSHKey(dn, key)
		session, _ := store.Get(r, "ldap-session")
		if err == nil {
			session.AddFlash("Chave removida.")
		} else {
			session.AddFlash(fmt.Sprintf("Erro: %v", err))
		}
		session.Save(r, w)
		http.Redirect(w, r, "/manage_keys?dn="+dn, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/users", http.StatusFound)
}
