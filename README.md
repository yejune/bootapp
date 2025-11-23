# docker-bootapp

Docker CLI Plugin for multi-project Docker networking made easy.

Automatically manages:
- /etc/hosts entries for containers with DOMAIN configuration
- Route setup for macOS (auto-detect docker-mac-net-connect or manual route)

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

This will:
1. Parse docker-compose file for DOMAIN/SSL_DOMAINS configuration
2. Start containers with docker-compose up
3. Discover container IPs from default compose network
4. Add /etc/hosts entries for containers with domain config
5. Setup routing if needed (macOS)

### Stop containers
```bash
docker bootapp down
docker bootapp -f docker-compose.local.yml down
```

Options:
- `-v, --volumes`: Remove volumes
- `--remove-orphans`: Remove orphan containers
- `--keep-hosts`: Keep /etc/hosts entries

### List projects
```bash
docker bootapp ls
```

## Domain Configuration

### Supported Environment Variables

In docker-compose.yml, use any of these:
- `DOMAIN`: Single domain
- `DOMAINS`: Comma or newline separated domains
- `SSL_DOMAINS`: Comma or newline separated domains

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

### Result

Only services with explicit domain configuration get /etc/hosts entries:

```
192.168.156.2    myapp.local        ## docker-bootapp:myproject
192.168.156.2    www.myapp.local    ## docker-bootapp:myproject
192.168.156.3    mysql.myapp.local  ## docker-bootapp:myproject
```

Services without DOMAIN config (like redis above) are not added to /etc/hosts.

## macOS Networking

For direct container IP access on macOS:

### Option 1: docker-mac-net-connect (Recommended)
```bash
brew install chipmk/tap/docker-mac-net-connect
sudo brew services start docker-mac-net-connect
```

### Option 2: Manual Route
If docker-mac-net-connect is not running, bootapp will attempt to set up routing automatically.

## Linux

No additional setup needed - Docker networking works natively.

## Configuration Files

### Local: `{project}/.docker/network.json`

```json
{
  "project": "myproject",
  "subnet": "172.18.0.0/16",
  "containers": {
    "app": {
      "domains": ["myapp.local", "www.myapp.local"],
      "ip": "192.168.156.2"
    },
    "mysql": {
      "domains": ["mysql.myapp.local"],
      "ip": "192.168.156.3"
    },
    "redis": {
      "ip": "192.168.156.4"
    }
  }
}
```

## License

MIT
