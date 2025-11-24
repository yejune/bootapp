# docker-bootapp

Docker CLI Plugin for multi-project Docker networking made easy.

Automatically manages:
- Unique subnet allocation per project (prevents conflicts)
- /etc/hosts entries for containers with DOMAIN configuration
- **SSL certificates** for domains in `SSL_DOMAINS` (auto-generated, system trusted)
- Smart route setup for macOS (checks connectivity before adding routes)

## Installation

### Quick Install
```bash
make build install
```

### Manual Install
```bash
make build
cp build/docker-bootapp ~/.docker/cli-plugins/docker-bootapp
chmod +x ~/.docker/cli-plugins/docker-bootapp
```

## Usage

### Start containers
```bash
# Auto-detect docker-compose.yml
docker bootapp up

# Specify compose file
docker bootapp -f docker-compose.local.yml up
```

If multiple compose files are found, you'll be prompted to select:
```
Multiple compose files found:
  [1] docker-compose.yml
  [2] docker-compose.local.yml
  [3] docker-compose.prod.yml

Select file (1-3):
```

Supported file patterns:
- `docker-compose.yml`, `docker-compose.yaml`
- `docker-compose.*.yml`, `docker-compose.*.yaml` (e.g., docker-compose.local.yml)
- `compose.yml`, `compose.yaml`

Options:
- `-d, --detach`: Run in background (default: true)
- `--no-build`: Don't build images
- `--pull`: Pull images before starting
- `-F, --force-recreate`: Force recreate containers + regenerate SSL certificates

This will:
1. Allocate unique subnet for the project (172.18-31.x.x range)
2. Parse docker-compose file for DOMAIN/SSL_DOMAINS configuration
3. **Generate SSL certificates** for `SSL_DOMAINS` (if not exists)
4. **Install certificates to system trust store** (macOS Keychain / Linux ca-certificates)
5. Start containers with docker-compose up
6. Discover container IPs from default compose network
7. Add /etc/hosts entries for containers with domain config
8. Setup routing if needed (macOS)

### Stop containers
```bash
docker bootapp down
docker bootapp -f docker-compose.local.yml down
```

Options:
- `-v, --volumes`: Remove volumes
- `--remove-orphans`: Remove orphan containers
- `--keep-hosts`: Keep /etc/hosts entries
- `--remove-config`: Remove project from global config

### List projects
```bash
docker bootapp ls
```

## Domain Configuration

### Supported Environment Variables

All of these environment variables are used (duplicates removed, each supports single, comma, or newline separated values):
- `DOMAIN`
- `DOMAINS`
- `SSL_DOMAINS`
- `APP_DOMAIN`
- `VIRTUAL_HOST` (nginx-proxy compatible)

```yaml
services:
  app:
    image: nginx
    environment:
      SSL_DOMAINS: |
        myapp.local
        www.myapp.local

  mysql:
    image: mysql:8
    environment:
      DOMAIN: mysql.myapp.local

  redis:
    image: redis
    # No DOMAIN = no /etc/hosts entry (IP only)
```

### Traefik Labels

Traefik router rules are also supported:

```yaml
services:
  web:
    image: nginx
    labels:
      - "traefik.http.routers.web.rule=Host(`web.local`)"
      # Multiple hosts supported:
      - "traefik.http.routers.api.rule=Host(`api.local`) || Host(`api2.local`)"
      # Or comma-separated:
      - "traefik.http.routers.app.rule=Host(`app.local`, `www.app.local`)"
```

### Result

Only services with explicit domain configuration get /etc/hosts entries:

```
172.18.0.2    myapp.local        ## docker-bootapp:myproject
172.18.0.2    www.myapp.local    ## docker-bootapp:myproject
172.18.0.3    mysql.myapp.local  ## docker-bootapp:myproject
```

Services without DOMAIN config (like redis above) are not added to /etc/hosts.

## SSL Certificates

### Automatic Generation

bootapp automatically generates self-signed SSL certificates for domains specified in `SSL_DOMAINS`:

```yaml
services:
  app:
    image: nginx
    environment:
      SSL_DOMAINS: myapp.test
    volumes:
      - ./var/certs:/etc/nginx/certs:ro
```

Certificates are:
- Generated in `./var/certs/` directory (`.crt`, `.key`, `.pem` files)
- Automatically trusted in system keychain (macOS) or ca-certificates (Linux)
- Valid for 10 years
- Include proper SAN (Subject Alternative Name) for browser compatibility

### Certificate Files

```
var/certs/
├── myapp.test.crt    # Certificate
├── myapp.test.key    # Private key
└── myapp.test.pem    # Combined cert + key
```

### Force Regenerate

To delete and regenerate certificates:

```bash
docker bootapp -f docker-compose.local.yml up -F
```

The `-F` flag will:
1. Remove existing certificates from trust store
2. Delete local certificate files
3. Generate new certificates
4. Install to trust store
5. Force recreate containers

### nginx Configuration Example

```nginx
server {
    listen 443 ssl;
    server_name myapp.test;

    ssl_certificate /etc/nginx/certs/myapp.test.crt;
    ssl_certificate_key /etc/nginx/certs/myapp.test.key;

    # ... rest of config
}
```

## macOS Networking

**docker-mac-net-connect is required** for direct container IP access on macOS.

Docker Desktop runs containers inside a Linux VM, so macOS cannot directly access container IPs without a network tunnel.

### Installation
```bash
brew install chipmk/tap/docker-mac-net-connect
sudo brew services start docker-mac-net-connect
```

bootapp will check for docker-mac-net-connect and show installation instructions if not found.

## Linux

No additional setup needed - Docker networking works natively.

## Domain TLD Recommendations

**Recommended TLDs:**
- `.test` - RFC 2606 reserved for testing ✅
- `.localhost` - Local only ✅
- `.internal` - Private networks ✅

**Avoid:**
- `.local` - Conflicts with macOS mDNS (slow DNS lookups)
- `.dev` - Google-owned, forces HTTPS
- `.app` - Google-owned, forces HTTPS

## Configuration

Global configuration is stored in `~/.docker-bootapp/projects.json`:

```json
{
  "myproject": {
    "path": "/path/to/project",
    "subnet": "172.18.0.0/16",
    "domain": "myproject.local"
  },
  "another-project": {
    "path": "/path/to/another",
    "subnet": "172.19.0.0/16",
    "domain": "another.local"
  }
}
```

Each project gets a unique subnet (172.18.x.x through 172.31.x.x) to prevent IP conflicts between projects.

## License

MIT
