# Guia do código do posfix — para quem está começando em Go

Este documento explica **todo** o projeto `posfix` passo a passo, supondo que você
quase não conhece Go. A ideia é que, ao terminar, você entenda cada arquivo, cada
decisão e consiga modificar o programa com segurança.

Sugestão de leitura: leia na ordem. As partes 1–2 dão a base; a partir da 3 a gente
mergulha no código de verdade.

---

## Parte 1 — O que o programa faz (visão de 30 segundos)

`posfix` é um programa de linha de comando que **renomeia arquivos** para um formato
limpo: tudo minúsculo, sem acentos, sem espaços nem símbolos, palavras separadas por
`_` (isso se chama *snake_case*). A extensão (`.pdf`, `.tar.gz`) é preservada.

```
"Café com Leite.txt"  ->  "cafe_com_leite.txt"
"Backup FINAL.TAR.GZ" ->  "backup_final.tar.gz"
```

O nome é um trocadilho com **POSIX** (o padrão de sistemas Unix).

---

## Parte 2 — Mini-curso de Go (só o que aparece neste projeto)

Não precisa decorar nada agora. Use esta parte como dicionário: quando bater dúvida
mais para frente, volte aqui.

### 2.1 Pacotes (`package`)

Todo arquivo `.go` começa declarando a que **pacote** ele pertence:

```go
package normalize
```

Um pacote é uma pasta com arquivos que compartilham o mesmo nome de pacote. É a
unidade de organização do Go. Existe um pacote especial chamado `main`: é o único que
vira um **programa executável** (os outros são bibliotecas, usadas por outros pacotes).

### 2.2 Módulo e `go.mod`

Um **módulo** é o projeto inteiro. Ele é descrito pelo arquivo `go.mod`:

```
module github.com/brunobreis/posfix   // o "nome" do projeto; vira o prefixo dos imports
go 1.25.0                              // versão mínima de Go exigida
require (                              // bibliotecas externas que usamos
	github.com/spf13/pflag v1.0.10
	golang.org/x/text v0.37.0
)
```

O `module ...` é importante: quando um arquivo importa
`github.com/brunobreis/posfix/internal/normalize`, o Go sabe que isso é a pasta
`internal/normalize` **deste** projeto.

### 2.3 Importar outros pacotes (`import`)

```go
import (
	"strings"                       // pacote da biblioteca padrão do Go
	"github.com/spf13/pflag"        // biblioteca externa (baixada)
	"github.com/brunobreis/posfix/internal/normalize"  // pacote nosso
)
```

Depois de importar `"strings"`, você usa as funções dele com o prefixo do nome:
`strings.ToLower(...)`, `strings.Join(...)` etc.

### 2.4 Funções

```go
func splitHiddenPrefix(name string) (prefix, rest string) {
	...
}
```

Lendo da esquerda para a direita:
- `func` — palavra-chave que inicia uma função.
- `splitHiddenPrefix` — nome.
- `(name string)` — recebe **um parâmetro** chamado `name`, do tipo `string`. (Repare:
  em Go o tipo vem **depois** do nome.)
- `(prefix, rest string)` — **retorna dois valores**, ambos `string`. Em Go uma função
  pode devolver vários valores de uma vez. Aqui eles até têm nome.

Nome começando com **letra maiúscula** (`Normalize`, `Run`, `Style`) é **público**:
outros pacotes podem usar. Minúscula (`tokenize`, `splitExt`) é **privado** ao pacote.
Isso é uma regra da linguagem, não convenção.

### 2.5 Variáveis: `var`, `const` e `:=`

```go
const fallback = "unnamed"          // constante: nunca muda
var version = "dev"                 // variável de pacote
errs := 0                           // dentro de função: "declara e atribui"
```

O `:=` é a forma curta de declarar uma variável **e** já dar um valor; o Go **deduz o
tipo** sozinho (aqui, `int`, porque `0` é inteiro). Só funciona dentro de funções.

### 2.6 `string`, `rune` e bytes

Uma `string` em Go é uma sequência de **bytes** em UTF-8. Para texto ASCII (a–z, 0–9),
1 caractere = 1 byte. Mas acentos ocupam mais de um byte. Quando você precisa pensar
em **caractere** (ponto de código Unicode), usa o tipo `rune` (que é um número, o
código do caractere). Você vai ver `rune` no tokenizador.

### 2.7 Slices (`[]string`) e maps (`map[...]...`)

- **Slice** é uma lista de tamanho variável. `[]string` = "lista de strings".
  `tokens[0]` é o primeiro elemento; `len(tokens)` é o tamanho.
