#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Get current version from Makefile
CURRENT_VERSION=$(grep "^VERSION=" Makefile | cut -d'=' -f2)

echo -e "${BLUE}🚀 Docker Bootapp Release Script${NC}"
echo ""
echo -e "Current version: ${GREEN}${CURRENT_VERSION}${NC}"
echo ""

# Parse version components
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

# Offer version bump options
echo "Select version bump:"
echo "  1) Patch (${MAJOR}.${MINOR}.$((PATCH + 1)))"
echo "  2) Minor (${MAJOR}.$((MINOR + 1)).0)"
echo "  3) Major ($((MAJOR + 1)).0.0)"
echo "  4) Custom version"
echo ""
read -p "Enter choice (1-4): " CHOICE

case $CHOICE in
  1)
    NEW_VERSION="${MAJOR}.${MINOR}.$((PATCH + 1))"
    ;;
  2)
    NEW_VERSION="${MAJOR}.$((MINOR + 1)).0"
    ;;
  3)
    NEW_VERSION="$((MAJOR + 1)).0.0"
    ;;
  4)
    read -p "Enter new version: " NEW_VERSION
    ;;
  *)
    echo -e "${RED}Invalid choice${NC}"
    exit 1
    ;;
esac

echo ""
echo -e "${YELLOW}Releasing version: ${NEW_VERSION}${NC}"
echo ""

# Confirm
read -p "Continue? (y/N): " CONFIRM
if [[ ! $CONFIRM =~ ^[Yy]$ ]]; then
    echo "Aborted"
    exit 0
fi

echo ""
echo -e "${BLUE}Step 1: Updating version in files...${NC}"

# Update Makefile
sed -i '' "s/^VERSION=.*/VERSION=${NEW_VERSION}/" Makefile
echo "✓ Updated Makefile"

# Commit version bump
git add Makefile
git commit -m "chore: bump version to ${NEW_VERSION}"
echo "✓ Committed version bump"

echo ""
echo -e "${BLUE}Step 2: Creating git tag...${NC}"

# Create annotated tag
git tag -a "v${NEW_VERSION}" -m "Release v${NEW_VERSION}"
echo "✓ Created tag v${NEW_VERSION}"

# Push commits and tags
git push origin main
git push origin "v${NEW_VERSION}"
echo "✓ Pushed to GitHub"

echo ""
echo -e "${BLUE}Step 3: Calculating SHA256...${NC}"

# Wait a moment for GitHub to process the release
sleep 3

# Download tarball and calculate SHA256
TARBALL_URL="https://github.com/yejune/docker-bootapp/archive/refs/tags/v${NEW_VERSION}.tar.gz"
TARBALL_FILE="/tmp/docker-bootapp-${NEW_VERSION}.tar.gz"

curl -L -o "$TARBALL_FILE" "$TARBALL_URL"
SHA256=$(shasum -a 256 "$TARBALL_FILE" | awk '{print $1}')

echo "✓ Downloaded tarball"
echo "✓ SHA256: ${SHA256}"

echo ""
echo -e "${BLUE}Step 4: Updating Homebrew formula...${NC}"

# Update docker-bootapp.rb
sed -i '' "s|url \"https://github.com/yejune/docker-bootapp/archive/refs/tags/v.*\.tar\.gz\"|url \"https://github.com/yejune/docker-bootapp/archive/refs/tags/v${NEW_VERSION}.tar.gz\"|" docker-bootapp.rb
sed -i '' "s/sha256 \".*\"/sha256 \"${SHA256}\"/" docker-bootapp.rb

echo "✓ Updated docker-bootapp.rb"

# Commit formula update
git add docker-bootapp.rb
git commit -m "chore: update Homebrew formula to v${NEW_VERSION}"
git push origin main
echo "✓ Pushed formula update"

echo ""
echo -e "${BLUE}Step 5: Updating homebrew-tap...${NC}"

# Update homebrew-tap repository
cd /tmp
rm -rf homebrew-tap
mkdir homebrew-tap
cd homebrew-tap
git init

# Copy updated formula
cp /Users/max/Work/tmp/docker-bootapp/docker-bootapp.rb .

# Commit and push
git add docker-bootapp.rb
git commit -m "Update docker-bootapp to v${NEW_VERSION}"
git remote add origin https://github.com/yejune/homebrew-tap.git
git branch -M main
git push -f origin main

echo "✓ Updated homebrew-tap"

echo ""
echo -e "${GREEN}🎉 Release v${NEW_VERSION} completed!${NC}"
echo ""
echo "Users can now install/upgrade with:"
echo "  brew update"
echo "  brew upgrade docker-bootapp"
echo ""
echo "Or fresh install:"
echo "  brew install yejune/tap/docker-bootapp"
