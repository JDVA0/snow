# <img src="docs/blizzard.svg" width="36" height="36" alt="Blizzard icon" style="vertical-align: middle;"> Blizzard 0.2

[![Go Report Card](https://goreportcard.com/badge/github.com/JDVA0/blizzard)](https://goreportcard.com/report/github.com/JDVA0/blizzard)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[Español](#español) | [English](#english)

---

## Español

Blizzard 0.2 es un lenguaje simple, directo y conciso para crear APIs web REST, herramientas de línea de comandos y automatizaciones.

> Proyecto creado por **JDVA0**.
> Hecho por diversión y experimentación.
> Libre y de código abierto.
>
> ⓘ Nota: este proyecto fue desarrollado con asistencia de inteligencia artificial.

> **Proyecto experimental de IA:** Blizzard está construido como una
> investigación abierta asistida por inteligencia artificial. La sintaxis, la
> API, el formato de paquetes y la estructura interna pueden cambiar de forma
> brusca entre versiones. No se recomienda usarlo todavía como dependencia
> estable en producción; fija una versión y conserva copias de tus fuentes.

### Estado 0.2

La referencia de esta versión está en [docs/blizzard-0.2.md](docs/blizzard-0.2.md).
El comando `blizzard` y el módulo Go `github.com/JDVA0/blizzard` son la interfaz
oficial de esta versión.

### Migración Snow → Blizzard

La versión 0.2 cambia oficialmente el nombre del proyecto y de sus artefactos:

| Antes | Ahora |
|---|---|
| `Snow` | `Blizzard` |
| `snowman` | `blizzard` |
| `.snow` | `.blizz` |
| `.snowpkg` | `.blizzpkg` |
| `snow.toml` | `blizzard.toml` |
| `snow.lock` | `blizzard.lock` |
| `snow-lsp` | `blizzard-lsp` |
| `github.com/JDVA0/snow` | `github.com/JDVA0/blizzard` |

Los ejemplos, paquetes, tests, snippets, gramática y documentación del repositorio
ya usan la nomenclatura nueva. Los proyectos antiguos deben renombrar sus archivos,
actualizar imports y regenerar su lockfile antes de ejecutar `blizzard run`.
La sintaxis antigua por indentación sigue leyéndose durante la transición, pero
los archivos nuevos deben usar `{}`, `let` e `import`.

### Herramientas 0.2

```text
blizzard fmt [-w] [--modern] archivo.blizz
blizzard check archivo.blizz
blizzard test [ruta]
blizzard run [archivo.blizz]
blizzard build [archivo.blizz]
blizzard init nombre
```

`build` analiza, comprueba y compila sin ejecutar el programa. Los diagnósticos
estructurados usan códigos estables: `B001` sintaxis, `B002` nombre indefinido,
`B003` módulo ausente, `B004` tipos, `B005` runtime y `B100` sintaxis heredada.
El checker infiere tipos ciertos de literales, variables, listas, diccionarios y
operaciones; una reasignación incompatible se detecta antes de ejecutar.

### Roadmap experimental

- Separar progresivamente lexer, parser, compilador, runtime y biblioteca estándar en paquetes internos reales.
- Retirar gradualmente la sintaxis heredada por indentación después de una fase de migración.
- Añadir inferencia estática más profunda sin convertir Blizzard en un lenguaje de tipos obligatorios.
- Formalizar resolución semántica de versiones, checksums obligatorios y caché global del gestor de paquetes.
- Ejecutar compatibilidad continua en Linux, macOS y Windows.
- Ampliar fuzzing de lexer/parser y publicar la extensión `blizzard-language` 0.2.0.

El gestor ya admite restricciones básicas (`1.2.3`, `^1.2.3`, `~1.2.3`, `>=1.2.3`)
y el lockfile conserva SHA-256. La caché global y la validación estricta de
checksums quedan activadas progresivamente para no romper instalaciones 0.2.

### Características

### Sintaxis moderna

Blizzard acepta una sintaxis basada en bloques con llaves e inferencia de tipos:

```blizzard
import json;

let port = 8080;

fn greet(name) {
    if (name != "") {
        print(json.stringify(name));
    } else {
        print("anonymous");
    }
}

greet("Ada");
```

Las declaraciones usan `let` o `const`, los módulos se cargan con `import` y no
se necesitan anotaciones de tipo. La forma anterior con `:` e indentación sigue
siendo aceptada durante la migración de proyectos existentes; `blizzard fmt`
conserva el estilo de entrada y formatea los archivos modernos con llaves.

- **Sintaxis por bloques:** La sintaxis moderna usa `{}` y `;`; la sintaxis por indentación permanece disponible durante la migración.
- **Servidor HTTP integrado (`import api`):** Rutas GET, POST, PUT, DELETE, PATCH, soporte JSON nativo, CORS y middlewares en pocas líneas.
- **Cliente HTTP integrado (`import http`):** Peticiones GET, POST, PUT, DELETE a APIs externas con parseo JSON automático sin depender de curl.
- **Base de datos clave-valor embebida (`import db`):** Almacén JSON estructurado y persistente en disco con escritura atómica.
- **Interpolación F-Strings y Operador Elvis (`??`):** `f"Hola {usuario}"`, cadenas multilínea con `"""` y valor por defecto para nulos.
- **Encadenado seguro (`?.`, `?[]`):** `data?.user?.profile?.name` recorre JSON anidado sin lanzar error y `??=` asigna solo si la variable es `nil`.
- **Cortes (`[i:j]`):** `lista[1:3]`, `texto[:2]`, índices negativos y slicing seguro `d?.a?.b?[1:]`.
- **`match` / `case`:** Comparación por casos en un solo bloque con múltiples valores por caso y comodín `case _`.
- **`for k, v in dict`:** Itera claves, valores o pares de diccionarios y listas en una sola línea.
- **`not in` y literales en base 2/8/16:** `if x not in lista`, `0b1010`, `0o17`, `0xFF`.
- **Stack traces:** Los errores no capturados muestran la pila de llamadas con archivo, línea y columna.
- **Linting:** `blizzard check archivo.blizz` detecta nombres no definidos, variables sin usar, código inalcanzable y módulos desconocidos sin ejecutar el programa.
- **Acceso a sistema y archivos (`import sys`, `import fs`):** Ejecución de comandos del sistema (`sys.sh`), lectura de entorno, control de procesos y operaciones de archivos.
- **Herramientas de consola (`import cli`):** Análisis de banderas/argumentos, tablas alineadas y cajas de texto formateadas.
- **Entorno y CSV (`import env`, `import csv`):** Variables de entorno, archivos `.env`, parseo y escritura de CSV.
- **Errores como valores:** `try` / `catch` y `fail(valor)`. Sin clases ni jerarquías. `nil` es ausencia, no un error.
- **Inferencia:** `let nombres = ["Julian", "Ana"]` y `fn sumar(a, b) { ... }` evitan anotaciones en el código nuevo.
- **Formato:** `blizzard fmt [-w] archivo.blizz` normaliza la sintaxis moderna con llaves y conserva archivos heredados.
- **REPL interactivo avanzado:** Historial persistente en `~/.blizz_history`, comandos (`help`, `.exit`, `.history`) y colores por tipo de dato.
- **Visibilidad de módulos (`pub` / `priv`):** Encapsulación por archivo. `pub` exporta, `priv` oculta del `import`, sin modificador = `pub` por defecto.
- **Tipos graduales:** Las anotaciones son opcionales. `age: int = 18`, `names: str[] = ["Ada"]` y las firmas de funciones se validan en ejecución sin restringir el código no anotado.
- **Paquetes locales:** Un proyecto con `blizzard.toml` y `src/` puede importar sus módulos por nombre: `import mi_app.utils`.
- **Pruebas de scripts:** `blizzard test` ejecuta todos los archivos `*_test.blizz` y reporta cada resultado.
- **Filtro `where` en `for`:** `for x in lista where cond:` filtra elementos directamente sin un `if` anidado.
- **Gestión automática de recursos (`with`):** `with abrir_recurso() as r:` libera el recurso (llama `r.close()`) automáticamente al salir del bloque.
- **Composición:** Los bloques `with`, filtros `where` y `import` de módulos con `pub/priv` funcionan juntos de forma natural.

### Funciones recientes

- **Constantes con `const`:** `const PORT = 8080` marca un valor como inmutable y una reasignación produce un error en tiempo de ejecución.
- **Bloque `always`:** puede seguir a `try` / `catch` y se ejecuta tanto después del camino normal como después del bloque de captura.
- **Expresión ternaria:** `estado = "open" if activo else "closed"` devuelve uno de dos valores sin un bloque `if`.
- **`enumerate()` y `zip()`:** `enumerate(lista)` produce pares índice-valor y `zip(a, b)` combina dos listas hasta la más corta.
- **Colecciones y números:** `first` y `last` funcionan con listas y strings; `take`, `drop`, `sum`, `any`, `all`, `clamp` y `2 ** 3` cubren operaciones frecuentes.
- **`blizzard update`:** actualiza los paquetes bloqueados desde el índice oficial.

```python
const LIMIT = 2
for index, value in enumerate(["a", "b"]):
    print(index, value)

pairs = zip([1, 2], ["one", "two"])
label = "ready" if len(pairs) == LIMIT else "empty"
```

### Instalación

#### Módulo en Go
```bash
go get github.com/JDVA0/blizzard
```

#### Compilar el binario `blizzard`
```bash
git clone https://github.com/JDVA0/blizzard.git
cd blizzard
go build -o blizzard ./cmd/blizzard
sudo mv blizzard /usr/local/bin/
```

#### LSP y extensión de VS Code

Blizzard incluye un servidor LSP con diagnósticos de sintaxis y análisis estático,
autocompletado de palabras clave, funciones y módulos. También reconoce los
miembros de módulos: después de `using http`, escribir `http.` sugiere
`get`, `post`, `put`, `delete`, `patch` y `request`.

```bash
cd LSP
go build -o blizzard-lsp ./server
cd vscode_extension
pnpm install
pnpm run compile
npx @vscode/vsce package --no-dependencies
code --install-extension blizzard-0.1.0.vsix --force
```

La extensión usa `LSP/blizzard-lsp` relativo al workspace por defecto. Se puede
configurar otra ruta con `blizzard.lsp.path` y desactivar el cliente con
`blizzard.lsp.enabled`. La documentación completa está en [LSP/README.md](LSP/README.md)
y [LSP/USAGE.md](LSP/USAGE.md).

### Ejemplo rápido: Servidor API (`api.blizz`)

```python
using api

api.cors()

api.get("/", fn(req): api.text("Servidor Blizzard activo"))

fn ver_usuario(req):
    return api.json({id: req.params.id, activo: true})

api.get("/usuarios/:id", ver_usuario)

fn crear(req):
    return api.json({recibido: req.json}, 201)

api.post("/datos", crear)

api.serve(8080)
```

Ejecutar:
```bash
blizzard api.blizz
```

### Modo interactivo (REPL)
```bash
blizzard repl
```

### Sentencia `match` (comparación por casos)

```python
fn start():  print("Arrancando...")
fn stop():   print("Deteniendo...")
fn status(): print("Estado: OK")

command = "stop"
match command:
    case "start":
        start()
    case "stop":
        stop()
    case "status":
        status()
    # El comodín solo puede aparecer una vez y debe ser el último caso.
    case _:
        print("Unknown command")
```

Ejecutar:
```bash
blizzard app.blizz
```

### Destructuring y acceso seguro

Las listas y diccionarios se pueden desempaquetar directamente:

```python
values = [10, 20]
[first, second] = values

person = {name: "Ada", age: 36}
{name, age} = person

display_name = person?.name ?? "anonymous"
```

### Visibilidad `pub` / `priv` en módulos

Blizzard tiene encapsulación por archivo. Todos los símbolos (funciones y variables) son **`pub` (exportados) por defecto**. Usa `priv` para ocultarlos de otros archivos que hagan `import`.

```python
# ========= math_utils.blizz =========
pub VERSION = "1.0.0"
priv _SAL = "no-exportado"   # oculto fuera del archivo

pub fn add(a, b):
    return a + b

# Sin modificador = pub por defecto
fn multiply(a, b):
    return _helper(a, b)      # dentro del SÍ podemos usar _SAL

priv fn _helper(a, b):        # oculta
    return a * b
```

Importar y usar desde otro archivo. Los símbolos se acceden con **sintaxis punto** `modulo.simbolo`:

```python
# ========= main.blizz =========
import math_utils as m

print(m.VERSION)        # 1.0.0   (pub, funciona)
print(m.add(2, 3))      # 5       (pub, funciona)
print(m.multiply(4, 5)) # 20      (sin modificador = pub, funciona)

# Los símbolos priv NO están en el módulo → error RUNTIME (no de parseo)
print(m._SAL)           # error: attribute not found: _SAL
print(m._helper(1, 2))  # error: attribute not found: _helper
```

Comportamiento clave:
- **No es error de parse** — el archivo compila. El símbolo `priv` simplemente no se agrega al Dict exportado.
- **Al acceder a `foo.no_existe`** se lanza un error en runtime: `attribute not found: no_existe` (no es `nil`).
- **`priv` es encapsulación, no seguridad**: el valor sigue existiendo en memoria del submódulo; solo no se exporta.

Ejemplos completos: `examples/math_utils.blizz`, `examples/string_utils.blizz` y `examples/modules_demo.blizz`. Este último prueba imports con y sin alias, y comprueba que los símbolos privados no se exportan.

### Diagnósticos, tipos graduales y paquetes

Los errores del ejecutable incluyen archivo, línea, columna, la línea de código y un marcador. Los mensajes del runtime y del parser están en inglés para que sean consistentes en herramientas y CI.

Las anotaciones de tipo son opcionales; una variable sin anotación conserva comportamiento dinámico:

```python
age: int = 18
name: str = "Ada"
tags: str[] = ["blizzard", "cli"]
```

Para organizar un proyecto local, crea `blizzard.toml` en la raíz:

```toml
name = "my_app"
source = "src" # opcional; src es el valor por defecto
```

Con `src/utils.blizz`, un script puede importarlo desde cualquier subdirectorio del proyecto:

```python
import my_app.utils
print(utils.slugify("Hello Blizzard"))
```

Los tests de Blizzard son scripts que terminan sin error. Usa `assert(condicion, [mensaje])` para marcar expectativas, guárdalos con el sufijo `_test.blizz` y ejecútalos con:

```bash
blizzard test
blizzard test tests unit/math_test.blizz
blizzard test --filter math
```

Los imports relativos explícitos eliminan ambigüedad entre módulos locales y paquetes:

```python
import ./utils.slug
import ../shared.validators
```

Blizzard detecta ciclos de importación y muestra el módulo que se estaba cargando.

### Gestor de paquetes `blizzard`

`blizzard get` descarga las bibliotecas oficiales desde el repositorio Blizzard en GitHub. Las dependencias por ruta siguen disponibles con `add`; `get-local` está reservado para desarrollar bibliotecas desde una carpeta `repo/` local.

```bash
blizzard init my_app
blizzard run
blizzard run --env .env
blizzard install
blizzard add text_tools ../text_tools
blizzard list
blizzard remove text_tools
blizzard get blizzard/text.blizz
blizzard get blizzard/math.blizz
blizzard get blizzard/arrays.blizz@0.1.0
blizzard get-local blizzard/text.blizz
blizzard search pagination
blizzard index
blizzard update
```

`get` instala la biblioteca en `packages/blizzard/src/`, registra `dep.blizz = "packages/blizzard"` y genera un lockfile reproducible con la versión, el origen y el checksum SHA-256. Usa `@0.1.0` para una versión exacta. `update` reinstala las bibliotecas fijadas en el lockfile.

`blizzard install` reinstala exactamente las versiones de `blizzard.lock`. `blizzard run` ejecuta `src/main.blizz` automáticamente cuando encuentra `blizzard.toml`; `blizzard run --env .env` carga variables del archivo antes de iniciar el programa.

Las bibliotecas oficiales viven ordenadas en `repo/packages/<nombre>/`, con su código en `src/` y metadatos en `package.toml`. `blizzard index` genera `repo/index.toml`. El catálogo oficial incluye 18 bibliotecas, entre ellas `sets` y `paths`. Consulta una biblioteca instalada con `blizzard info blizzard/text.blizz`.

El proyecto [PkgsExamples](/PkgsExamples) contiene una integración completa: sus paquetes están instalados en `PkgsExamples/packages/blizzard/` y `PkgsExamples/src/main.blizz` importa y ejecuta las cuatro bibliotecas oficiales.

Al finalizar, Blizzard imprime el paquete y la versión instalada. Comprueba además el resultado con `blizzard list` y revisando `packages/blizzard/src/`.

Para desarrollo local usa `blizzard get-local` y configura `BLIZZARD_REPO` apuntando a la carpeta `repo/`.

---

### Filtro `where` dentro de `for`

Agrega una condición de filtrado directamente a un `for`. Es semánticamente equivalente a meter un `if cond:` como primera línea del cuerpo, pero más conciso.

```python
# Equivalente a:
#   for user in users:
#       if user.active:
#           print(user.name)
for user in users where user.active:
    print(user.name)
```

Funciona con cualquier tipo de iterable (listas, rangos, resultado de llamadas) y acepta cualquier expresión booleana:

### Pequeñas expresiones y limpieza

```python
const PORT = 8080
estado = "open" if "port" in {port: PORT} else "closed"

try:
    fail("request failed")
catch err:
    print(err)
always:
    print("cleanup runs")
```

`in` sobre un diccionario comprueba sus claves. `always` se ejecuta tanto después del bloque normal como después de `catch`.

```python
# Números pares del 1 al 20
for x in range(1, 21) where (x % 2) == 0:
    print(x)

# Múltiples condiciones
for u in db.all() where u.active and u.age > 21:
    print(u.name)
```

Ejemplo completo: `examples/where_demo.blizz`.

---

### Gestión automática de recursos `with`

Asegura que un recurso se "limpie" (cierre) automáticamente al salir del bloque. Blizzard busca un atributo `close` (tipo `fn` o `native`) sobre el valor y lo invoca sin argumentos.

```python
with fs.open("data.txt") as file:
    content = file.read()
    print(content)
# al salir: file.close() se llama AUTOMÁTICAMENTE
```

Si el valor no tiene método `close`, el bloque simplemente ejecuta el cuerpo sin error (no-op, liberación a mejor esfuerzo).

Recursos hechos a mano (dict con `close`) funcionan igual que objetos built-in:

```python
make_conn = fn(url):
    c = {"url": url, "open": true}
    c["close"] = fn():
        c.open = false
        print("closed " + url)
    return c

with make_conn("postgres://localhost") as db:
    print("connected: " + str(db.open))   # true
# aquí se imprime "closed postgres://localhost"
print(db.open)                            # false
```

Ejemplo completo: `examples/with_demo.blizz`.

---

### Combinados: `with` + `where` + `import`

Todos los features son 100% composables. Ejemplo del spec:

```python
import math_utils as math

with db.open("users.db") as db:
    # Filtramos activos directamente dentro del for
    total = 0
    for user in db.all() where user.active:
        total = math.add(total, user.score)
    print("Suma activos: " + str(total))
```

Ejemplo completo y ejecutable: `examples/combined_demo.blizz`.

---

### Documentación completa

Visita la documentación bilingüe en [https://jdva0.github.io/blizzard/](https://jdva0.github.io/blizzard/) o consulta los archivos en la carpeta `docs/`.

---

## English

Blizzard is a simple, straightforward, and concise programming language designed to make creating REST web APIs and command-line tools fast and effortless, with zero external dependencies.

> Project created by **JDVA0**.
> Built for fun and experimentation.
> Free and open source.
>
> ⓘ Note: this project was developed with the assistance of artificial intelligence.

### Features

- **Clean indented syntax:** Blocks are structured with 4 spaces—no braces `{}` or semicolons `;`.
- **Built-in HTTP server (`using api`):** First-class support for GET, POST, PUT, DELETE, PATCH routes, automatic JSON parsing, CORS, and middlewares.
- **Built-in HTTP client (`using http`):** Make GET, POST, PUT, DELETE requests to external APIs with automatic JSON handling and no external curl dependencies.
- **Embedded key-value store (`using db`):** Lightweight, atomic, JSON-backed persistent storage on disk for effortless state persistence.
- **F-Strings and Elvis operator (`??`):** `f"Hello {user}"`, multiline strings (`"""..."""`), and clean fallback default values.
- **Safe chaining (`?.`, `?[]`):** `data?.user?.profile?.name` walks nested JSON without error, and `??=` assigns only when the variable is nil.
- **Slicing (`[i:j]`):** `list[1:3]`, `text[:2]`, negative indices, and safe slicing `d?.a?.b?[1:]`.
- **`match` / `case`:** Case-based comparison in a single block with multiple values per case and a `case _` wildcard.
- **`for k, v in dict`:** Iterate keys, values or pairs of dicts and lists in one line.
- **`not in` and base-2/8/16 literals:** `if x not in list`, `0b1010`, `0o17`, `0xFF`.
- **Stack traces:** Uncaught errors print the call stack with file, line and column.
- **Linting:** `blizzard check file.blizz` flags undefined names, unused variables, unreachable code and unknown modules without running the program.
- **System and file access (`using sys`, `using fs`):** Run shell commands (`sys.sh`), manage environment variables, inspect processes, and handle disk files directly.
- **CLI toolkit (`using cli`):** Command-line flag parsing, structured text tables, and framed display boxes.
- **Environment and CSV (`using env`, `using csv`):** Environment variables, `.env` files, CSV parse and write.
- **Errors as values:** `try` / `catch` and `fail(value)`. No classes or hierarchies. `nil` is absence, not an error.
- **Typed lists:** `names: str[] = ["Julian", "Ana"]` validates elements at runtime. Also on functions: `fn add(a: int, b: int) -> int:`.
- **Formatter:** `blizzard fmt [-w] file.blizz` reprints source with 4-space indentation.
- **Enhanced REPL:** Persistent command history in `~/.blizz_history`, built-in navigation (`help`, `.exit`, `.history`), and colorized output.
- **Module visibility (`pub` / `priv`):** File-level encapsulation. `pub` exports symbols for `import`, `priv` hides them; no modifier = `pub` by default.
- **Gradual types:** Optional annotations such as `age: int = 18` and `names: str[] = ["Ada"]` are checked at runtime while unannotated code remains dynamic.
- **Local packages:** A project with `blizzard.toml` and `src/` can import its modules by package name: `import my_app.utils`.
- **Script tests:** `blizzard test` runs every `*_test.blizz` file and reports each result.
- **`where` filter inside `for`:** `for x in list where cond:` filters elements inline without a nested `if`.
- **Automatic resource management (`with`):** `with open_resource() as r:` calls `r.close()` automatically at block end (cleanup guaranteed, even if block returns).
- **Composable:** `with` blocks, `where` filters and `pub`/`priv` modules work together seamlessly out of the box.
- **Recent language features:** `const` immutable bindings, `always` cleanup blocks after `try`/`catch`, ternary expressions (`value if condition else other_value`), `enumerate()` / `zip()` list helpers, collection helpers, and the `**` power operator.
- **Package updates:** `blizzard update` refreshes locked packages from the official registry.
- **CLI colors:** Blizzard colors errors, package status, and diagnostics in interactive terminals. Set `NO_COLOR=1` for plain output in scripts and CI.

### Installation

#### Go Module
```bash
go get github.com/JDVA0/blizzard
```

#### Build the `blizzard` binary
```bash
git clone https://github.com/JDVA0/blizzard.git
cd blizzard
go build -o blizzard ./cmd/blizzard
sudo mv blizzard /usr/local/bin/
```

### Quick Example: API Server (`api.blizz`)

```python
using api

api.cors()

api.get("/", fn(req): api.text("Blizzard server active"))

fn get_user(req):
    return api.json({id: req.params.id, active: true})

api.get("/users/:id", get_user)

fn create(req):
    return api.json({received: req.json}, 201)

api.post("/data", create)

api.serve(8080)
```

Run:
```bash
blizzard api.blizz
```

### Interactive REPL
```bash
blizzard repl
```

### `match` statement (case comparison)

```python
fn start():  print("Starting...")
fn stop():   print("Stopping...")
fn status(): print("Status: OK")

command = "stop"
match command:
    case "start":
        start()
    case "stop":
        stop()
    case "status":
        status()
    # The wildcard may appear only once and must be the final case.
    case _:
        print("Unknown command")
```

Run:
```bash
blizzard app.blizz
```

### Module visibility: `pub` / `priv`

Blizzard has file-level encapsulation. Every symbol (functions and variables) is **`pub` (exported) by default**. Use `priv` to hide symbols from external `import`s.

```python
# ========= math_utils.blizz =========
pub VERSION = "1.0.0"
priv _SECRET = "not exported"

pub fn add(a, b):
    return a + b

# No modifier = pub by default
fn multiply(a, b):
    return _helper(a, b)      # inside the file we CAN use priv symbols

priv fn _helper(a, b):        # hidden from other modules
    return a * b
```

Import and use from another file with **dot-access syntax** `module.symbol`:

```python
# ========= main.blizz =========
import math_utils as m

print(m.VERSION)        # 1.0.0   (pub, works)
print(m.add(2, 3))      # 5       (pub, works)
print(m.multiply(4, 5)) # 20      (no modifier = pub, works)

# Priv symbols are NOT in the exported module → runtime error (not a parse error)
print(m._SECRET)        # error: attribute not found: _SECRET
print(m._helper(1, 2))  # error: attribute not found: _helper
```

Key behavior:
- **Not a parse error** — the file compiles fine. `priv` symbols are simply filtered out when building the exported module Dict.
- **Accessing `foo.does_not_exist`** throws a runtime error: `attribute not found: does_not_exist` (it doesn't silently become `nil`).
- **`priv` = encapsulation, not security**: the value still lives in the sub-interpreter memory; it's just not reachable via the module surface.

Complete examples: `examples/math_utils.blizz`, `examples/string_utils.blizz`, and `examples/modules_demo.blizz`. The last one tests aliased and unaliased imports, and verifies that private symbols are not exported.

### Diagnostics, gradual types, and packages

The command-line runner reports errors with the file, line, column, source line, and a caret. Parser and runtime diagnostic messages are in English so they remain consistent in tooling and CI.

Type annotations are optional; unannotated variables stay dynamic:

```python
age: int = 18
name: str = "Ada"
tags: str[] = ["blizzard", "cli"]
```

To organize a local package, add `blizzard.toml` at the project root:

```toml
name = "my_app"
source = "src" # optional; src is the default
```

With `src/utils.blizz`, any script under the project can use:

```python
import my_app.utils
print(utils.slugify("Hello Blizzard"))
```

Blizzard tests are scripts that complete without an error. Use `assert(condition, [message])` for expectations, name them with `_test.blizz`, and run:

```bash
blizzard test
blizzard test tests unit/math_test.blizz
blizzard test --filter math
```

Explicit relative imports remove ambiguity between local modules and packages:

```python
import ./utils.slug
import ../shared.validators
```

Blizzard detects import cycles and reports the module that was being loaded.

### `blizzard` package manager

`blizzard get` downloads official libraries from Blizzard's GitHub repository. Path dependencies remain available through `add`; `get-local` is reserved for developing libraries from a local `repo/` directory.

```bash
blizzard init my_app
blizzard add text_tools ../text_tools
blizzard list
blizzard remove text_tools
blizzard get blizzard/text.blizz
blizzard get blizzard/math.blizz
blizzard get blizzard/arrays.blizz@0.1.0
blizzard get-local blizzard/text.blizz
blizzard search pagination
blizzard index
blizzard update
```

`get` installs the library in `packages/blizzard/src/`, writes `dep.blizz = "packages/blizzard"`, and creates a reproducible lockfile with version, source, and SHA-256 checksum. Use `@0.1.0` for an exact version. `update` manages locked packages.

Official libraries are organized in `repo/packages/<name>/`, with source in `src/` and metadata in `package.toml`. The official catalog includes 16 libraries: `text`, `math`, `collections`, `validate`, `arrays`, `dict`, `strings`, `numbers`, `query`, `csvutil`, `pagination`, `result`, `guards`, `ids`, `template`, and `stats`. Inspect an installed library with `blizzard info blizzard/text.blizz`.

[PkgsExamples](/PkgsExamples) is a complete integration project: its packages live in `PkgsExamples/packages/blizzard/`, and `PkgsExamples/src/main.blizz` imports and runs all four official libraries.

When it finishes, Blizzard prints the installed package and version. You can verify the result with `blizzard list` and by inspecting `packages/blizzard/src/`.

For local development use `blizzard get-local` and set `BLIZZARD_REPO` to the `repo/` directory.

---

### `where` clause inside `for`

Adds an inline filter condition straight into a `for` loop. Semantically equivalent to nesting an `if cond:` as the first body line — just shorter.

```python
# Equivalent to:
#   for user in users:
#       if user.active:
#           print(user.name)
for user in users where user.active:
    print(user.name)
```

Works with any iterable (lists, ranges, call results) and accepts any boolean expression:

```python
# Even numbers from 1 to 20
for x in range(1, 21) where (x % 2) == 0:
    print(x)

# Multiple conditions
for u in db.all() where u.active and u.age > 21:
    print(u.name)
```

Full example: `examples/where_demo.blizz`.

---

### Automatic resource management with `with`

Guarantees a resource is cleaned up (closed) automatically when the block exits. Blizzard looks for a `close` attribute (of type `fn` or `native`) on the value and invokes it with no arguments.

```python
with fs.open("data.txt") as file:
    content = file.read()
    print(content)
# on exit: file.close() is called AUTOMATICALLY
```

If a value has no `close` method, the body still runs with no error (best-effort / no-op cleanup).

Hand-rolled resources (a dict carrying `close`) work exactly like built-in objects:

```python
make_conn = fn(url):
    c = {"url": url, "open": true}
    c["close"] = fn():
        c.open = false
        print("closed " + url)
    return c

with make_conn("postgres://localhost") as db:
    print("connected: " + str(db.open))   # true
# → prints "closed postgres://localhost" here
print(db.open)                            # false
```

Full example: `examples/with_demo.blizz`.

---

### Composing everything: `with` + `where` + `import`

All features compose 100% naturally. Example from the spec:

```python
import math_utils as math

with db.open("users.db") as db:
    # Filter active users inline inside the loop
    total = 0
    for user in db.all() where user.active:
        total = math.add(total, user.score)
    print("Active sum: " + str(total))
```

Full runnable example: `examples/combined_demo.blizz`.

---

### Full Documentation

Read the complete bilingual documentation at [https://jdva0.github.io/blizzard/](https://jdva0.github.io/blizzard/) or explore the `docs/` directory.

---

## License / Licencia

Distribuido bajo la Licencia MIT. / Released under the MIT License.