- **Map** é um dicionário (chave → valor). `map[string][]string` = "de string para
  lista de strings". `claimed := make(map[string]bool)` cria um conjunto: `claimed[x]
  = true` marca, `claimed[x]` devolve `false` se nunca foi marcado.

`append(lista, item)` adiciona ao final e devolve a lista nova (sempre reatribua:
`lista = append(lista, item)`).

### 2.8 Struct (registro com campos)

```go
type Options struct {
	DryRun    bool
	Recursive bool
	All       bool
	Style     normalize.Style
}
```

Um `struct` agrupa vários valores. É como um formulário com campos. Você cria um assim:

```go
opts := renamer.Options{DryRun: true, Recursive: false}
```

`Snake struct{}` é um struct **vazio** (sem campos) — existe só para "ter um tipo".

### 2.9 Métodos

Um método é uma função "presa" a um tipo:

```go
func (Snake) Apply(tokens []string) string { ... }
```

O `(Snake)` antes do nome é o **receptor**: diz que `Apply` é um método do tipo
`Snake`. Você chama com `Snake{}.Apply(...)`. (Aqui o receptor nem tem nome porque o
método não precisa olhar para os dados do `Snake` — `Snake` não tem dados.)

### 2.10 Interface (o conceito mais importante deste projeto)

Uma **interface** descreve um *comportamento* (uma lista de métodos), sem dizer quem o
executa:

```go
type Style interface {
	Apply(tokens []string) string
}
```

Isso diz: "qualquer tipo que tenha um método `Apply([]string) string` **é** um
`Style`". Em Go isso é automático — `Snake` não precisa declarar "eu implemento
Style"; basta ter o método certo. Isso permite escrever código que aceita "qualquer
estilo" e, no futuro, criar `Kebab`, `Camel` etc. sem mudar o resto. Volte aqui quando
chegar na Parte 5.2.

### 2.11 Vários retornos e tratamento de erro

Go não tem exceções. Funções que podem falhar devolvem um **erro** como último valor:

```go
info, err := os.Lstat(target)
if err != nil {            // deu erro?
	// trata e segue
}
// se chegou aqui, info é válido
```

`nil` é o "vazio/nada" do Go. `err != nil` significa "houve erro". Esse padrão
`if err != nil` aparece o tempo todo.

O `_` (sublinhado) é o "lixo": usa-se para **descartar** um valor que não interessa:

```go
out, _, err := transform.String(...)   // ignora o segundo retorno
```

### 2.12 Ponteiros e `*` (o básico, sem susto)

Às vezes uma função devolve um **ponteiro** — um "endereço" para um valor, escrito
`*bool` ("ponteiro para bool"). Para ler o valor apontado, você coloca `*` na frente:

```go
dryRun := fs.BoolP("dry-run", "n", false, "...")  // dryRun é *bool (um endereço)
if *dryRun { ... }                                 // *dryRun é o bool de verdade
```

Por que ponteiro aqui? Porque a biblioteca de flags precisa **escrever** nessa variável
quando você passa `-n` na linha de comando. Ela guarda o endereço e mexe lá dentro.

### 2.13 `for ... range`

O único laço do Go é o `for`. A forma `range` percorre coleções:

```go
for i, name := range names { ... }   // i = índice, name = valor (slice)
for _, ce := range compositeExtensions { ... }  // só o valor (índice descartado)
for dir := range byDir { ... }       // num map, dá as CHAVES
```

### 2.14 Closure (função dentro de função)

Uma função pode ser guardada numa variável e "lembra" das variáveis ao redor:

```go
printUsage := func(w io.Writer) {
	fmt.Fprint(w, "...")
}
printUsage(stderr)   // chama
```

Isso é uma *closure*. Usamos para não repetir o texto de ajuda.

Com essa base, dá para ler tudo. Vamos à arquitetura.

---

## Parte 3 — A arquitetura em 3 camadas (a "big picture")

A ideia central do projeto é **separar a lógica pura dos efeitos colaterais**. Há três
camadas, cada uma numa pasta:

```
cmd/posfix/          → o programa em si: lê flags, chama as outras camadas, imprime
internal/normalize/  → LÓGICA PURA: recebe um nome (string), devolve o nome novo (string)
internal/renamer/    → DISCO: encontra arquivos, evita colisões, renomeia de fato
```

Por que isso importa?

- **`normalize` não toca em disco.** Ele só transforma texto. Por isso é trivial de
  testar: "dado este nome, espero este resultado" — sem criar arquivos. Toda a
  complicação (acentos, extensões, símbolos) mora aqui, isolada.
- **`renamer` é a única camada que mexe em arquivos.** Ela pergunta ao `normalize` qual
  é o nome novo e cuida do mundo real: arquivo oculto, colisão de nomes, não
  sobrescrever nada.
- **`cmd/posfix` é só a "cola".** Lê as opções da linha de comando e liga as peças.

A pasta especial **`internal/`** tem um significado no Go: pacotes dentro de `internal`
só podem ser importados por código **do mesmo projeto**. É uma forma de dizer "isto é
detalhe interno, não é API pública".

A regra de ouro para mexer no projeto: **a complexidade da transformação vai para
`normalize`; a complexidade do sistema de arquivos vai para `renamer`; `main` quase não
tem lógica.**

---

## Parte 4 — O fluxo de ponta a ponta

O que acontece quando você roda, por exemplo, `posfix -n ~/Downloads`:

1. **`main()`** (em `cmd/posfix/main.go`) é o ponto de entrada. Ele chama `run(...)`
   passando os argumentos e para onde escrever (`os.Stdout`, `os.Stderr`).
2. **`run`** interpreta as flags (`-n` vira "dry-run = true") e monta um `Options`.
   Decide o estilo: `Snake{}`. Chama `renamer.Run(alvos, opts, stdout, stderr)`.
3. **`renamer.Run`** descobre os arquivos (`collect` → `scanDir`), agrupados por pasta.
4. Para cada pasta, **`renameDir`** percorre os arquivos. Para cada um, pergunta:
   `normalize.Normalize(nome, estilo)` → recebe o nome novo.
5. **`normalize.Normalize`** roda o "pipeline": tira a extensão, remove acentos, quebra
   em palavras, junta com `_`, recoloca a extensão. Devolve a string.
6. De volta no `renameDir`, ele decide: se o nome não mudou, não faz nada; se o destino
   já existe, **pula com aviso**; senão, imprime `antigo -> novo` e renomeia (a menos
   que seja dry-run).
7. Os números de erro sobem de volta até `main`, que vira o **código de saída** do
   processo (0 = ok, 1 = houve erro).

Agora vamos ler cada arquivo com calma, na ordem do fluxo.

---

## Parte 5 — Arquivo por arquivo

### 5.1 `cmd/posfix/main.go` — o ponto de entrada

```go
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
```

- `main()` é a função que o Go executa quando o programa começa (só existe no
  `package main`).
- `os.Args` é a lista de argumentos da linha de comando; `os.Args[0]` é o nome do
  programa, então `os.Args[1:]` é "tudo depois do nome" (a fatia a partir do índice 1).
- `os.Stdout` / `os.Stderr` são as saídas padrão (texto normal) e de erro.
- `os.Exit(n)` encerra com um **código de saída** `n`. Por convenção Unix, `0` = sucesso.

**Por que `main` é tão curtinha?** Porque a lógica de verdade está em `run`, que recebe
para onde escrever em vez de usar `os.Stdout` direto. Isso deixa `run` **testável**: o
teste chama `run` mandando escrever num "buffer" de memória e depois confere o texto.
Esse é um padrão muito comum e idiomático em Go.

```go
func run(args []string, stdout, stderr io.Writer) int {
	fs := pflag.NewFlagSet("posfix", pflag.ContinueOnError)
	fs.SortFlags = false
	fs.SetOutput(stderr)
```

- `io.Writer` é uma **interface**: "qualquer coisa em que dá para escrever". Pode ser a
  tela (`os.Stdout`) ou um buffer de teste. `run` não sabe nem se importa qual é.
- `pflag` é a biblioteca de flags (estilo GNU: aceita `-n` e `--dry-run`, e agrupar
  `-rn`). `NewFlagSet` cria um conjunto de flags. `ContinueOnError` diz "se o usuário
  errar uma flag, não derruba o programa; me devolve um erro para eu tratar".

```go
	dryRun := fs.BoolP("dry-run", "n", false, "show what would happen, without renaming")
	recursive := fs.BoolP("recursive", "r", false, "descend into subdirectories")
	all := fs.BoolP("all", "a", false, "include hidden files ...")
	showHelp := fs.BoolP("help", "h", false, "show this help and exit")
	showVersion := fs.Bool("version", false, "show version and exit")
```

`BoolP` registra uma flag booleana com nome longo, **letra curta**, valor padrão e
descrição. Ela devolve um `*bool` (ponteiro): quando `fs.Parse` rodar, vai escrever
`true` ali se a flag aparecer. Por isso depois lemos com `*dryRun`. (`Bool`, sem o
`P`, é igual mas sem letra curta — `--version` não tem versão de uma letra.)

```go
	printUsage := func(w io.Writer) {
		fmt.Fprint(w, "posfix - normalize file names to snake_case ASCII\n\n")
		...
		fmt.Fprint(w, fs.FlagUsages())
	}
	fs.Usage = func() { printUsage(stderr) }
```

- `printUsage` é uma **closure** guardada numa variável: imprime o texto de ajuda em
  `w`. `fmt.Fprint(w, ...)` escreve em `w` (em vez de `fmt.Print`, que iria sempre para
  a tela). `fs.FlagUsages()` gera automaticamente a lista de flags formatada.
- `fs.Usage = ...` define o que a biblioteca chama quando precisa mostrar ajuda sozinha.

```go
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "posfix: %v\n", err)
		printUsage(stderr)
		return 2
	}
