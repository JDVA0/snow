# Blizzard 0.2

> Blizzard es un proyecto experimental asistido por IA. Puede cambiar de forma
> brusca: la sintaxis, la API y los formatos de proyecto no son contratos de
> estabilidad. Usa versiones fijadas y no lo trates todavía como dependencia
> de producción.

Blizzard 0.2 es el nuevo nombre de Snow. El runtime usa el módulo Go
`github.com/JDVA0/blizzard` y el comando `blizzard`.

## Cambio de nombre

| Snow 0.1 | Blizzard 0.2 |
|---|---|
| `snowman` | `blizzard` |
| `.snow` | `.blizz` |
| `.snowpkg` | `.blizzpkg` |
| `snow.toml` | `blizzard.toml` |
| `snow.lock` | `blizzard.lock` |
| `snow-lsp` | `blizzard-lsp` |
| `github.com/JDVA0/snow` | `github.com/JDVA0/blizzard` |

El repositorio ya contiene ejemplos, paquetes, tests y documentación con los
nombres nuevos. Para migrar un proyecto anterior, renombra primero los archivos,
cambia sus imports y crea un lockfile nuevo.

## Sintaxis estable

Los bloques modernos usan llaves y los saltos de línea separan sentencias:

```blizzard
import json

let values = [1, 2, 3]

fn total(items) {
    let result = 0
    for item in items {
        result += item
    }
    return result
}

if total(values) > 0 {
    print(json.stringify(values))
} else {
    print("empty")
}
```

Los puntos y coma son opcionales. Se pueden usar para varias sentencias en una
misma línea. `let` declara una variable mutable y `const` declara un valor que
no puede reasignarse. Los tipos se infieren cuando no se escribe una anotación.

## Control de flujo

```blizzard
match status {
    case "ready" {
        print("go")
    }
    case "waiting", "queued" {
        print("later")
    }
    case _ {
        print("unknown")
    }
}

try {
    fail("problem")
} catch err {
    print(err)
} always {
    print("cleanup")
}
```

También están disponibles `while`, `break`, `continue`, `with`, `where`,
funciones anónimas, destructuring, slicing, acceso seguro (`?.`, `?[`),
coalescencia (`??`) y f-strings.

## Módulos

`import` es la forma recomendada para módulos estándar y locales:

```blizzard
import json
import my_app.helpers as helpers

print(json.stringify(helpers.build()))
```

`using` continúa aceptándose como sintaxis heredada. Los archivos importados
conservan `pub` y `priv` para controlar sus símbolos exportados.

## Compatibilidad de migración

La sintaxis antigua con `:` e indentación sigue siendo válida durante la serie
0.2. Esto permite migrar archivo por archivo. `blizzard fmt` conserva el estilo
heredado y formatea los archivos modernos con llaves, `let`, `import` y
separadores apropiados.

## Pruebas de estabilidad

Los programas de estrés están en `testdata/programs/`. La suite Go comprueba:

- ejecución de un programa completo con imports, closures, loops, `match` y
  `try/catch/always`;
- bloques multilínea sin `;`;
- diccionarios anidados dentro de bloques;
- errores de llaves incompletas y casos inválidos;
- idempotencia del formatter.

Ejecuta todas las comprobaciones con:

```bash
go test ./...
cd LSP && go test ./...
```

`blizzard build archivo.blizz` valida el programa y comprueba que el compilador
puede producir sus operaciones sin ejecutarlo. Los errores posicionados exponen
los códigos `B001` a `B005` para que editores y herramientas puedan reaccionar
sin analizar el texto humano del mensaje. `B100` avisa cuando un archivo todavía
usa bloques por indentación. El checker también infiere tipos ciertos de
literales, variables y operaciones antes de ejecutar.

El gestor acepta versiones exactas y restricciones `^`, `~`, `>=`, `<=`, `<` y
`>`. Los lockfiles guardan un checksum SHA-256; la caché global y la validación
obligatoria de todos los artefactos son parte de la estabilización posterior.
