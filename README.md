## Requisitos Funcionales (RF)

Las acciones y comportamientos especÃ­ficos que el sistema debe ejecutar para el usuario final y los procesos internos.

1. **RF01 - Escaneo de Directorios:** El sistema debe recorrer directorios especÃ­ficos definidos en busca de archivos de video vÃ¡lidos (`.mp4`, `.mkv`).
2. **RF02 - ExtracciÃ³n de Metadatos TÃ©cnicos:** El sistema debe extraer la duraciÃ³n exacta y el cÃ³dec de video de cada archivo detectado utilizando `ffprobe`.
3. **RF03 - Escaneo Diferencial:** El sistema debe identificar si un archivo es nuevo, si fue modificado (tamaÃ±o/fecha) o si fue desconectado, actualizando el catÃ¡logo sin reprocesar archivos inalterados.
4. **RF04 - Streaming Personal (VOD):** El sistema debe servir archivos de video directamente al cliente permitiendo saltos de tiempo (*seek* / avance / retroceso) mediante peticiones HTTP parciales.
5. **RF05 - Motor de Canal de TV (Live Sync):** El sistema debe calcular en tiempo real un ciclo infinito de reproducciÃ³n y sincronizar a los clientes en el mismo segundo de emisiÃ³n, independientemente de cuÃ¡ndo se conecten.
6. **RF06 - API RESTful:** El sistema debe exponer endpoints HTTP para iniciar escaneos manuales, consultar el catÃ¡logo y obtener el estado del stream en vivo.
7. **RF07 - Interfaz de ReproducciÃ³n (Web UI):** El sistema debe proveer un cliente web en HTML/JS capaz de consumir la API, reproducir el video e integrarse con el motor del Canal de TV de forma autÃ³noma.
8. **RF08 - Resiliencia de Discos ExtraÃ­bles:** El sistema debe marcar los videos como "no disponibles" (`is_available = false`) en lugar de eliminarlos si el disco duro o pendrive de origen se desconecta.

## Requisitos No Funcionales (RNF)

Las restricciones de diseÃ±o, rendimiento y arquitectura obligatorias para que el sistema opere en el hardware objetivo.

1. **RNF01 - RestricciÃ³n de Hardware (Raspberry Pi 3):** El consumo de CPU debe mantenerse al mÃ­nimo mediante el uso de concurrencia controlada (*Worker Pools*), y el uso de memoria RAM no debe superar los 500 MB durante escaneos intensivos.
2. **RNF02 - Cero TranscodificaciÃ³n (Direct Play):** El servidor tiene estrictamente prohibido re-codificar video al vuelo. La carga de decodificaciÃ³n debe delegarse Ã­ntegramente al dispositivo cliente (navegador/Smart TV).
3. **RNF03 - Concurrencia de Base de Datos:** La lectura (streaming) y la escritura (escÃ¡ner) de datos deben ocurrir simultÃ¡neamente sin generar errores de tipo `database is locked`. Se exige el uso de SQLite en modo WAL (*Write-Ahead Logging*).
4. **RNF04 - Portabilidad CGO-Free:** El backend debe estar escrito enteramente en Go puro (usando `modernc.org/sqlite`), evitando dependencias en C para asegurar la compilaciÃ³n cruzada a ARM sin fricciÃ³n.
5. **RNF05 - Despliegue Simple:** El sistema debe poder instalarse como ejecutable nativo, con scripts que preparen FFmpeg y las dependencias de compilaciÃ³n.

## Módulos del Sistema y Mapeo de Requisitos

| MÃ³dulo | Responsabilidad Principal | Requisitos Satisfechos |
| --- | --- | --- |
| **1. Motor de Persistencia (`internal/repository`)** | Gestiona la conexiÃ³n a la base de datos SQLite y ejecuta operaciones CRUD con control de concurrencia y contexto. | RF03, RF08, RNF03, RNF04 |
| **2. Scanner & Metadata Engine (`internal/scanner`)** | Recorre directorios (O(1) para archivos conocidos) y administra un Worker Pool de `ffprobe` para no asfixiar el procesador. | RF01, RF02, RF03, RNF01 |
| **3. Streaming Engine (`internal/streaming`)** | Despacha bytes nativamente vÃ­a `http.ServeContent` para VOD y calcula algorÃ­tmicamente la posiciÃ³n actual del loop en vivo para TV. | RF04, RF05, RNF01, RNF02 |
| **4. API & Handlers (`internal/handler`)** | Expone y multiplexa las rutas HTTP, uniendo el motor de streaming, el repositorio y el escÃ¡ner con el exterior. | RF06 |
| **5. Cliente Frontend (`web/`)** | Consume los metadatos de la API, carga la interfaz nativa del reproductor (Video.js) y maneja la re-sincronizaciÃ³n. | RF07, RNF02 |

## Uso en LAN

La demo sirve vi­deo directamente, sin transcodificación ni dependencias de Internet. Para una reproducciÃ³n compatible con navegadores, Smart TV y Raspberry Pi, se recomienda MP4 con H.264/AAC.

### Windows

1. Clona o descarga el repositorio.
2. Haz doble clic en **`Instalar HomeReel.cmd`**. Instala Go/FFmpeg si faltan, compila el proyecto y abre el puerto LAN.
3. Haz doble clic en **`HomeReel.exe`**. La consola queda abierta mientras el servidor estÃ¡ activo.

