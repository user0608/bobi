# Bobi skills para OpenCode

Skills portables para que OpenCode use correctamente la libreria Go `github.com/user0608/bobi`.

## Instalacion en otro proyecto

Desde la raiz del proyecto consumidor, copia el contenido de esta carpeta dentro de `.opencode/skills/`:

```bash
mkdir -p .opencode/skills
cp -R /ruta/a/bobi/.opencode-skills/bobi .opencode/skills/
```

La estructura final debe ser:

```text
.opencode/skills/bobi/SKILL.md
.opencode/skills/bobi/httpserver/SKILL.md
...
```

OpenCode busca `SKILL.md` recursivamente. Reinicia OpenCode despues de copiar o actualizar las skills.

## Actualizacion

Vuelve a copiar la carpeta `bobi` cuando cambie la API publica de la libreria. Las skills describen la version del modulo indicada en el proyecto origen; confirma siempre la version de `bobi` en `go.mod` del proyecto consumidor antes de aplicar un ejemplo.