```

`fs.Parse(args)` lê os argumentos e preenche as flags. Repare na forma
`if err := ...; err != nil`: declara `err` **e** testa, tudo na mesma linha (o `err` só
existe dentro do `if`). Se a flag for inválida, escrevemos o erro e a ajuda em `stderr`
e retornamos **2** (código de saída para "erro de uso").

`fmt.Fprintf` é como `printf` do C: `%v` é "imprima o valor em formato padrão", `\n` é
quebra de linha.

```go
	if *showHelp {
		printUsage(stdout)
		return 0
	}
	if *showVersion {
		fmt.Fprintf(stdout, "posfix %s\n", version)
		return 0
	}
```

Se o usuário pediu `--help` ou `--version`, atendemos e saímos com `0` (sucesso). Note
que aqui a ajuda vai para `stdout` (o usuário pediu de propósito), enquanto no caso de
erro foi para `stderr`. `%s` é "insira esta string".

```go
	opts := renamer.Options{
		DryRun:    *dryRun,
		Recursive: *recursive,
		All:       *all,
		Style:     normalize.Snake{},
	}
	if renamer.Run(fs.Args(), opts, stdout, stderr) > 0 {
		return 1
	}
	return 0
}
```

- Monta o `Options` (um struct) a partir das flags. `normalize.Snake{}` cria um valor
  do estilo snake_case — é o único estilo por enquanto.
- `fs.Args()` são os argumentos **que não eram flags** (os alvos: arquivos/pastas).
- `renamer.Run(...)` devolve o número de erros. Se for `> 0`, o programa sai com `1`;
  senão, `0`.

Sobre o `var version = "dev"` lá no topo: é a versão. O comentário mostra como dar uma
versão real **na hora de compilar** (`-ldflags "-X main.version=v1.0.0"`), sem mudar o
código. Detalhe avançado; pode ignorar por ora.

### 5.2 `internal/normalize/style.go` — a interface `Style` e o `Snake`

Este arquivo é curto mas é o coração do design extensível.

```go
type Style interface {
	Apply(tokens []string) string
}
```

Define o **comportamento** "saber montar o nome final a partir de uma lista de
palavras (tokens)". Qualquer tipo com um método `Apply([]string) string` é um `Style`.

```go
type Snake struct{}

