#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Linux*)     OS_TYPE=linux;;
    Darwin*)    OS_TYPE=macos;;
    *)          OS_TYPE="unknown";;
esac

echo "🚀 Installing bootapp..."
echo ""

# Request sudo access upfront (for later /usr/local/bin installation)
echo "📋 Requesting sudo access for installation..."
sudo -v
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${NC}"
    echo "Please install Go from https://golang.org/dl/"
    exit 1
fi

echo "✓ Go found: $(go version)"

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    echo ""
    echo "Please install one of the following:"
    echo "  • Docker Desktop: https://www.docker.com/products/docker-desktop"
    if [ "$OS_TYPE" = "macos" ]; then
        echo "  • OrbStack: https://orbstack.dev"
        echo "  • Colima: brew install colima"
    fi
    exit 1
fi

echo "✓ Docker found: $(docker --version)"

# Detect container runtime using docker info (context show returns "default" for OrbStack)
RUNTIME="unknown"
OS_INFO=$(docker info --format '{{.OperatingSystem}}' 2>/dev/null)
if echo "$OS_INFO" | grep -iq "orbstack"; then
    RUNTIME="OrbStack"
elif echo "$OS_INFO" | grep -iq "colima"; then
    RUNTIME="Colima"
else
    RUNTIME="Docker Desktop"
fi

if [ "$RUNTIME" != "unknown" ]; then
    echo "✓ Using container runtime: $RUNTIME"
fi

# Build the binary
echo "📦 Building bootapp..."
go build -o build/bootapp .

if [ ! -f "build/bootapp" ]; then
    echo -e "${RED}Error: Build failed${NC}"
    exit 1
fi

echo "✓ Build successful"

# Create Docker CLI plugins directory
PLUGIN_DIR="$HOME/.docker/cli-plugins"
mkdir -p "$PLUGIN_DIR"

# Copy binary to Docker CLI plugins directory
echo "📋 Installing to $PLUGIN_DIR..."
cp build/bootapp "$PLUGIN_DIR/bootapp"
chmod +x "$PLUGIN_DIR/bootapp"

# Also install as standalone binary
echo ""
echo "📋 Installing standalone binary to /usr/local/bin/bootapp..."
if sudo cp build/bootapp /usr/local/bin/bootapp 2>/dev/null && sudo chmod +x /usr/local/bin/bootapp 2>/dev/null; then
    echo "✓ Standalone binary installed (you can use 'bootapp' command)"
else
    echo -e "${YELLOW}⚠️  Warning: Could not install standalone binary${NC}"
    echo "   You can still use 'docker bootapp' commands"
    echo "   Or manually install: sudo cp build/bootapp /usr/local/bin/bootapp"
fi

echo -e "${GREEN}✓ bootapp installed successfully!${NC}"
echo ""

# Verify installation
echo "✓ Verifying installation..."
if docker bootapp help > /dev/null 2>&1; then
    echo ""
    docker bootapp help
else
    echo -e "${YELLOW}Warning: Unable to verify installation${NC}"
    echo "Try running: docker bootapp help"
fi

# Platform specific checks
echo ""
if [ "$OS_TYPE" = "macos" ]; then
    echo "🍎 macOS detected - checking dependencies..."

    # OrbStack has built-in network support, no need for docker-mac-net-connect
    if [ "$RUNTIME" = "OrbStack" ]; then
        echo "✓ OrbStack has built-in network support"
    else
        if ! command -v docker-mac-net-connect &> /dev/null; then
            echo -e "${YELLOW}⚠️  docker-mac-net-connect is NOT installed${NC}"
            echo ""
            echo "On macOS with $RUNTIME, docker-mac-net-connect is recommended"
            echo "to access container IPs directly."
            echo ""
            echo "Install with:"
            echo "  brew install chipmk/tap/docker-mac-net-connect"
            echo "  sudo brew services start docker-mac-net-connect"
            echo ""
        else
            echo "✓ docker-mac-net-connect is installed"

            # Check if process is actually running (more reliable than brew services)
            if pgrep -f docker-mac-net-connect > /dev/null 2>&1; then
                echo "✓ docker-mac-net-connect service is running"
            else
                echo -e "${YELLOW}⚠️  docker-mac-net-connect is installed but not running${NC}"
                echo ""
                echo "Start the service with:"
                echo "  sudo brew services start docker-mac-net-connect"
                echo ""
            fi
        fi
    fi
elif [ "$OS_TYPE" = "linux" ]; then
    echo "🐧 Linux detected"
    echo "✓ Container IPs are directly accessible on Linux"
    echo "✓ SSL certificate trust supported (update-ca-certificates/update-ca-trust)"
    echo "✓ /etc/hosts management supported"
fi

echo ""
echo -e "${GREEN}🎉 Installation complete!${NC}"
