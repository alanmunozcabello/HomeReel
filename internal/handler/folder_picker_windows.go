//go:build windows

package handler

import (
	"context"
	"os/exec"
	"strings"
)

func pickMediaFolder(ctx context.Context) (string, error) {
	script := `Add-Type -AssemblyName System.Windows.Forms; $d=New-Object System.Windows.Forms.FolderBrowserDialog; $d.Description='Selecciona la carpeta raiz de medios'; if($d.ShowDialog() -eq 'OK'){[Console]::Write($d.SelectedPath)}`
	out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-STA", "-Command", script).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