func (Snake) Apply(tokens []string) string {
	return strings.Join(tokens, "_")
}
```

`Snake` é um struct vazio. Seu método `Apply` recebe as palavras já em minúsculo e só
as **junta com `_`**: `["cafe","com","leite"]` → `"cafe_com_leite"`. (`strings.Join`
junta os elementos de um slice usando um separador.)

**Por que tanta cerimônia para só juntar com `_`?** Porque amanhã você pode querer
`kebab-case` (junta com `-`) ou `camelCase` (Café → cafeComLeite). Cada um seria um
novo tipo com seu próprio `Apply` — e **nada mais no projeto muda**, porque todo o
resto fala com a interface `Style`, não com o `Snake` diretamente. Isso é o "padrão
strategy". É também por isso que a regra de **minúsculas** está no pipeline (vale para
todos) mas a de **juntar** está no estilo (cada um junta do seu jeito).

### 5.3 `internal/normalize/normalize.go` — o pipeline puro

Primeiro, três coisas declaradas no nível do pacote:

```go
const fallback = "unnamed"
```
Nome usado quando, depois de remover tudo, não sobra nenhuma palavra (ex.: um arquivo
chamado `@#$.txt`).

```go
var compositeExtensions = []string{
	".tar.gz", ".tar.bz2", ".tar.xz", ".tar.zst", ".tar.lz", ".tar.lzma", ".tar.z",
}
```
Lista de extensões "compostas" (duas partes). Precisamos dela porque, em `backup.tar.gz`,
a extensão certa é `.tar.gz` inteira — não só `.gz`. **Para suportar mais formatos,
basta adicionar nesta lista.**

```go
var transliterator = transform.Chain(
	norm.NFKD,
	runes.Remove(runes.In(unicode.Mn)),
	norm.NFC,
)
```
Esse é o "removedor de acentos", montado com a biblioteca `golang.org/x/text`. A ideia
(sem precisar dominar Unicode):
- `NFKD` **separa** uma letra acentuada em "letra base" + "marca de acento" (o `é` vira
  `e` + `´`). Também simplifica formas compatíveis (o `ª` vira `a`).
