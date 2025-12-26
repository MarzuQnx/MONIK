#!/bin/bash

# Version Bump Script for MONIK-ENTERPRISE
# This script automates version bumping based on conventional commits

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to get current version
get_current_version() {
    if [ -f "go.mod" ]; then
        # For Go modules
        go mod edit -json | jq -r '.Module.Path' | sed 's/.*@//'
    elif [ -f "package.json" ]; then
        # For Node.js packages
        jq -r '.version' package.json
    else
        print_error "No package file found (go.mod or package.json)"
        exit 1
    fi
}

# Function to bump version
bump_version() {
    local current_version=$1
    local bump_type=$2
    
    # Parse version components
    IFS='.' read -ra VERSION_PARTS <<< "$current_version"
    local major=${VERSION_PARTS[0]}
    local minor=${VERSION_PARTS[1]}
    local patch=${VERSION_PARTS[2]}
    
    case $bump_type in
        "major")
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        "minor")
            minor=$((minor + 1))
            patch=0
            ;;
        "patch")
            patch=$((patch + 1))
            ;;
        *)
            print_error "Invalid bump type: $bump_type"
            exit 1
            ;;
    esac
    
    echo "${major}.${minor}.${patch}"
}

# Function to update version in files
update_version() {
    local new_version=$1
    
    print_info "Updating version to $new_version"
    
    # Update go.mod if exists
    if [ -f "go.mod" ]; then
        go mod edit -module "monik-enterprise@$new_version"
        print_success "Updated go.mod"
    fi
    
    # Update package.json if exists
    if [ -f "package.json" ]; then
        jq --arg version "$new_version" '.version = $version' package.json > package.json.tmp && mv package.json.tmp package.json
        print_success "Updated package.json"
    fi
    
    # Update .env.example if exists
    if [ -f ".env.example" ]; then
        sed -i "s/^APP_VERSION=.*/APP_VERSION=$new_version/" .env.example
        print_success "Updated .env.example"
    fi
}

# Function to detect version bump type from commits
detect_bump_type() {
    local commits_since_last_tag
    
    # Get commits since last tag
    if git describe --tags --abbrev=0 2>/dev/null; then
        local last_tag=$(git describe --tags --abbrev=0)
        commits_since_last_tag=$(git log $last_tag..HEAD --oneline)
    else
        commits_since_last_tag=$(git log --oneline)
    fi
    
    # Analyze commit messages
    local has_breaking=false
    local has_feat=false
    local has_fix=false
    
    while IFS= read -r commit; do
        if [[ $commit =~ ^breaking: ]] || [[ $commit =~ BREAKING[[:space:]]CHANGE ]]; then
            has_breaking=true
        elif [[ $commit =~ ^feat: ]]; then
            has_feat=true
        elif [[ $commit =~ ^fix: ]]; then
            has_fix=true
        fi
    done <<< "$commits_since_last_tag"
    
    if [ "$has_breaking" = true ]; then
        echo "major"
    elif [ "$has_feat" = true ]; then
        echo "minor"
    else
        echo "patch"
    fi
}

# Main execution
main() {
    print_info "Starting version bump process..."
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        print_error "Not in a git repository"
        exit 1
    fi
    
    # Get current version
    current_version=$(get_current_version)
    print_info "Current version: $current_version"
    
    # Detect bump type
    bump_type=$(detect_bump_type)
    print_info "Detected bump type: $bump_type"
    
    # Calculate new version
    new_version=$(bump_version "$current_version" "$bump_type")
    print_info "New version: $new_version"
    
    # Update version in files
    update_version "$new_version"
    
    # Create git tag
    git tag "v$new_version"
    print_success "Created git tag: v$new_version"
    
    # Create commit with version bump
    git add .
    git commit -m "chore(version): bump version to $new_version [skip ci]"
    print_success "Created version bump commit"
    
    print_success "Version bump completed successfully!"
    print_info "New version: $new_version"
    print_info "Git tag: v$new_version"
}

# Run main function
main "$@"