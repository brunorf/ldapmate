# LDAPMate

LDAPMate é um assistente leve, distribuído como um único binário em Go, para gerenciar de forma ágil e segura o seu diretório LDAP. Em vez de depender de pesadas interfaces Java/PHP ou de scripts Bash complexos, o LDAPMate oferece **duas formas otimizadas** de gerenciar contas:

1. **Um Shell Interativo de Linha de Comando (CLI):** Ideal para sysadmins operando diretamente de terminais via SSH. Conta com histórico inteligente, autocompletar e não expõe suas senhas (sempre solicita de forma segura usando o `x/term`).
2. **Um Painel Web Integrado:** Uma interface gráfica limpa, pronta para o time de Suporte (Help Desk) resetar senhas ou alterar chaves SSH, sem tocar diretamente no servidor de linha de comando.

## Por Que o LDAPMate?
- **Um Único Arquivo (`//go:embed`):** Tanto as páginas HTML do servidor web quanto o mapeamento da rede ficam costurados diretamente dentro do executável compilado. É fazer o build local e jogar o executável final em qualquer servidor Linux que você tenha!
- **Auto-Alocação de UIDs:** Não precisa se preocupar em ficar checando "qual é o próximo uid vago" ao criar usuários. O sistema aceita esquemas de `faixas de IDs` por departamento (Ex: Estudantes na faixa 30000+, Professores na faixa 20000+).
- **Agnóstico:** Totalmente desamarrado de strings fixas; crie as faixas e o nome da sua base e ele só vai se conectar de forma nativa ao seu OpenLDAP.

## Configuração

Antes de compilar seu próprio executável do LDAPMate, acesse o arquivo de configuração e aponte para a base e portas da sua instituição. Por ser `embedded`, isso previne que o arquivo `config.json` fique "caindo/perdido" na hora do *deploy* no servidor final.

No arquivo `internal/config/config.json`:

```json
{
  "ldap_host": "ldap.sua.empresa.com",
  "ldap_base": "dc=sua,dc=empresa,dc=com",
  "admin_dn": "cn=admin,dc=sua,dc=empresa,dc=com",
  "secret_key": "senha-forte-para-os-cookies-web-aqui",
  "category_ranges": {
    "admin": [10000, 10999],
    "aluno": [30000, 89999],
    "guest": [90000, 99999]
  },
  "default_category": "aluno",
  "server_port": ":8080"
}
```

## Compilação (Build)

Certifique-se de ter o Go 1.25+ instalado e simplesmente execute:

```bash
go fmt ./...
go build -o ldapmate ./cmd/web
```
Isso vai gerar o arquivo `./ldapmate`.

## Uso

```bash
# Ajuda Geração e Subcomandos
./ldapmate -h

# Iniciar o Servidor Web (painel gráfico)
./ldapmate serve

# Iniciar o Shell Interativo (Você dita a senha do Administrador uma única vez)
./ldapmate shell 
```

### Exemplos no Shell Interativo (`ldap>`)

```bash
# Listar todos os usuários
user list

# Criar usuário novo definindo grupos e um nome longo
user add -uid joaopedro -name "João Pedro Gomes" -type aluno -groups acesso_ssh,estudantes

# Listar Grupos
group list
```

## Licença
Distribuído de forma livre sob a premissa Open-Source (MIT License). Caso isso salve algumas horas do seu dia-a-dia de Sysadmin, seja bem-vindo para fazer um PR!