- `runes.Remove(runes.In(unicode.Mn))` **joga fora** essas marcas de acento (a categoria
  Unicode `Mn` = "Mark, nonspacing").
- `NFC` recompõe o que sobrou.

Resultado: `café` → `cafe`, `programação` → `programacao`. Letras sem decomposição
(como `ø`, `ł`) **não** são convertidas por esse caminho — elas seguem adiante e acabam
descartadas na hora de quebrar em palavras (viram "separador"). Isso é uma limitação
conhecida e documentada, não um bug.

Agora a função principal:

```go
func Normalize(name string, style Style) string {
	hidden, rest := splitHiddenPrefix(name)
	base, ext := splitExt(rest)

	tokens := tokenize(transliterate(base))
	out := style.Apply(tokens)
	if out == "" {
		out = fallback
	}

	return hidden + out + strings.ToLower(ext)
}
```

Repare que `Normalize` recebe um `Style` (a interface!) e o usa em `style.Apply(...)` —
ela não sabe se é `Snake` ou outro. Lendo o pipeline:

1. `splitHiddenPrefix` separa um ponto inicial (arquivos ocultos).
2. `splitExt` separa o miolo (`base`) da extensão (`ext`).
3. `transliterate(base)` tira os acentos; `tokenize(...)` quebra em palavras.
4. `style.Apply(tokens)` monta o miolo final.
5. Se sobrou vazio, usa `"unnamed"`.
6. Junta tudo de volta: `hidden + miolo + extensão em minúsculas`. (Em Go, `+` concatena
   strings.)

Vamos às funções auxiliares.

```go
func splitHiddenPrefix(name string) (prefix, rest string) {
	if strings.HasPrefix(name, ".") {
		return ".", name[1:]
	}
	return "", name
}
```
Se o nome começa com `.` (ex.: `.bashrc`), devolve `(".", "bashrc")`. Senão, devolve
`("", name)`. Isso evita que o ponto inicial seja tratado como separador e o arquivo
oculto perca o ponto. `name[1:]` é "a string a partir do byte 1" (seguro aqui porque
`.` é 1 byte).

```go
func splitExt(name string) (base, ext string) {
	lower := strings.ToLower(name)
	for _, ce := range compositeExtensions {
		if strings.HasSuffix(lower, ce) {
			cut := len(name) - len(ce)
			return name[:cut], name[cut:]
		}
	}
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[:i], name[i:]
	}
	return name, ""
}
```
A lógica de extensão, em três tentativas:
1. Primeiro testa as **compostas**: passa por cada `ce` da lista e, se o nome (em
   minúsculas, para ignorar maiúsculas/minúsculas) **termina com** ela
   (`strings.HasSuffix`), corta ali. `name[:cut]` é "do começo até `cut`"; `name[cut:]`
   é "de `cut` até o fim". Ex.: `Backup.TAR.GZ` → `("Backup", ".TAR.GZ")`.
2. Senão, acha o **último ponto** (`strings.LastIndex`). Se existe (`i >= 0`), corta ali.
   Ex.: `foo.bar.txt` → `("foo.bar", ".txt")` (só o último ponto conta).
3. Senão (sem ponto nenhum), não há extensão: `(name, "")`.

A extensão é devolvida com as maiúsculas originais **de propósito**: quem chama
(`Normalize`) é que faz `strings.ToLower(ext)`.

```go
func transliterate(s string) string {
	out, _, err := transform.String(transliterator, s)
	if err != nil {
		return s
	}
	return out
}
```
Aplica o removedor de acentos. `transform.String` devolve três coisas (resultado,
quantos bytes processou, erro); só queremos o resultado, então descartamos o do meio
com `_`. Se der erro (raríssimo), devolve a string original sem quebrar.

```go
func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !isWordRune(r)
	})
}

func isWordRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}
```
Esta é a quebra em palavras:
- Primeiro passa tudo para minúsculo (`strings.ToLower`).
- `strings.FieldsFunc(texto, função)` quebra o texto sempre que a `função` disser que
  um caractere é **separador**. A função recebe um `rune` (caractere) e devolve `true`
  se for separador.
- Aqui o separador é "tudo que **não** é letra a–z ou dígito 0–9". `isWordRune` diz o
  que **faz parte** de uma palavra; o `!` (não) inverte. Então espaço, `@`, `-`, `!`,
  acento que sobrou… tudo vira separador.
- Bônus: `FieldsFunc` **já descarta pedaços vazios**, então `"---draft---"` vira
  `["draft"]` sem strings vazias no meio.

