## Requisitos Funcionales (RF)

Las acciones y comportamientos específicos que el sistema debe ejecutar para el usuario final y los procesos internos.

1. **RF01 - Escaneo de Directorios:** El sistema debe recorrer directorios específicos definidos en busca de archivos de video válidos (`.mp4`, `.mkv`).
2. **RF02 - Extracción de Metadatos Técnicos:** El sistema debe extraer la duración exacta y el códec de video de cada archivo detectado utilizando `ffprobe`.
3. **RF03 - Escaneo Diferencial:** El sistema debe identificar si un archivo es nuevo, si fue modificado (tamaño/fecha) o si fue desconectado, actualizando el catálogo sin reprocesar archivos inalterados.
4. **RF04 - Streaming Personal (VOD):** El sistema debe servir archivos de video directamente al cliente permitiendo saltos de tiempo (*seek* / avance / retroceso) mediante peticiones HTTP parciales.
5. **RF05 - Motor de Canal de TV (Live Sync):** El sistema debe calcular en tiempo real un ciclo infinito de reproducción y sincronizar a los clientes en el mismo segundo de emisión, independientemente de cuándo se conecten.
6. **RF06 - API RESTful:** El sistema debe exponer endpoints HTTP para iniciar escaneos manuales, consultar el catálogo y obtener el estado del stream en vivo.
7. **RF07 - Interfaz de Reproducción (Web UI):** El sistema debe proveer un cliente web en HTML/JS capaz de consumir la API, reproducir el video e integrarse con el motor del Canal de TV de forma autónoma.
8. **RF08 - Resiliencia de Discos Extraíbles:** El sistema debe marcar los videos como "no disponibles" (`is_available = false`) en lugar de eliminarlos si el disco duro o pendrive de origen se desconecta.

## Requisitos No Funcionales (RNF)

Las restricciones de diseño, rendimiento y arquitectura obligatorias para que el sistema opere en el hardware objetivo.

1. **RNF01 - Restricción de Hardware (Raspberry Pi 3):** El consumo de CPU debe mantenerse al mínimo mediante el uso de concurrencia controlada (*Worker Pools*), y el uso de memoria RAM no debe superar los 500 MB durante escaneos intensivos.
2. **RNF02 - Cero Transcodificación (Direct Play):** El servidor tiene estrictamente prohibido re-codificar video al vuelo. La carga de decodificación debe delegarse íntegramente al dispositivo cliente (navegador/Smart TV).
3. **RNF03 - Concurrencia de Base de Datos:** La lectura (streaming) y la escritura (escáner) de datos deben ocurrir simultáneamente sin generar errores de tipo `database is locked`. Se exige el uso de SQLite en modo WAL (*Write-Ahead Logging*).
4. **RNF04 - Portabilidad CGO-Free:** El backend debe estar escrito enteramente en Go puro (usando `modernc.org/sqlite`), evitando dependencias en C para asegurar la compilación cruzada a ARM sin fricción.
5. **RNF05 - Despliegue Simple:** El sistema debe poder instalarse como ejecutable nativo, con scripts que preparen FFmpeg y las dependencias de compilación.

## Módulos del Sistema y Mapeo de Requisitos

| Módulo | Responsabilidad Principal | Requisitos Satisfechos |
| --- | --- | --- |
| **1. Motor de Persistencia (`internal/repository`)** | Gestiona la conexión a la base de datos SQLite y ejecuta operaciones CRUD con control de concurrencia y contexto. | RF03, RF08, RNF03, RNF04 |
| **2. Scanner & Metadata Engine (`internal/scanner`)** | Recorre directorios (O(1) para archivos conocidos) y administra un Worker Pool de `ffprobe` para no asfixiar el procesador. | RF01, RF02, RF03, RNF01 |
| **3. Streaming Engine (`internal/streaming`)** | Despacha bytes nativamente vía `http.ServeContent` para VOD y calcula algorítmicamente la posición actual del loop en vivo para TV. | RF04, RF05, RNF01, RNF02 |
| **4. API & Handlers (`internal/handler`)** | Expone y multiplexa las rutas HTTP, uniendo el motor de streaming, el repositorio y el escáner con el exterior. | RF06 |
| **5. Cliente Frontend (`web/`)** | Consume los metadatos de la API, carga la interfaz nativa del reproductor (Video.js) y maneja la re-sincronización. | RF07, RNF02 |

## Uso en LAN

La demo sirve vídeo directamente, sin transcodificación ni dependencias de Internet. Para una reproducción compatible con navegadores, Smart TV y Raspberry Pi, se recomienda MP4 con H.264/AAC.

### Windows

1. Clona o descarga el repositorio.
2. Haz doble clic en **`Instalar HomeReel.cmd`**. Instala Go/FFmpeg si faltan, compila el proyecto y abre el puerto LAN.
3. Haz doble clic en **`HomeReel.exe`**. La consola queda abierta mientras el servidor está activo.

### Raspberry Pi / Linux

