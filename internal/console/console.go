package console

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Listen reads local administrator commands from the foreground terminal.
// It calls the same local HTTP API used by the web UI, so both interfaces
// always manipulate the exact same live state.
func Listen(port string) {
	go func() {
		fmt.Println("Consola admin: help | status | pause | play | auto | seek <segundos> | select <id> | media <ruta> | scan")
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			execute(port, strings.TrimSpace(scanner.Text()))
		}
	}()
}

func execute(port, line string) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return
	}
	base := "http://127.0.0.1:" + port
	if parts[0] == "help" {
		fmt.Println("pause, play, auto, seek <segundos>, select <id>, media <ruta>, scan, status")
		return
	}
	if parts[0] == "status" {
		request(http.MethodGet, base+"/api/v1/tv/live", nil)
		request(http.MethodGet, base+"/api/v1/settings/media", nil)
		return
	}
	if parts[0] == "scan" {
		request(http.MethodPost, base+"/api/v1/library/scan", nil)
		return
	}
	if parts[0] == "media" {
		path := strings.TrimSpace(strings.TrimPrefix(line, "media"))
		if path == "" {
			fmt.Println("Uso: media <ruta>")
			return
		}
		request(http.MethodPut, base+"/api/v1/settings/media", map[string]string{"media_dir": path})
		request(http.MethodPost, base+"/api/v1/library/scan", nil)
		return
	}
	payload := map[string]any{"action": parts[0]}
	if parts[0] == "seek" || parts[0] == "select" {
		if len(parts) != 2 {
			fmt.Printf("Uso: %s <numero>\n", parts[0])
			return
		}
		n, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			fmt.Println("Numero invalido")
			return
		}
		if parts[0] == "seek" {
			payload["seek_seconds"] = n
		} else {
			payload["media_id"] = n
		}
	}
	if parts[0] != "pause" && parts[0] != "play" && parts[0] != "auto" && parts[0] != "seek" && parts[0] != "select" {
		fmt.Println("Comando desconocido. Escribe help.")
		return
	}
	request(http.MethodPost, base+"/api/v1/tv/control", payload)
}

func request(method, url string, body any) {
	var reader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		fmt.Println(err)
		return
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	fmt.Printf("%s\n", strings.TrimSpace(string(data)))
	if resp.StatusCode >= 300 {
		fmt.Println(resp.Status)
	}
}