Exemplo completo passo a passo, com `"Café - Vol 1!!!.PDF"`:
1. `splitHiddenPrefix` → `("", "Café - Vol 1!!!.PDF")`
2. `splitExt` → base `"Café - Vol 1!!!"`, ext `".PDF"`
3. `transliterate` → `"Cafe - Vol 1!!!"`
4. `tokenize` → `["cafe", "vol", "1"]`
5. `Snake.Apply` → `"cafe_vol_1"`
6. junta → `"cafe_vol_1" + ".pdf"` = **`"cafe_vol_1.pdf"`**

### 5.4 `internal/renamer/renamer.go` — a camada de disco

Esta é a parte que lida com o mundo real (arquivos, pastas, colisões). Começa pelo
struct de opções:

```go
type Options struct {
	DryRun    bool
	Recursive bool
	All       bool
	Style     normalize.Style
}
```
Repare que `Style` aqui é a **interface** — `renamer` aceita qualquer estilo.

```go
func Run(targets []string, opts Options, stdout, stderr io.Writer) int {
	if len(targets) == 0 {
		targets = []string{"."}
	}
	if opts.Style == nil {
		opts.Style = normalize.Snake{}
	}

	byDir, errs := collect(targets, opts, stderr)

	dirs := make([]string, 0, len(byDir))
	for dir := range byDir {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		errs += renameDir(dir, byDir[dir], opts, stdout, stderr)
	}
	return errs
}
```
Passo a passo:
- Sem alvos, usa `"."` (a pasta atual). `[]string{"."}` é um slice com um elemento.
- Se ninguém definiu estilo (`== nil`), assume `Snake`. (Uma interface pode estar
  "vazia": vale `nil`.)
- `collect` devolve um **map** `pasta → [arquivos]` e a contagem de erros.
- As próximas linhas pegam as chaves do map (as pastas) e **ordenam**. Por quê? Porque a
  ordem de um map em Go é aleatória; ordenar deixa a saída **previsível** (e os testes
  estáveis). `make([]string, 0, len(byDir))` cria um slice vazio já com espaço reservado
  (otimização pequena).
- Para cada pasta, chama `renameDir` e **soma** os erros (`+=`). Retorna o total.

```go
func collect(targets []string, opts Options, stderr io.Writer) (map[string][]string, int) {
	byDir := make(map[string][]string)
	errs := 0

	for _, target := range targets {
		info, err := os.Lstat(target)
		if err != nil {
			fmt.Fprintf(stderr, "posfix: %v\n", err)
			errs++
			continue
		}
		if info.IsDir() {
			errs += scanDir(target, opts, byDir, stderr)
			continue
		}
		if info.Mode().IsRegular() {
			byDir[filepath.Dir(target)] = append(byDir[filepath.Dir(target)], filepath.Base(target))
		}
	}
	return byDir, errs
}
```
- `os.Lstat(target)` pergunta ao sistema "o que é isto?" sem seguir links simbólicos.
  Devolve `info` (metadados) e `err`. Se deu erro (não existe, sem permissão), avisa,
  conta e `continue` (pula para o próximo alvo do laço).
- `info.IsDir()` → é pasta: manda para `scanDir`.
- `info.Mode().IsRegular()` → é arquivo "normal" (não é pasta, não é dispositivo): então
  guarda no map. `filepath.Dir(target)` é a pasta do caminho; `filepath.Base(target)` é
  só o nome do arquivo. A linha
  `byDir[chave] = append(byDir[chave], valor)` é o jeito idiomático de "adicionar item à
  lista guardada nessa chave do map".
- Um arquivo **passado explicitamente** entra mesmo se for oculto — diferente da
  varredura de pasta, que filtra ocultos.

```go
func scanDir(dir string, opts Options, byDir map[string][]string, stderr io.Writer) int {
	if !opts.Recursive {
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintf(stderr, "posfix: %v\n", err)
			return 1
		}
		for _, e := range entries {
			if keepEntry(e, opts.All) {
				byDir[dir] = append(byDir[dir], e.Name())
			}
		}
		return 0
	}
	...
```
Dois modos:
- **Sem `-r` (não recursivo):** `os.ReadDir(dir)` lista o conteúdo direto da pasta. Para
  cada item, `keepEntry` decide se entra (ver abaixo).
- **Com `-r` (recursivo):** usa `filepath.WalkDir`, que **desce** por todas as subpastas:

```go
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(stderr, "posfix: %v\n", err)
			errs++
			return nil
		}
		if d.IsDir() {
			if path != dir && isHidden(d.Name()) && !opts.All {
				return filepath.SkipDir
			}
			return nil
		}
		if keepEntry(d, opts.All) {
			byDir[filepath.Dir(path)] = append(byDir[filepath.Dir(path)], d.Name())
		}
		return nil
	})
```
`WalkDir` recebe a pasta e uma **função de callback** que ele chama para cada
arquivo/pasta encontrado. Dentro dela:
- Se houve erro naquele item, avisa e segue (`return nil` = "continue o passeio").
- Se for **pasta**: não fazemos nada com ela (não renomeamos pastas), mas se for uma
  subpasta **oculta** e não veio `--all`, devolvemos `filepath.SkipDir` — isso diz ao
  `WalkDir` "nem entre aí". (O `path != dir` garante que não pulamos a própria pasta
  raiz que o usuário pediu.)
