package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"syscall"

	"dirmate/internal/config"
	"dirmate/internal/ldapclient"

	"github.com/peterh/liner"
	"golang.org/x/term"
)

func runCLI() bool {
	if len(os.Args) < 2 {
		printMainHelp()
		return false
	}

	cmd := os.Args[1]

	if cmd == "serve" || cmd == "server" {
		return true
	}

	if cmd == "shell" {
		runInteractiveShell()
		return false
	}

	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		printMainHelp()
		return false
	}

	fmt.Print("Digite a senha do administrador LDAP (admin-pass): ")
	bytePass, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Println("Erro ao ler senha.")
		os.Exit(1)
	}
	adminPass := strings.TrimSpace(string(bytePass))

	backend := ldapclient.NewLDAPBackend(config.Cfg.LDAPHost, config.Cfg.AdminDN, adminPass, config.Cfg.LDAPBase)
	ok, err := backend.Connect()
	if !ok || err != nil {
		fmt.Printf("Erro na conexão LDAP: %v\n", err)
		os.Exit(1)
	}

	handleCommand(backend, os.Args[1:])
	return false
}

func printMainHelp() {
	fmt.Println("Uso: ./dirmate <comando> [argumentos]")
	fmt.Println("\nComandos disponíveis:")
	fmt.Println("  serve / server Inicia o servidor web")
	fmt.Println("  shell          Inicia um shell interativo (requer senha apenas uma vez)")
	fmt.Println("  user           Gerenciar usuários (add, edit, list, del)")
	fmt.Println("  group          Gerenciar grupos (add, edit, list, del)")
	fmt.Println("  key            Gerenciar chaves SSH (add, list, del)")
	fmt.Println("\nExemplos:")
	fmt.Println("  ./dirmate serve")
	fmt.Println("  ./dirmate shell")
	fmt.Println("  ./dirmate user add -uid fulano -name \"Fulano\" -type aluno")
	fmt.Println("  ./dirmate group list")
	fmt.Println("\nPara ajuda específica, digite: ./dirmate <comando> -h")
}

func printShellHelp() {
	fmt.Println("\nComandos disponíveis:")
	fmt.Println("  user add|edit|list|del [args]   Gerencia usuários")
	fmt.Println("  group add|edit|list|del [args]  Gerencia grupos")
	fmt.Println("  key add|list|del [args]         Gerencia chaves SSH")
	fmt.Println("  help                            Exibe esta ajuda")
	fmt.Println("  exit / quit                     Sai do shell")
}

func runInteractiveShell() {
	fmt.Print("Digite a senha do administrador LDAP (admin-pass): ")
	bytePass, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Println("Erro ao ler senha.")
		os.Exit(1)
	}
	adminPass := strings.TrimSpace(string(bytePass))

	backend := ldapclient.NewLDAPBackend(config.Cfg.LDAPHost, config.Cfg.AdminDN, adminPass, config.Cfg.LDAPBase)
	ok, err := backend.Connect()
	if !ok || err != nil {
		fmt.Printf("Erro na conexão LDAP: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Shell Interativo LDAP ===")
	fmt.Println("Digite 'help' para ajuda ou 'exit' para sair. Setas ↑/↓ para histórico.")

	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	for {
		cmdLine, err := line.Prompt("ldap> ")
		if err != nil {
			break
		}
		cmdLine = strings.TrimSpace(cmdLine)
		if cmdLine == "" {
			continue
		}
		line.AppendHistory(cmdLine)

		if cmdLine == "exit" || cmdLine == "quit" {
			break
		}
		if cmdLine == "help" {
			printShellHelp()
			continue
		}

		args := parseCommandLine(cmdLine)
		if len(args) == 0 {
			continue
		}

		handleCommand(backend, args)
	}
}

