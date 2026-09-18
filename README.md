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
- **Acceso a sistema y archivos (`using sys`, `using fs`):** Ejecución de comandos del sistema (`sys.sh`), lectura de entorno, control de procesos y operaciones de archivos.
- **Herramientas de consola (`using cli`):** Análisis de banderas/argumentos, formateo de tablas y cajas de texto.

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
- **System and file access (`using sys`, `using fs`):** Run shell commands (`sys.sh`), manage environment variables, inspect processes, and handle disk files directly.
- **CLI toolkit (`using cli`):** Command-line flag parsing, structured text tables, and framed display boxes.

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
