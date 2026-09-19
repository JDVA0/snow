# <img src="docs/snowflake.svg" width="36" height="36" alt="Snow icon" style="vertical-align: middle;"> Snow 0.1

[![Go Report Card](https://goreportcard.com/badge/github.com/JDVA0/snow)](https://goreportcard.com/report/github.com/JDVA0/snow)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[Español](#español) | [English](#english)

---

## Español

Snow es un lenguaje de programación simple, directo y conciso, diseñado específicamente para facilitar la creación de APIs web REST y herramientas de línea de comandos sin dependencias externas.

> **Nota:** Hecho por **JDVA0** con asistencia de inteligencia artificial. Creado por diversión y para experimentar, totalmente libre y abierto para que cualquier persona en el mundo lo use, aprenda y cree APIs sin complicaciones.

### Características

- **Sintaxis limpia por indentación:** Bloques definidos con 4 espacios, sin llaves `{}` ni puntos y comas `;`.
- **Servidor HTTP integrado (`using api`):** Rutas GET, POST, PUT, DELETE, PATCH, soporte JSON nativo, CORS y middlewares en pocas líneas.
- **Cliente HTTP integrado (`using http`):** Peticiones GET, POST, PUT, DELETE a APIs externas con parseo JSON automático sin depender de curl.
- **Base de datos clave-valor embebida (`using db`):** Almacén JSON estructurado y persistente en disco con escritura atómica.
- **Interpolación F-Strings y Operador Elvis (`??`):** `f"Hola {usuario}"`, cadenas multilínea con `"""` y valor por defecto para nulos.
- **Encadenado seguro (`?.`, `?[]`):** `data?.user?.profile?.name` recorre JSON anidado sin lanzar error y `??=` asigna solo si la variable es `nil`.
- **Cortes (`[i:j]`):** `lista[1:3]`, `texto[:2]`, índices negativos y slicing seguro `d?.a?.b?[1:]`.
- **`match` / `case`:** Comparación por casos en un solo bloque con soporte de múltiples valores y `else`.
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

### Documentación completa

Visita la documentación bilingüe en [https://jdva0.github.io/snow/](https://jdva0.github.io/snow/) o consulta los archivos en la carpeta `docs/`.

---

## English

Snow is a simple, straightforward, and concise programming language designed to make creating REST web APIs and command-line tools fast and effortless, with zero external dependencies.

> **Note:** Made by **JDVA0** with artificial intelligence assistance. Built for fun and experimentation, completely free and open for anyone in the world to use, learn, and create APIs without hassle.

### Features

- **Clean indented syntax:** Blocks are structured with 4 spaces—no braces `{}` or semicolons `;`.
- **Built-in HTTP server (`using api`):** First-class support for GET, POST, PUT, DELETE, PATCH routes, automatic JSON parsing, CORS, and middlewares.
- **Built-in HTTP client (`using http`):** Make GET, POST, PUT, DELETE requests to external APIs with automatic JSON handling and no external curl dependencies.
- **Embedded key-value store (`using db`):** Lightweight, atomic, JSON-backed persistent storage on disk for effortless state persistence.
- **F-Strings and Elvis operator (`??`):** `f"Hello {user}"`, multiline strings (`"""..."""`), and clean fallback default values.
- **Safe chaining (`?.`, `?[]`):** `data?.user?.profile?.name` walks nested JSON without error, and `??=` assigns only when the variable is nil.
- **Slicing (`[i:j]`):** `list[1:3]`, `text[:2]`, negative indices, and safe slicing `d?.a?.b?[1:]`.
- **`match` / `case`:** Case-based comparison in a single block with multi-values and `else`.
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

### Full Documentation

Read the complete bilingual documentation at [https://jdva0.github.io/snow/](https://jdva0.github.io/snow/) or explore the `docs/` directory.

---

## License / Licencia

Distribuido bajo la Licencia MIT. / Released under the MIT License.