func parseCommandLine(cmd string) []string {
	var args []string
	var current string
	inQuotes := false

	for _, r := range cmd {
		if r == '"' {
			inQuotes = !inQuotes
		} else if r == ' ' && !inQuotes {
			if current != "" {
				args = append(args, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		args = append(args, current)
	}
	return args
}

func handleCommand(backend *ldapclient.LDAPBackend, args []string) {
	if len(args) == 0 {
		return
	}

	domain := args[0]
	if len(args) < 2 && domain != "user" && domain != "group" && domain != "key" {
		fmt.Println("Comando inválido. Use user, group ou key.")
		return
	}

	var action string
	var fArgs []string
	if len(args) >= 2 {
		action = args[1]
		fArgs = args[2:]
	}

	switch domain {
	case "user":
		handleUserCommand(backend, action, fArgs)
	case "group":
		handleGroupCommand(backend, action, fArgs)
	case "key":
		handleKeyCommand(backend, action, fArgs)
	default:
		fmt.Println("Domínio inválido:", domain)
	}
}

func handleUserCommand(backend *ldapclient.LDAPBackend, action string, args []string) {
	fs := flag.NewFlagSet("user "+action, flag.ContinueOnError)
	uid := fs.String("uid", "", "UIDs separados por vírgula")
	name := fs.String("name", "", "Nome completo")
	cat := fs.String("type", "", "Categoria do usuário")
	pass := fs.String("password", "", "Senha")
	shell := fs.String("shell", "/bin/bash", "Login shell")
	groups := fs.String("groups", "", "Grupos associados")

	if action == "list" {
		users, _ := backend.ListUsers()
		fmt.Println("--- Usuários ---")
		for _, u := range users {
			fmt.Printf("%-15s | %-20s | %s\n", u.UID, u.CN, u.Category)
		}
		return
	}

	if err := fs.Parse(args); err != nil {
		return
	}

	if action == "del" {
		if *uid == "" {
			fmt.Println("Erro: -uid obrigatório.")
			return
		}
		dn := "uid=" + strings.Split(*uid, ",")[0] + ",ou=People," + config.Cfg.LDAPBase
		if err := backend.DeleteUser(dn); err != nil {
			fmt.Printf("Erro: %v\n", err)
		} else {
			fmt.Println("Usuário excluído.")
		}
		return
	}

	uids := strings.Split(*uid, ",")
	for i := range uids {
		uids[i] = strings.TrimSpace(uids[i])
	}
	primaryUID := uids[0]

	if action == "add" {
		if primaryUID == "" || *name == "" || *cat == "" {
			fmt.Println("Erro: -uid, -name e -type são obrigatórios.")
			return
		}
		p := *pass
		if p == "" {
			p = generateRandomPassword(12)
			fmt.Printf("Senha gerada aleatoriamente: %s\n", p)
		}
		var aliases []string
		if len(uids) > 1 {
			aliases = uids[1:]
		}
		if err := backend.AddUser(primaryUID, *name, p, *cat, aliases, *shell); err != nil {
			fmt.Printf("Erro: %v\n", err)
			return
		}
		if *groups != "" {
			for _, g := range strings.Split(*groups, ",") {
				backend.AddToGroup("uid="+primaryUID+",ou=People,"+config.Cfg.LDAPBase, strings.TrimSpace(g))
			}
		}
		fmt.Println("Usuário criado com sucesso!")
		return
	}

	if action == "edit" {
		if primaryUID == "" {
			fmt.Println("Erro: -uid obrigatório.")
			return
		}
		dn := "uid=" + primaryUID + ",ou=People," + config.Cfg.LDAPBase
		u, err := backend.GetUser(dn)
		if err != nil {
			fmt.Printf("Erro: %v\n", err)
			return
		}
		upName, upShell, upAliases := u.CN, u.Shell, u.Aliases
		isGS := false
		var gSlice []string

		fs.Visit(func(f *flag.Flag) {
			switch f.Name {
			case "name":
				upName = *name
			case "shell":
				upShell = *shell
			case "uid":
				if len(uids) >= 1 {
					upAliases = aliasesFrom(uids)
				}
			case "groups":
				isGS = true
				if *groups != "" {
					for _, g := range strings.Split(*groups, ",") {
						gSlice = append(gSlice, strings.TrimSpace(g))
					}
				}
			}
		})
		if !isGS {
			if uG, _ := backend.GetUserGroups(dn); uG != nil {
				for _, g := range uG {
					gSlice = append(gSlice, g.CN)
				}
			}
		}
		if err := backend.UpdateUser(dn, upName, upShell, upAliases, gSlice, *pass); err != nil {
			fmt.Printf("Erro: %v\n", err)
		} else {
			fmt.Println("Usuário editado com sucesso!")
		}
		return
	}

	fmt.Println("Ação inválida. Use add, edit, list ou del.")
}

func aliasesFrom(uids []string) []string {
	if len(uids) > 1 {
		return uids[1:]
	}
	return nil
}

func handleGroupCommand(backend *ldapclient.LDAPBackend, action string, args []string) {
	fs := flag.NewFlagSet("group "+action, flag.ContinueOnError)
	cn := fs.String("cn", "", "CN do grupo")
	name := fs.String("name", "", "Nome")
	desc := fs.String("desc", "", "Descrição")
	gid := fs.String("gid", "", "GID")
	uid := fs.String("uid", "", "Membros (UIDs separados por vírgula)")

	if action == "list" {
		grps, _ := backend.ListGroups()
		fmt.Println("--- Grupos ---")
		for _, g := range grps {
			fmt.Printf("%-20s | GID: %-5s | %s\n", g.CN, g.GID, g.Desc)
		}
		return
	}

	if err := fs.Parse(args); err != nil {
		return
	}

	if action == "del" {
		if *cn == "" {
			fmt.Println("Erro: -cn obrigatório.")
			return
		}
		if err := backend.DeleteGroup(*cn); err != nil {
			fmt.Println("Erro:", err)
		} else {
			fmt.Println("Grupo deletado.")
		}
		return
	}

	mems := []string{}
	if *uid != "" {
		for _, m := range strings.Split(*uid, ",") {
			mems = append(mems, "uid="+strings.TrimSpace(m)+",ou=People,"+config.Cfg.LDAPBase)
		}
	}

	if action == "add" {
		if *cn == "" {
			fmt.Println("Erro: -cn obrigatório.")
			return
		}
		d, n := *desc, *name
		if n == "" {
			n = *cn
		}
		if d == "" {
			d = n
		}
		if err := backend.CreateGroup(*cn, *gid, d, mems); err != nil {
			fmt.Println("Erro:", err)
		} else {
			fmt.Println("Grupo criado!")
		}
		return
	}

	if action == "edit" {
		if *cn == "" {
			fmt.Println("Erro: -cn obrigatório.")
			return
		}
		dn := "cn=" + *cn + ",ou=Groups," + config.Cfg.LDAPBase
		gD, err := backend.GetGroup(*cn)
		if err != nil {
			fmt.Println("Erro:", err)
			return
		}
		upDesc := gD.Desc
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "desc" || f.Name == "name" {
				if *desc != "" {
					upDesc = *desc
				} else {
					upDesc = *name
				}
			}
		})
		backend.UpdateGroupInfo(dn, upDesc)
		if *uid != "" {
			backend.UpdateGroupMembers(dn, mems)
		}
		fmt.Println("Grupo atualizado!")
		return
	}

	fmt.Println("Ação inválida para group: add, edit, list, del.")
}

func handleKeyCommand(backend *ldapclient.LDAPBackend, action string, args []string) {
	fs := flag.NewFlagSet("key "+action, flag.ContinueOnError)
	uid := fs.String("uid", "", "UID do usuário")
	key := fs.String("key", "", "String da chave SSH")

	if err := fs.Parse(args); err != nil {
		return
	}

	if *uid == "" {
		fmt.Println("Erro: -uid obrigatório para chaves.")
		return
	}
	dn := "uid=" + *uid + ",ou=People," + config.Cfg.LDAPBase

	if action == "list" {
		keys, err := backend.GetSSHKeys(dn)
		if err != nil {
			fmt.Println("Erro:", err)
			return
		}
		for i, k := range keys {
			fmt.Printf("[%d] %s\n", i+1, k)
		}
		return
	}
	if action == "add" {
		if *key == "" {
			fmt.Println("Erro: -key obrigatório.")
			return
		}
		if err := backend.AddSSHKey(dn, *key); err != nil {
			fmt.Println("Erro:", err)
		} else {
			fmt.Println("Chave adicionada.")
		}
		return
	}
	if action == "del" {
		if *key == "" {
			fmt.Println("Erro: -key obrigatório.")
			return
		}
		if err := backend.RemoveSSHKey(dn, *key); err != nil {
			fmt.Println("Erro:", err)
		} else {
			fmt.Println("Chave removida.")
		}
		return
	}
	fmt.Println("Ação inválida para key. Use add, del, list.")
}
