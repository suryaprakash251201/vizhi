# Vizhi — Architecture & Implementation Guide

## ASCII Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                    FLUTTER CLIENT (Android/iOS/Desktop)          │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐  │
│  │ Riverpod     │  │ Dio (REST)   │  │ WebSocket Channel     │  │
│  │ State Mgmt   │◄─┤ GET/POST     │◄─┤ (gorilla/websocket)   │  │
│  │ 3 Providers  │  │ File Upload  │  │ Real-time stats push  │  │
│  └──────┬───────┘  └──────┬───────┘  └───────────┬────────────┘  │
│         │                 │                      │               │
│         └─────────────────┴──────────────────────┘               │
└────────────────────────────┬─────────────────────────────────────┘
                             │ HTTPS / WSS (TLS 1.3)
                             │ JWT Bearer Token in Authorization header
                             ▼
┌──────────────────────────────────────────────────────────────────┐
│                    CLOUDFLARE TUNNEL (optional)                  │
│              cloudflared -> localhost:8443 (originates TLS)      │
└──────────────────────────────────────┬───────────────────────────┘
                                       │
┌──────────────────────────────────────┴───────────────────────────┐
│              DOCKER CONTAINER (vizhi-backend)                     │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  Go HTTP Server (chi router)                               │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐  │  │
│  │  │ Auth     │ │ Monitor  │ │ File     │ │ WebSocket    │  │  │
│  │  │ JWT+BCrypt│ │ gopsutil │ │ Transfer │ │ Broadcast    │  │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────────┘  │  │
│  │  ┌──────────────────────────────────────────────────────┐  │  │
│  │  │ App Lifecycle (whitelist-based process control)      │  │  │
│  │  └──────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────┘  │
└───────────────────────┬──────────────────────────────────────────┘
                        │
    ┌───────────────────┼───────────────────────┐
    │                   │                       │
    ▼                   ▼                       ▼
┌──────────┐  ┌──────────────┐  ┌──────────────────────────┐
│ /proc    │  │ D-Bus Socket │  │ /usr/bin, /snap/bin      │
│ (host ns)│  │ /run/user/   │  │ Host PID namespace       │
│ CPU/MEM  │  │ 1000/bus     │  │ Process launch/terminate │
└──────────┘  └──────────────┘  └──────────────────────────┘
    │               │                    │
    └───────────────┴────────────────────┘
                    │
                    ▼
        ┌─────────────────────┐
        │  LINUX HOST OS      │
        │  (Ubuntu Server /   │
        │   Fedora)           │
        └─────────────────────┘
```

## Container → Host Access Strategy

The container gains host access via these Docker configs:

| Capability | Mechanism | Security Notes |
|---|---|---|
| Process metrics | `pid: "host"` — shares host PID ns | Read-only; gopsutil can't write |
| Process killing | `pid: "host"` + gopsutil | Whitelist restricts which apps can be killed |
| Launching GUI apps | `--pid=host` + D-Bus socket mount + X11 socket | Binary name must match allowlist exactly |
| Host filesystem | `/proc` mounted `:ro` | Read-only metric collection |
| D-Bus session | `/run/user/1000/bus` bind mount | Only the session bus, not system bus |
| Network | `network_mode: "host"` | Drops isolation — use only on trusted LAN or behind Cloudflare Tunnel |

**Recommended alternative** (more secure): Run `network_mode: bridge`, port-map 8443,
and forward only the necessary sockets. However, D-Bus and X11 paths become brittle
across distros. For simplicity, host networking is used here.

## Security Model

### Authentication Flow
```
Client                          Server
  │                                │
  │  POST /auth/login              │
  │  { password: "..." }           │
  │───────────────────────────────>│
  │                                │  bcrypt.CompareHashAndPassword
  │                                │  jwt.Generate(role="admin", exp=12h)
  │  { token: "eyJ...",            │
  │    expires_at: 1712345678 }     │
  │<───────────────────────────────│
  │                                │
  │  GET /api/v1/stats             │
  │  Authorization: Bearer eyJ...  │
  │───────────────────────────────>│
  │                                │  jwt.ParseWithClaims
  │  { cpu: {...}, memory: ... }   │
  │<───────────────────────────────│