1. Clona el repositorio.
2. Ejecuta `bash scripts/install-linux.sh`. El script instala Go y FFmpeg con `apt`, y compila el binario para la arquitectura local.
3. Inicia con `./homereel`.

### macOS

1. Descarga `HomeReel-macos-arm64.pkg` para Apple Silicon o `HomeReel-macos-amd64.pkg` para Intel desde los archivos adjuntos del Release de GitHub.
2. Abre el paquete y sigue el instalador de macOS.
3. El paquete instala FFmpeg automaticamente si Homebrew ya esta disponible. Si no, instala Homebrew desde https://brew.sh y ejecuta `brew install ffmpeg`.
4. Inicia con `homereel`. Por defecto utiliza `~/HomeReel/media`.

El flujo de GitHub Actions `macOS package` genera ambos paquetes. Para distribuirlos sin advertencias de Gatekeeper, deben firmarse y notarizarse con una cuenta Apple Developer.

En los tres casos abre `http://localhost:8080`.

4. La primera carga escanea automáticamente. El botón **Actualizar** vuelve a indexar sólo los archivos cambiados. También puede invocarse `POST /api/v1/library/scan`.
5. Desde otro equipo de la misma LAN usa `http://IP-DEL-SERVIDOR:8080`.

Variables opcionales:

- `MEDIA_DIR`: directorio absoluto o relativo de la biblioteca (por defecto `./media`).
- `DB_PATH`: ubicación de SQLite (por defecto `./data/streamer.db`).
- `PORT`: puerto HTTP (por defecto `8080`).

El despliegue principal no usa Docker. Go mantiene las dependencias de compilación en `go.mod`; la única dependencia del sistema es FFmpeg (`ffprobe`), que el instalador de Windows verifica o instala con winget.

La pestaña **Biblioteca** reproduce contenido bajo demanda. `GET /api/v1/library` entrega el catálogo y `GET /api/v1/stream/{id}` acepta solicitudes HTTP Range, por lo que el reproductor puede avanzar y retroceder sin cargar el archivo completo.

La pestaña **TV en vivo** reproduce el ciclo completo del catálogo. Cada cliente obtiene el vídeo y segundo actual desde `GET /api/v1/tv/live`; se sincroniza al entrar, al acabar un vídeo y cada 30 segundos. No se genera ni transcodifica un stream adicional.

## Organización de biblioteca y control local

La interfaz agrupa automáticamente las rutas bajo `MEDIA_DIR`:

- `series/<serie>/<temporada>/<capitulo>.mp4`: serie, temporadas y capítulos.
- `series/<serie>/<capitulo>.mp4`: una única temporada ("Temporada 1").
- `peliculas/<pelicula>/<archivo>.mp4` o `peliculas/<archivo>.mp4`: películas y vídeos sueltos.

El botón **Admin** abre controles locales del canal compartido: seleccionar un vídeo, pausar, reanudar, buscar 30 segundos y volver a la programación automática. No hay autenticación ni sesiones: el estado pertenece al proceso del servidor y se aplica a todos los clientes. Sólo debe exponerse en una LAN de confianza.

Desde ese mismo panel el administrador puede cambiar la carpeta raíz de medios. En Windows, **Elegir carpeta** abre el selector nativo en el equipo que ejecuta el servidor; la selección se guarda en `data/settings.json` y se escanea inmediatamente. En Raspberry Pi/Linux se puede escribir la ruta montada, por ejemplo `/mnt/media`, y pulsar **Usar ruta**.

## Consola de administrador

Al ejecutar el servidor en una terminal, ésta acepta comandos locales que controlan el mismo canal compartido que la web:

```text
help
status
pause
play
seek 120
select 42
auto
media /mnt/media
scan
```

`media <ruta>` cambia y guarda la carpeta raíz, y después inicia un escaneo. Para administrar una Raspberry Pi mediante SSH, inicia el proceso en esa sesión o usa los mismos endpoints HTTP locales que consume la interfaz.

## Arranque opcional en Raspberry Pi/Linux

Si prefieres que el servidor arranque junto al sistema, crea un servicio `systemd`. Ajusta `User`, `WorkingDirectory`, `ExecStart` y `MEDIA_DIR` a tu instalación:

```ini
# /etc/systemd/system/homereel.service
[Unit]
Description=HomeReel
After=network.target

[Service]
Type=simple
User=pi
WorkingDirectory=/opt/homereel
Environment=MEDIA_DIR=/mnt/media
Environment=PORT=8080
ExecStart=/opt/homereel/homereel
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Después de guardar el archivo, activa el arranque automático con:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now homereel
```

Consulta el estado con `sudo systemctl status homereel`. Un servicio no tiene una consola interactiva; usa la interfaz web o las rutas HTTP de administración desde SSH para controlarlo.

El escaneo conserva los vídeos ausentes como no disponibles y evita invocar `ffprobe` si tamaño y fecha de modificación no cambiaron. Para archivos nuevos o modificados utiliza como máximo dos procesos `ffprobe`, adecuado para no saturar una Raspberry Pi 3.
