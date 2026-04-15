package main

import (
	"encoding/gob"
	"fmt"
	"dirmate/internal/config"
	"dirmate/templates"
	"html/template"
	"log"
	"net/http"

	"dirmate/internal/ldapclient"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

var store *sessions.CookieStore


func init() {
	gob.Register(&ldapclient.User{})
	gob.Register(&ldapclient.Group{})
}

type TemplateData struct {
	Session *sessions.Session
	Flashes []string
	Data    map[string]interface{}
}

func getFlashes(w http.ResponseWriter, r *http.Request, session *sessions.Session) []string {
	var flashes []string
	if flashesObj := session.Flashes(); len(flashesObj) > 0 {
		for _, f := range flashesObj {
			flashes = append(flashes, fmt.Sprintf("%v", f))
		}
		session.Save(r, w)
	}
	return flashes
}

func getLDAP(r *http.Request) *ldapclient.LDAPBackend {
	session, _ := store.Get(r, "ldap-session")
	pass, ok := session.Values["auth_pass"].(string)
	if !ok || pass == "" {
		return nil
	}
	backend := ldapclient.NewLDAPBackend(config.Cfg.LDAPHost, config.Cfg.AdminDN, pass, config.Cfg.LDAPBase)
	ok, _ = backend.Connect()
	if !ok {
		return nil
	}
	return backend
}

func main() {
	if err := config.LoadConfig("config.json"); err != nil {
		fmt.Println("Criando config.json padrão. Ajuste e rode novamente!")
		config.CreateDefaultConfig("config.json")
		return
	}
	store = sessions.NewCookieStore([]byte(config.Cfg.SecretKey))

	if !runCLI() {
		return
	}
	r := mux.NewRouter()

	r.HandleFunc("/login", loginHandler).Methods("GET", "POST")
	r.HandleFunc("/logout", logoutHandler).Methods("GET")
	r.HandleFunc("/users", usersHandler).Methods("GET")
	r.HandleFunc("/edit", editUserHandler).Methods("GET", "POST")
	r.HandleFunc("/add", addUserHandler).Methods("GET", "POST")
	r.HandleFunc("/delete", deleteUserHandler).Methods("GET", "POST")
	r.HandleFunc("/manage_keys", manageKeysHandler).Methods("GET", "POST")
	r.HandleFunc("/delete_key", deleteKeyHandler).Methods("POST")

	r.HandleFunc("/groups", groupsHandler).Methods("GET")
	r.HandleFunc("/groups/add", addGroupHandler).Methods("GET", "POST")
	r.HandleFunc("/groups/edit/{cn}", editGroupHandler).Methods("GET", "POST")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	}).Methods("GET")

	log.Println("Starting server on " + config.Cfg.ServerPort)
	log.Fatal(http.ListenAndServe(config.Cfg.ServerPort, r))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "ldap-session")
	if val, ok := session.Values["auth_pass"].(string); ok && val != "" {
		http.Redirect(w, r, "/users", http.StatusFound)
		return
	}
	if r.Method == "POST" {
		password := r.FormValue("password")
		backend := ldapclient.NewLDAPBackend(config.Cfg.LDAPHost, config.Cfg.AdminDN, password, config.Cfg.LDAPBase)
		ok, err := backend.Connect()
		if ok {
			session.Values["auth_pass"] = password
			session.Save(r, w)
			http.Redirect(w, r, "/users", http.StatusFound)
			return
		}
		session.AddFlash(fmt.Sprintf("Erro: %v", err))
		session.Save(r, w)
	}
	renderTemplateLocal(w, r, "login.html", map[string]interface{}{})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "ldap-session")
	session.Values["auth_pass"] = ""
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	ldap := getLDAP(r)
	if ldap == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	cat := r.URL.Query().Get("cat")
	if cat == "" {
		cat = "all"
	}
	allUsers, err := ldap.ListUsers()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	var users []ldapclient.User
	if cat == "all" {
		users = allUsers
	} else {
		for _, u := range allUsers {
			if u.Category == cat {
				users = append(users, u)
			}
		}
	}
	renderTemplateLocal(w, r, "users.html", map[string]interface{}{
		"Users":      users,
		"CurrentCat": cat,
	})
}

func groupsHandler(w http.ResponseWriter, r *http.Request) {
	ldap := getLDAP(r)
	if ldap == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	groups, err := ldap.ListGroups()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	renderTemplateLocal(w, r, "groups.html", map[string]interface{}{
		"Groups": groups,
	})
}

func renderTemplateLocal(w http.ResponseWriter, r *http.Request, tmplFile string, data map[string]interface{}) {
	session, _ := store.Get(r, "ldap-session")
	val := TemplateData{
		Session: session,
		Flashes: getFlashes(w, r, session),
		Data:    data,
	}
	tmpl, err := template.ParseFS(templates.FS, "base.html", tmplFile)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	tmpl.ExecuteTemplate(w, "base.html", val)
}