### Raspberry Pi / Linux

1. Clona el repositorio.
2. Ejecuta `bash scripts/install-linux.sh`. El script instala Go y FFmpeg con `apt`, y compila el binario para la arquitectura local.
3. Inicia con `MEDIA_DIR=/mnt/media ./homereel`.

### macOS

1. Descarga `HomeReel-macos-arm64.pkg` para Apple Silicon o `HomeReel-macos-amd64.pkg` para Intel desde los artefactos de GitHub Actions.
2. Abre el paquete y sigue el instalador de macOS.
3. El paquete instala FFmpeg automaticamente si Homebrew ya esta disponible. Si no, instala Homebrew desde https://brew.sh y ejecuta `brew install ffmpeg`.
4. Inicia con `MEDIA_DIR="/Users/tuusuario/Media" homereel`.

El flujo de GitHub Actions `macOS package` genera ambos paquetes. Para distribuirlos sin advertencias de Gatekeeper, deben firmarse y notarizarse con una cuenta Apple Developer.

En los tres casos abre `http://localhost:8080`.

4. La primera carga escanea automÃ¡ticamente. El botÃ³n **Actualizar** vuelve a indexar sÃ³lo los archivos cambiados. TambiÃ©n puede invocarse `POST /api/v1/library/scan`.
5. Desde otro equipo de la misma LAN usa `http://IP-DEL-SERVIDOR:8080`.

Variables opcionales:

- `MEDIA_DIR`: directorio absoluto o relativo de la biblioteca (por defecto `./media`).
- `DB_PATH`: ubicaciÃ³n de SQLite (por defecto `./data/streamer.db`).
- `PORT`: puerto HTTP (por defecto `8080`).

El despliegue principal no usa Docker. Go mantiene las dependencias de compilaciÃ³n en `go.mod`; la Ãºnica dependencia del sistema es FFmpeg (`ffprobe`), que el instalador de Windows verifica o instala con winget.

La pestaÃ±a **Biblioteca** reproduce contenido bajo demanda. `GET /api/v1/library` entrega el catÃ¡logo y `GET /api/v1/stream/{id}` acepta solicitudes HTTP Range, por lo que el reproductor puede avanzar y retroceder sin cargar el archivo completo.

La pestaÃ±a **TV en vivo** reproduce el ciclo completo del catÃ¡logo. Cada cliente obtiene el vÃ­deo y segundo actual desde `GET /api/v1/tv/live`; se sincroniza al entrar, al acabar un vÃ­deo y cada 30 segundos. No se genera ni transcodifica un stream adicional.

## OrganizaciÃ³n de biblioteca y control local

La interfaz agrupa automÃ¡ticamente las rutas bajo `MEDIA_DIR`:

- `series/<serie>/<temporada>/<capitulo>.mp4`: serie, temporadas y capÃ­tulos.
- `series/<serie>/<capitulo>.mp4`: una Ãºnica temporada ("Temporada 1").
- `peliculas/<pelicula>/<archivo>.mp4` o `peliculas/<archivo>.mp4`: pelÃ­culas y vÃ­deos sueltos.

El botÃ³n **Admin** abre controles locales del canal compartido: seleccionar un vÃ­deo, pausar, reanudar, buscar 30 segundos y volver a la programaciÃ³n automÃ¡tica. No hay autenticaciÃ³n ni sesiones: el estado pertenece al proceso del servidor y se aplica a todos los clientes. SÃ³lo debe exponerse en una LAN de confianza.

Desde ese mismo panel el administrador puede cambiar la carpeta raÃ­z de medios. En Windows, **Elegir carpeta** abre el selector nativo en el equipo que ejecuta el servidor; la selecciÃ³n se guarda en `data/settings.json` y se escanea inmediatamente. En Raspberry Pi/Linux se puede escribir la ruta montada, por ejemplo `/mnt/media`, y pulsar **Usar ruta**.

## Consola de administrador

Al ejecutar el servidor en una terminal, Ã©sta acepta comandos locales que controlan el mismo canal compartido que la web:

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

`media <ruta>` cambia y guarda la carpeta raÃ­z, y despuÃ©s inicia un escaneo. Para administrar una Raspberry Pi mediante SSH, inicia el proceso en esa sesiÃ³n o usa los mismos endpoints HTTP locales que consume la interfaz.

## Arranque opcional en Raspberry Pi/Linux

Si prefieres que el servidor arranque junto al sistema, crea un servicio `systemd`. Ajusta `User`, `WorkingDirectory`, `ExecStart` y `MEDIA_DIR` a tu instalaciÃ³n:

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

DespuÃ©s de guardar el archivo, activa el arranque automÃ¡tico con:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now homereel
```

Consulta el estado con `sudo systemctl status homereel`. Un servicio no tiene una consola interactiva; usa la interfaz web o las rutas HTTP de administraciÃ³n desde SSH para controlarlo.

El escaneo conserva los vÃ­deos ausentes como no disponibles y evita invocar `ffprobe` si tamaÃ±o y fecha de modificaciÃ³n no cambiaron. Para archivos nuevos o modificados utiliza como mÃ¡ximo dos procesos `ffprobe`, adecuado para no saturar una Raspberry Pi 3.
