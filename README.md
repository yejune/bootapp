# bootapp

Docker CLI Plugin for multi-project Docker networking made easy.

Automatically manages:
- Unique subnet allocation per project (prevents conflicts)
- /etc/hosts entries for containers with DOMAIN configuration
- **SSL certificates** for domains in `SSL_DOMAINS` (auto-generated, system trusted)
- Smart route setup for macOS (checks connectivity before adding routes)

## Installation

### Method 1: Using Homebrew (macOS)

```bash
brew install yejune/tap/bootapp
```

Homebrew automatically:
- Downloads and builds the latest version
- Installs as Docker CLI plugin (`docker bootapp`)
- Installs standalone binary (`bootapp`)
- Checks dependencies

### Method 2: Using install script (Linux/macOS)

```bash
# Clone the repository
git clone https://github.com/yejune/bootapp.git
cd bootapp

# Run install script
bash install.sh
```

The install script automatically:
- Checks for Go and Docker
- Builds the binary
- Installs to `~/.docker/cli-plugins/bootapp` (Docker plugin)
- Installs to `/usr/local/bin/bootapp` (standalone binary)
- Checks platform-specific dependencies

### Method 3: Using go install
```bash
go install github.com/yejune/bootapp@latest
bootapp install
```

Or build locally:
```bash
git clone https://github.com/yejune/bootapp.git
cd bootapp
go build
./bootapp install
```

The `install` command automatically:
- Copies binary to `~/.docker/cli-plugins/bootapp`
- Installs standalone binary to `/usr/local/bin/bootapp`
- Sets executable permissions
- Checks platform dependencies

### Method 4: Manual installation
```bash
make build
cp build/bootapp ~/.docker/cli-plugins/bootapp
chmod +x ~/.docker/cli-plugins/bootapp
sudo cp build/bootapp /usr/local/bin/bootapp
sudo chmod +x /usr/local/bin/bootapp
```

## Usage

### Start containers
```bash
# Auto-detect docker-compose.yml
docker bootapp up

# Specify compose file
docker bootapp -f docker-compose.local.yml up
```

If multiple compose files are found, you'll be prompted to select interactively:
```
Select compose file:
▸ docker-compose.yml
  docker-compose.local.yml
  docker-compose.prod.yml

Use ↑/↓ arrows to navigate, Enter to select
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

All of these environment variables are used for both:
- **Host machine access** (/etc/hosts)
- **Container-to-container access** (Docker network aliases)

Supported variables (duplicates removed, each supports single, comma, space, or newline separated values):
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

  db:
    image: mysql:8
    environment:
      DOMAIN: db.local db-backup.local

  redis:
    image: redis
    # No DOMAIN = no /etc/hosts entry (IP only)
```

### Container-to-Container Communication

Domains set via `DOMAIN`/`DOMAINS` are automatically registered as Docker network aliases, allowing containers to reach each other:

```yaml
services:
  db:
    image: mariadb
    environment:
      DOMAIN: db.local db-delivery.local

  api:
    image: nginx
    environment:
      DOMAIN: api.local
    # api can reach db via: db.local or db-delivery.local
    # db can reach api via: api.local
```

This replaces the deprecated `external_links` and works automatically with Docker's built-in DNS.

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
## bootapp:myproject
172.18.0.2    myapp.local
## bootapp:myproject
172.18.0.2    www.myapp.local
## bootapp:myproject
172.18.0.3    mysql.myapp.local
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

✅ **All features work natively on Linux!**

Linux support includes:

1. **Docker Networking**
   - Unique subnet per container
   - Direct container IP access (no additional tools needed)

2. **SSL Certificate Auto-generation & Trust**
   - Debian/Ubuntu: `update-ca-certificates`
   - RHEL/CentOS: `update-ca-trust`
   - Self-signed certificates automatically trusted system-wide

3. **Automatic /etc/hosts Management**
   - Domain → Container IP mapping
   - Auto register/cleanup per project

4. **Standalone Binary + Docker Plugin**
   - Use `bootapp` or `docker bootapp` commands

**Installation:**
```bash
# Requires Go
bash install.sh
# or
make build install
```

Unlike macOS, Linux doesn't need additional network tools (`docker-mac-net-connect`) - everything works out of the box!

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

Global configuration is stored in `~/.bootapp/projects.json`:

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