- Se for **arquivo**, `keepEntry` decide e adiciona no map sob a pasta dele.

```go
func keepEntry(e fs.DirEntry, all bool) bool {
	if !e.Type().IsRegular() {
		return false
	}
	return all || !isHidden(e.Name())
}
```
"Vale a pena renomear este item?" Só se for arquivo normal **e** (veio `--all` **ou**
não é oculto). O `||` é "ou".

Agora o coração da segurança — `renameDir`:

```go
func renameDir(dir string, names []string, opts Options, stdout, stderr io.Writer) int {
	sort.Strings(names)

	errs := 0
	claimed := make(map[string]bool)
	last := ""

	for i, name := range names {
		if i > 0 && name == last {
			continue
		}
		last = name

		newName := normalize.Normalize(name, opts.Style)
		if newName == name {
			claimed[name] = true
			continue
		}

		dest := filepath.Join(dir, newName)
		if claimed[newName] || pathExists(dest) {
			fmt.Fprintf(stderr, "posfix: skipped %q: target %q already exists\n", name, newName)
			continue
		}

		fmt.Fprintf(stdout, "%s%s  ->  %s\n", dryRunPrefix(opts.DryRun), name, newName)
		if !opts.DryRun {
			if err := os.Rename(filepath.Join(dir, name), dest); err != nil {
				fmt.Fprintf(stderr, "posfix: %v\n", err)
				errs++
				continue
			}
		}
		claimed[newName] = true
	}
	return errs
}
```
Esta função processa **uma pasta**. Linha a linha:
- `sort.Strings(names)` ordena os arquivos — de novo, para resultado previsível e para
  a regra de colisão ser determinística (quem vem primeiro "ganha" o nome).
- `claimed` é um **conjunto** dos nomes de destino já reservados nesta rodada. Serve
  para detectar duas origens que viram o mesmo destino.
- `last` + o `if i > 0 && name == last` ignoram um mesmo arquivo listado duas vezes.
- `newName := normalize.Normalize(name, opts.Style)` — **aqui as duas camadas se
  encontram**: a camada de disco pergunta à camada pura qual é o nome novo.
- Se `newName == name`, o arquivo **já está normalizado**: não imprime nada, mas
  **reserva** o nome em `claimed` (para um irmão não tentar tomar esse nome).
- `dest := filepath.Join(dir, newName)` monta o caminho completo do destino
  (`filepath.Join` junta pasta + nome com a barra certa do sistema).
- **A trava de segurança:** se `newName` já foi reservado nesta rodada (`claimed[newName]`)
  **ou** já existe no disco (`pathExists(dest)`), então **pula com aviso** no `stderr`.
  Isso é o "nunca sobrescrever". `%q` imprime entre aspas.
- Senão: imprime `antigo -> novo` no `stdout` (com prefixo `[dry-run] ` se for simulação)
  e, **se não for dry-run**, renomeia de fato com `os.Rename`. Se o `os.Rename` falhar,
  conta um erro de verdade. Por fim, reserva o nome em `claimed`.

Detalhe importante: em dry-run a função **imprime mas não chama `os.Rename`** — por isso
a simulação não toca no disco.

```go
func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func isHidden(name string) bool {
	return strings.HasPrefix(name, ".")
}

func dryRunPrefix(dryRun bool) string {
	if dryRun {
		return "[dry-run] "
	}
	return ""
}
```
Três ajudantes pequenos:
- `pathExists`: tenta obter info do caminho; se **não** deu erro (`err == nil`), existe.
  Usa `Lstat` para que até um link quebrado conte como "existe" (não vamos pisar nele).
- `isHidden`: começa com `.`?
- `dryRunPrefix`: devolve o rótulo `[dry-run] ` ou vazio.

---

## Parte 6 — Os testes

Go tem testes embutidos. Arquivos terminados em `_test.go` contêm testes; funções
`func TestXxx(t *testing.T)` são executadas pelo comando `go test`.

O projeto usa **testes orientados a tabela** (*table-driven tests*) — o estilo
idiomático em Go. Em vez de escrever um teste por caso, você faz uma **lista de casos**
e um laço que roda todos. Exemplo (de `internal/normalize/normalize_test.go`):