```

### Risks flagged

1. **`network_mode: "host"`** exposes the Go server directly on the host network.
   Mitigation: `VIZHI_HOST=127.0.0.1` + Cloudflare Tunnel or Tailscale, never bind
   `0.0.0.0` on a public IP.

2. **Process launch** requires D-Bus + X11 to work for GUI apps. The container runs
   with `read_only: true` and `cap_drop: ALL` to limit blast radius.

3. **No arbitrary command execution** — the code never uses `bash -c` or shell
   expansion. The whitelist approach means only predefined binary names (e.g.
   "firefox", "code") are accepted. `exec.LookPath` resolves the binary path
   safely before `exec.Command`.

4. **TLS is strongly recommended** — if disabled, the server prints a loud warning
   every startup. The Flutter client should only connect over HTTPS/WSS.

## Library Choices

| Purpose | Go Library | Flutter Library |
|---|---|---|
| HTTP routing | `go-chi/chi/v5` | `dio` |
| WebSocket | `gorilla/websocket` | `web_socket_channel` |
| System metrics | `shirou/gopsutil/v3` | — |
| Auth (JWT) | `golang-jwt/jwt/v5` + `golang.org/x/crypto/bcrypt` | `flutter_secure_storage` |
| File transfer | Chunked multipart + SHA-256 | `file_picker` |
| GUI app control | `os/exec` + D-Bus env | — |
| UUID | `google/uuid` | — |
| State management | — | `flutter_riverpod` |

## Deployment

### 1. Build Go backend

```bash
cd backend
CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/vizhi-server ./cmd/vizhi-server
```

### 2. Generate TLS cert (self-signed for dev)

```bash
mkdir -p tls
openssl req -x509 -nodes -days 365 -newkey rsa:4096 \
  -keyout tls/server.key -out tls/server.crt \
  -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:192.168.1.100"
```

### 3. Configure and run with Docker

```bash
# Generate bcrypt hash offline
go run cmd/gen-hash/main.go "your-master-password"

# Set env and start
export VIZHI_JWT_SECRET=$(openssl rand -hex 32)
export VIZHI_MASTER_PASSWORD_HASH='$2a$10$...'  # from gen-hash
docker compose up -d
```

### 4. Run Flutter app

```bash
cd frontend
flutter run -d <device-id>
```

Connect to `https://<host-ip>:8443` from the login screen.

### 5. Production hardening

- Set `VIZHI_HOST=127.0.0.1` and front with Cloudflare Tunnel (`cloudflared tunnel --url http://localhost:8443`)
- Or use Tailscale Funnel for private networking
- Set `VIZHI_TLS_ENABLED=true` with real certs (Let's Encrypt via certbot)
- Rotate `VIZHI_JWT_SECRET` regularly
- Keep `VIZHI_ALLOWED_APPS` minimal — only the specific binaries you need

## API Endpoints

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Health check (no auth) |
| POST | `/auth/login` | Get JWT token (no auth) |
| GET | `/api/v1/stats` | Full system stats |
| GET | `/api/v1/stats/stream` | WebSocket — real-time push |
| GET | `/api/v1/files` | List uploaded files |
| POST | `/api/v1/files/upload/init` | Start chunked upload session |
| POST | `/api/v1/files/upload/chunk` | Upload one chunk (multipart) |
| POST | `/api/v1/files/upload/complete` | Finalize upload, assemble chunks |
| GET | `/api/v1/files/download/{path}` | Download a file |
| DELETE | `/api/v1/files/{path}` | Delete a file |
| GET | `/api/v1/apps` | List allowed apps with status |
| POST | `/api/v1/apps/launch` | Launch an allowed app |
| POST | `/api/v1/apps/terminate` | Terminate a running app |
