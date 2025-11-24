# docker-bootapp

Docker CLI Plugin for multi-project Docker networking made easy.

Automatically manages:
- Unique subnet allocation per project (prevents conflicts)
- /etc/hosts entries for containers with DOMAIN configuration
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

This will:
1. Allocate unique subnet for the project (172.18-31.x.x range)
2. Parse docker-compose file for DOMAIN/SSL_DOMAINS configuration
3. Start containers with docker-compose up
4. Discover container IPs from default compose network
5. Add /etc/hosts entries for containers with domain config
6. Setup routing if needed (macOS)

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

## macOS Networking

For direct container IP access on macOS:

### Option 1: docker-mac-net-connect (Recommended)
```bash
brew install chipmk/tap/docker-mac-net-connect
sudo brew services start docker-mac-net-connect
```

### Option 2: Automatic Route
If no working route exists, bootapp will automatically:
1. Check if route already exists and test connectivity
2. If not working, add route through Docker VM gateway
3. Skip if connectivity is already working

## Linux

No additional setup needed - Docker networking works natively.

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