```go
tests := []struct {
	name string
	in   string
	want string
}{
	{"latin accents", "café com leite.txt", "cafe_com_leite.txt"},
	{"composite extension", "meu backup.tar.gz", "meu_backup.tar.gz"},
	// ... mais casos
}

for _, tt := range tests {
	t.Run(tt.name, func(t *testing.T) {
		if got := Normalize(tt.in, Snake{}); got != tt.want {
			t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	})
}
```
- `[]struct{...}{...}` é um slice de structs anônimos — cada item é um caso com entrada
  (`in`) e resultado esperado (`want`).
- O laço roda cada caso; `t.Run(nome, ...)` cria um **subteste** com nome próprio (fica
  fácil ver qual falhou).
- `t.Errorf(...)` marca falha se o resultado (`got`) não bater com o esperado (`want`).

**Por que isso é tão bom aqui:** adicionar um novo caso é só acrescentar uma linha na
tabela. Por isso a camada `normalize`, sendo pura, consegue cobrir dezenas de regras
sem nenhum arquivo de verdade.

Já os testes de `renamer` precisam de arquivos reais — e usam `t.TempDir()`, que cria
uma pasta temporária descartável só para aquele teste. Eles verificam coisas como: o
dry-run não mexe no disco, ocultos são ignorados sem `--all`, colisão pula um arquivo
com aviso, nunca sobrescreve um arquivo existente, etc.

E os testes de `cmd/posfix` aproveitam aquele truque de `run(args, stdout, stderr)`:
chamam `run` com buffers de memória e conferem o texto e o código de saída — sem
realmente rodar o programa no terminal.

---

## Parte 7 — Como rodar, compilar e testar

Todos os comandos rodam na raiz do projeto (`/home/brunobreis/Documents/posfix`).

```sh
# Rodar sem instalar (passando flags e alvos depois do --):
go run ./cmd/posfix -- -n ~/Downloads

# Compilar um executável chamado "posfix" na pasta atual:
go build -o posfix ./cmd/posfix
./posfix --help

# Instalar no seu sistema (vai para ~/go/bin):
go install ./cmd/posfix

# Rodar TODOS os testes:
go test ./...

# Rodar os testes de UM pacote só:
go test ./internal/normalize/

# Rodar UM teste específico (por nome, com regex):
go test ./internal/normalize/ -run TestNormalize_Snake

# Rodar UM caso específico da tabela (o nome vira subteste, espaços viram _):
go test ./internal/normalize/ -run TestNormalize_Snake/dotfile_is_unchanged

# Ver os testes passando um a um (verboso):
go test -v ./...

# Conferir formatação (não deve listar nada), checagem estática e lint:
gofmt -l .
go vet ./...
```

Observações úteis:
- `./...` significa "este pacote e todos os subpacotes" — é o "tudo".
- No `go run ./cmd/posfix`, use `--` antes das flags do **seu** programa para o Go não
  confundir com flags dele.
- `gofmt` é o formatador oficial; em Go a formatação é padronizada (sem discussão de
  estilo). Rodar `gofmt -w .` formata os arquivos no lugar.

---

## Parte 8 — Glossário relâmpago

- **pacote**: pasta de arquivos `.go` com o mesmo `package`. `main` é o executável.
- **módulo**: o projeto todo, descrito por `go.mod`.
- **slice** `[]T`: lista de tamanho variável de elementos do tipo `T`.
- **map** `map[K]V`: dicionário de chave `K` para valor `V`.
- **struct**: registro com campos.
- **interface**: lista de métodos; "qualquer tipo que os tenha serve". Implementada
  automaticamente.
- **método**: função presa a um tipo (`func (T) Nome(...)`).
- **rune**: um caractere Unicode (um número).
- **erro**: último valor de retorno; `if err != nil` trata.
- **ponteiro** `*T`: endereço de um valor; `*p` lê o valor.
- **`_`**: descarta um valor.
- **`nil`**: vazio/ausente.
- **closure**: função guardada em variável que lembra o contexto ao redor.
- **`io.Writer`**: "algo onde dá para escrever" (tela, arquivo, buffer de teste).
- **table-driven test**: testes como uma lista de casos percorrida por um laço.

---

### Onde olhar quando for mexer em algo

| Quero mudar... | Mexo em... |
|---|---|
| Como o nome é transformado (regras de texto) | `internal/normalize/normalize.go` |
| Suportar nova extensão composta (ex.: `.tar.lz4`) | a lista `compositeExtensions` |
| Um novo estilo (kebab, camel...) | novo tipo com `Apply` em `internal/normalize/` |
| Como arquivos são descobertos / colisão / dry-run | `internal/renamer/renamer.go` |
| Flags da linha de comando, ajuda, código de saída | `cmd/posfix/main.go` |

Pronto. Se você leu até aqui, já consegue navegar e alterar o `posfix` com segurança.
A melhor forma de fixar: abra cada arquivo ao lado da seção correspondente da Parte 5
e leia os dois em paralelo.
