package textmatch

import (
	"slices"
	"strings"
)

func IsActive(input string) bool {
	v := strings.ToLower(strings.TrimSpace(input))
	actives := []string{
		"activo", "activa", "activos", "on", "si", "sí", "true", "1", "habilitado", "habilitada",
		"enable", "enabled", "ok", "operativo", "funcional", "vigente",
	}
	inactives := []string{
		"inactivo", "inactiva", "inactivos", "off", "no", "false", "0", "deshabilitado", "deshabilitada",
		"disable", "disabled", "inoperativo", "no operativo", "caducado", "suspendido", "baja",
	}
	if slices.Contains(actives, v) {
		return true
	}
	if slices.Contains(inactives, v) {
		return false
	}
	return false
}
