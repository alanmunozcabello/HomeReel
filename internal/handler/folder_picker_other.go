//go:build !windows

package handler

import (
	"context"
	"fmt"
)

func pickMediaFolder(context.Context) (string, error) {
	return "", fmt.Errorf("el selector nativo solo esta disponible en Windows; escribe la ruta manualmente")
}
