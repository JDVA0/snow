# <img src="docs/snowflake.svg" width="36" height="36" alt="Snow icon" style="vertical-align: middle;"> Snow 0.1

[![Go Report Card](https://goreportcard.com/badge/github.com/JDVA0/snow)](https://goreportcard.com/report/github.com/JDVA0/snow)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[Español](#español) | [English](#english)

---

## Español

Snow es un lenguaje de programación simple, directo y conciso, diseñado específicamente para facilitar la creación de APIs web REST y herramientas de línea de comandos sin dependencias externas.

> Proyecto creado por **JDVA0**.
> Hecho por diversión y experimentación.
> Libre y de código abierto.
>
> ⓘ Nota: este proyecto fue desarrollado con asistencia de inteligencia artificial.

### Características

- **Sintaxis limpia por indentación:** Bloques definidos con 4 espacios, sin llaves `{}` ni puntos y comas `;`.
- **Servidor HTTP integrado (`using api`):** Rutas GET, POST, PUT, DELETE, PATCH, soporte JSON nativo, CORS y middlewares en pocas líneas.
- **Cliente HTTP integrado (`using http`):** Peticiones GET, POST, PUT, DELETE a APIs externas con parseo JSON automático sin depender de curl.
- **Base de datos clave-valor embebida (`using db`):** Almacén JSON estructurado y persistente en disco con escritura atómica.
- **Interpolación F-Strings y Operador Elvis (`??`):** `f"Hola {usuario}"`, cadenas multilínea con `"""` y valor por defecto para nulos.
- **Encadenado seguro (`?.`, `?[]`):** `data?.user?.profile?.name` recorre JSON anidado sin lanzar error y `??=` asigna solo si la variable es `nil`.
- **Cortes (`[i:j]`):** `lista[1:3]`, `texto[:2]`, índices negativos y slicing seguro `d?.a?.b?[1:]`.
- **`match` / `case`:** Comparación por casos en un solo bloque con múltiples valores por caso y comodín `case _`.
- **`for k, v in dict`:** Itera claves, valores o pares de diccionarios y listas en una sola línea.
- **`not in` y literales en base 2/8/16:** `if x not in lista`, `0b1010`, `0o17`, `0xFF`.
- **Stack traces:** Los errores no capturados muestran la pila de llamadas con archivo, línea y columna.
- **Linting:** `snowman check archivo.snow` detecta nombres no definidos, variables sin usar, código inalcanzable y módulos desconocidos sin ejecutar el programa.
- **Acceso a sistema y archivos (`using sys`, `using fs`):** Ejecución de comandos del sistema (`sys.sh`), lectura de entorno, control de procesos y operaciones de archivos.
- **Herramientas de consola (`using cli`):** Análisis de banderas/argumentos, tablas alineadas y cajas de texto formateadas.
- **Entorno y CSV (`using env`, `using csv`):** Variables de entorno, archivos `.env`, parseo y escritura de CSV.
- **Errores como valores:** `try` / `catch` y `fail(valor)`. Sin clases ni jerarquías. `nil` es ausencia, no un error.
- **Listas tipadas:** `nombres: str[] = ["Julian", "Ana"]` valida elementos en tiempo de ejecución. También en funciones: `fn sumar(a: int, b: int) -> int:`.
- **Formato:** `snowman fmt [-w] archivo.snow` reindenta el código con 4 espacios.
- **REPL interactivo avanzado:** Historial persistente en `~/.snow_history`, comandos (`help`, `.exit`, `.history`) y colores por tipo de dato.
- **Visibilidad de módulos (`pub` / `priv`):** Encapsulación por archivo. `pub` exporta, `priv` oculta del `import`, sin modificador = `pub` por defecto.
- **Filtro `where` en `for`:** `for x in lista where cond:` filtra elementos directamente sin un `if` anidado.
- **Gestión automática de recursos (`with`):** `with abrir_recurso() as r:` libera el recurso (llama `r.close()`) automáticamente al salir del bloque.
- **Composición:** Los bloques `with`, filtros `where` y `import` de módulos con `pub/priv` funcionan juntos de forma natural.

### Instalación

#### Módulo en Go
```bash
go get github.com/JDVA0/snow
```

#### Compilar el binario `snowman`
```bash
git clone https://github.com/JDVA0/snow.git
cd snow
go build -o snowman ./cmd/snowman
sudo mv snowman /usr/local/bin/
```

### Ejemplo rápido: Servidor API (`api.snow`)

```python
using api

api.cors()

api.get("/", fn(req): api.text("Servidor Snow activo"))

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
snowman api.snow
```

### Modo interactivo (REPL)
```bash
snowman repl
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
    case _:
        print("Unknown command")
```

Ejecutar:
```bash
snowman app.snow
```

### Visibilidad `pub` / `priv` en módulos

Snow tiene encapsulación por archivo. Todos los símbolos (funciones y variables) son **`pub` (exportados) por defecto**. Usa `priv` para ocultarlos de otros archivos que hagan `import`.

```python
# ========= math_utils.snow =========
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
# ========= main.snow =========
import math_utils as m

print(m.VERSION)        # 1.0.0   (✅ pub, funciona)
print(m.add(2, 3))      # 5       (✅ pub, funciona)
print(m.multiply(4, 5)) # 20      (✅ sin modificador = pub, funciona)

# Los símbolos priv NO están en el módulo → error RUNTIME (no de parseo)
print(m._SAL)           # ❌ attribute not found: _SAL
print(m._helper(1, 2))  # ❌ attribute not found: _helper
```

Comportamiento clave:
- **No es error de parse** — el archivo compila. El símbolo `priv` simplemente no se agrega al Dict exportado.
- **Al acceder a `foo.no_existe`** se lanza un error en runtime: `attribute not found: no_existe` (no es `nil`).
- **`priv` es encapsulación, no seguridad**: el valor sigue existiendo en memoria del submódulo; solo no se exporta.

Ejemplos completos: `examples/math_utils.snow` + `examples/import_demo.snow`.

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

```python
# Números pares del 1 al 20
for x in range(1, 21) where (x % 2) == 0:
    print(x)

# Múltiples condiciones
for u in db.all() where u.active and u.age > 21:
    print(u.name)
```

Ejemplo completo: `examples/where_demo.snow`.

---

### Gestión automática de recursos `with`

Asegura que un recurso se "limpie" (cierre) automáticamente al salir del bloque. Snow busca un atributo `close` (tipo `fn` o `native`) sobre el valor y lo invoca sin argumentos.

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

Ejemplo completo: `examples/with_demo.snow`.

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

Ejemplo completo y ejecutable: `examples/combined_demo.snow`.

---

### Documentación completa

Visita la documentación bilingüe en [https://jdva0.github.io/snow/](https://jdva0.github.io/snow/) o consulta los archivos en la carpeta `docs/`.

---

## English

Snow is a simple, straightforward, and concise programming language designed to make creating REST web APIs and command-line tools fast and effortless, with zero external dependencies.

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
- **Linting:** `snowman check file.snow` flags undefined names, unused variables, unreachable code and unknown modules without running the program.
- **System and file access (`using sys`, `using fs`):** Run shell commands (`sys.sh`), manage environment variables, inspect processes, and handle disk files directly.
- **CLI toolkit (`using cli`):** Command-line flag parsing, structured text tables, and framed display boxes.
- **Environment and CSV (`using env`, `using csv`):** Environment variables, `.env` files, CSV parse and write.
- **Errors as values:** `try` / `catch` and `fail(value)`. No classes or hierarchies. `nil` is absence, not an error.
- **Typed lists:** `names: str[] = ["Julian", "Ana"]` validates elements at runtime. Also on functions: `fn add(a: int, b: int) -> int:`.
- **Formatter:** `snowman fmt [-w] file.snow` reprints source with 4-space indentation.
- **Enhanced REPL:** Persistent command history in `~/.snow_history`, built-in navigation (`help`, `.exit`, `.history`), and colorized output.
- **Module visibility (`pub` / `priv`):** File-level encapsulation. `pub` exports symbols for `import`, `priv` hides them; no modifier = `pub` by default.
- **`where` filter inside `for`:** `for x in list where cond:` filters elements inline without a nested `if`.
- **Automatic resource management (`with`):** `with open_resource() as r:` calls `r.close()` automatically at block end (cleanup guaranteed, even if block returns).
- **Composable:** `with` blocks, `where` filters and `pub`/`priv` modules work together seamlessly out of the box.

### Installation

#### Go Module
```bash
go get github.com/JDVA0/snow
```

#### Build the `snowman` binary
```bash
git clone https://github.com/JDVA0/snow.git
cd snow
go build -o snowman ./cmd/snowman
sudo mv snowman /usr/local/bin/
```

### Quick Example: API Server (`api.snow`)

```python
using api

api.cors()

api.get("/", fn(req): api.text("Snow server active"))

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
snowman api.snow
```

### Interactive REPL
```bash
snowman repl
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
    case _:
        print("Unknown command")
```

Run:
```bash
snowman app.snow
```

### Module visibility: `pub` / `priv`

Snow has file-level encapsulation. Every symbol (functions and variables) is **`pub` (exported) by default**. Use `priv` to hide symbols from external `import`s.

```python
# ========= math_utils.snow =========
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
# ========= main.snow =========
import math_utils as m

print(m.VERSION)        # 1.0.0   (✅ pub, works)
print(m.add(2, 3))      # 5       (✅ pub, works)
print(m.multiply(4, 5)) # 20      (✅ no modifier = pub, works)

# Priv symbols are NOT in the exported module → runtime error (not a parse error)
print(m._SECRET)        # ❌ attribute not found: _SECRET
print(m._helper(1, 2))  # ❌ attribute not found: _helper
```

Key behavior:
- **Not a parse error** — the file compiles fine. `priv` symbols are simply filtered out when building the exported module Dict.
- **Accessing `foo.does_not_exist`** throws a runtime error: `attribute not found: does_not_exist` (it doesn't silently become `nil`).
- **`priv` = encapsulation, not security**: the value still lives in the sub-interpreter memory; it's just not reachable via the module surface.

Complete examples: `examples/math_utils.snow` + `examples/import_demo.snow`.

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

Full example: `examples/where_demo.snow`.

---

### Automatic resource management with `with`

Guarantees a resource is cleaned up (closed) automatically when the block exits. Snow looks for a `close` attribute (of type `fn` or `native`) on the value and invokes it with no arguments.

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

Full example: `examples/with_demo.snow`.

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

Full runnable example: `examples/combined_demo.snow`.

---

### Full Documentation

Read the complete bilingual documentation at [https://jdva0.github.io/snow/](https://jdva0.github.io/snow/) or explore the `docs/` directory.

---

## License / Licencia

Distribuido bajo la Licencia MIT. / Released under the MIT License.
