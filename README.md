# Snow 0.1

Snow es un lenguaje de programación simple, directo y conciso, diseñado para facilitar la creación de APIs web REST y herramientas de línea de comandos.

> **Nota del proyecto:** Hecho por **JDVA0** con asistencia de inteligencia artificial para facilitar la creación de APIs y utilidades de sistema.

---

## Características

- **Sintaxis limpia por indentación:** Los bloques se definen con 4 espacios. No se usan llaves `{}` ni puntos y comas `;`.
- **Servidor HTTP integrado (`api`):** Creación de rutas GET, POST, PUT, DELETE y PATCH con soporte automático de JSON, CORS en una sola línea y middleware.
- **Acceso al sistema (`sys`):** Ejecución de comandos del sistema operativo (`sys.sh`), consulta de variables de entorno, fecha/hora y control de procesos.
- **Manejo de archivos (`fs`):** Lectura, escritura, adición y comprobación de existencia de archivos de forma directa.
- **Herramientas de consola (`cli`):** Formateo de tablas de texto estructuradas, marcos de texto y análisis de argumentos y banderas de línea de comandos.

---

## Instalación

### Módulo de Go

El módulo oficial para Go es:

```bash
go get github.com/JDVA0/snow
```

### Compilar el binario `snowman`

Para ejecutar archivos `.snow` directamente en tu terminal:

```bash
git clone https://github.com/JDVA0/snow.git
cd snow
go build -o snowman ./cmd/snowman
sudo mv snowman /usr/local/bin/
```

---

## Uso rápido

### 1. Servidor API Web (`servidor.snow`)

```python
using api

api.cors()

api.get("/", fn(req): api.text("Servidor Snow funcionando"))

fn obtener_usuario(req):
    id = req.params.id
    return api.json({id: id, activo: true})

api.get("/usuarios/:id", obtener_usuario)

fn crear_elemento(req):
    return api.json({recibido: req.json}, 201)

api.post("/elementos", crear_elemento)

api.serve(8080)
```

Ejecución:

```bash
snowman servidor.snow
```

### 2. Script de consola (`info.snow`)

```python
using cli
using sys
using fs

cli.box("INFORMACION", "Estado del sistema")
cli.info("Plataforma: " + sys.platform + " (" + sys.arch + ")")

archivos = fs.list(".")
filas = []
for a in archivos:
    filas = append(filas, [a])

cli.table(["ARCHIVOS EN DIRECTORIO"], filas)
```

### 3. Modo interactivo (REPL)

```bash
snowman repl
```

---

## Documentación completa

La documentación completa y bilingüe (Español / Inglés) con ejemplos prácticos está disponible en la carpeta `docs/` o en [https://jdva0.github.io/snow/](https://jdva0.github.io/snow/).

---

## Licencia

Este proyecto está distribuido bajo la licencia MIT.
