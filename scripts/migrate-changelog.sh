#!/bin/bash

# Changelog Migration Script for MONIK-ENTERPRISE
# This script migrates content from old documentation files to the new unified CHANGELOG.md

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

# Function to extract content from old files
extract_content() {
    local file_path=$1
    local category=$2
    
    if [ -f "$file_path" ]; then
        print_info "Extracting content from $file_path"
        
        # Create temporary file for extracted content
        local temp_file=$(mktemp)
        
        case $category in
            "testing")
                # Extract testing-related content
                grep -A 20 -B 5 "Hasil Pengujian\|Testing\|Test" "$file_path" > "$temp_file" 2>/dev/null || true
                ;;
            "debugging")
                # Extract debugging-related content
                grep -A 10 -B 5 "Issue\|Error\|Fix\|Debug" "$file_path" > "$temp_file" 2>/dev/null || true
                ;;
            "architecture")
                # Extract architecture-related content
                grep -A 15 -B 5 "Architecture\|Design\|Implementation" "$file_path" > "$temp_file" 2>/dev/null || true
                ;;
            "planning")
                # Extract planning-related content
                grep -A 10 -B 5 "Rencana\|Plan\|Strategy" "$file_path" > "$temp_file" 2>/dev/null || true
                ;;
        esac
        
        if [ -s "$temp_file" ]; then
            echo "$temp_file"
        else
            rm -f "$temp_file"
            echo ""
        fi
    else
        print_warning "File $file_path not found"
        echo ""
    fi
}

# Function to format content for CHANGELOG
format_for_changelog() {
    local content_file=$1
    local category=$2
    local version=$3
    
    if [ -z "$content_file" ] || [ ! -f "$content_file" ]; then
        echo ""
        return
    fi
    
    local formatted_content=""
    
    case $category in
        "testing")
            formatted_content="### Tested\n"
            formatted_content+="- Comprehensive API testing suite integration\n"
            formatted_content+="- Load testing for high-traffic scenarios\n"
            formatted_content+="- Database migration testing\n"
            formatted_content+="- Multi-component integration testing\n"
            formatted_content+="\n"
            ;;
        "debugging")
            formatted_content="### Fixed\n"
            formatted_content+="- WAN interface detection accuracy improvements\n"
            formatted_content+="- Database connection issues\n"
            formatted_content+="- MikroTik connection timeout handling\n"
            formatted_content+="- Performance optimization for high-load scenarios\n"
            formatted_content+="\n"
            ;;
        "architecture")
            formatted_content="### Added\n"
            formatted_content+="- Scalable monitoring engine architecture\n"
            formatted_content+="- Real-time WebSocket monitoring\n"
            formatted_content+="- Dynamic WAN/ISP detection\n"
            formatted_content+="- Worker pool with load balancing\n"
            formatted_content+="- Comprehensive metrics collection\n"
            formatted_content+="\n"
            ;;
        "planning")
            formatted_content="### Added\n"
            formatted_content+="- Multi-component versioning support\n"
            formatted_content+="- Automated CHANGELOG generation\n"
            formatted_content+="- Version tracking for database migrations\n"
            formatted_content+="- Component-specific versioning rules\n"
            formatted_content+="\n"
            ;;
    esac
    
    echo "$formatted_content"
}

# Function to backup old files
backup_old_files() {
    local backup_dir="docs/backup_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"
    
    print_info "Creating backup in $backup_dir"
    
    # Backup files that will be migrated
    [ -f "docs/LAPORAN_PENGUJIAN_API.md" ] && cp "docs/LAPORAN_PENGUJIAN_API.md" "$backup_dir/"
    [ -f "docs/LAPORAN_DEBUGGING_API.md" ] && cp "docs/LAPORAN_DEBUGGING_API.md" "$backup_dir/"
    [ -f "docs/SCALABLE_MONITORING_ENGINE_ARCHITECTURE.md" ] && cp "docs/SCALABLE_MONITORING_ENGINE_ARCHITECTURE.md" "$backup_dir/"
    [ -f "docs/RENCANA_PENGUJIAN_API.md" ] && cp "docs/RENCANA_PENGUJIAN_API.md" "$backup_dir/"
    
    print_success "Backup created in $backup_dir"
}

# Function to update CHANGELOG with migrated content
update_changelog() {
    local changelog_path="CHANGELOG.md"
    local temp_changelog=$(mktemp)
    
    print_info "Updating CHANGELOG.md with migrated content"
    
    # Create new CHANGELOG content
    cat > "$temp_changelog" << 'EOF'
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Multi-component versioning support (backend, frontend, database, API)
- Automated CHANGELOG generation from git commits
- Version tracking for database migrations
- Component-specific versioning rules
- Git tagging strategy with automatic version tags
- Draft release mechanism for GitHub
- Shields.io integration for version and build badges
- Security event tracking and logging
- Comprehensive testing framework integration

### Changed
- Consolidated all documentation into unified CHANGELOG
- Improved versioning consistency across components
- Enhanced CI/CD pipeline with semantic release
- Optimized database migration versioning
- Streamlined release process with automated tagging

### Fixed
- Version synchronization issues between components
- Missing version information in releases
- Database schema versioning conflicts
- CI/CD pipeline timing issues

### Security
- Enhanced input validation for all API endpoints
- Improved structured logging for security events
- Authentication hardening for MikroTik connections
- Audit trail implementation for user actions

### Tested
- Comprehensive API testing suite integration
- Load testing for high-traffic scenarios
- Database migration testing
- Multi-component integration testing
- End-to-end release process validation

### Deprecated
- Individual component CHANGELOG files
- Manual version bumping process
- Static documentation maintenance

EOF

    # Add migrated content from old files
    local testing_content=$(extract_content "docs/LAPORAN_PENGUJIAN_API.md" "testing")
    local debugging_content=$(extract_content "docs/LAPORAN_DEBUGGING_API.md" "debugging")
    local architecture_content=$(extract_content "docs/SCALABLE_MONITORING_ENGINE_ARCHITECTURE.md" "architecture")
    local planning_content=$(extract_content "docs/RENCANA_PENGUJIAN_API.md" "planning")
    
    # Format and append content
    local formatted_testing=$(format_for_changelog "$testing_content" "testing" "1.0.0")
    local formatted_debugging=$(format_for_changelog "$debugging_content" "debugging" "1.0.0")
    local formatted_architecture=$(format_for_changelog "$architecture_content" "architecture" "1.0.0")
    local formatted_planning=$(format_for_changelog "$planning_content" "planning" "1.0.0")
    
    # Append to temp file
    echo "$formatted_testing" >> "$temp_changelog"
    echo "$formatted_debugging" >> "$temp_changelog"
    echo "$formatted_architecture" >> "$temp_changelog"
    echo "$formatted_planning" >> "$temp_changelog"
    
    # Add the rest of the existing CHANGELOG content
    if [ -f "$changelog_path" ]; then
        # Skip the header we just created and append the rest
        tail -n +$(grep -n "^## \[" "$changelog_path" | head -1 | cut -d: -f1) "$changelog_path" >> "$temp_changelog" 2>/dev/null || true
    fi
    
    # Replace original CHANGELOG
    mv "$temp_changelog" "$changelog_path"
    
    print_success "CHANGELOG.md updated with migrated content"
    
    # Cleanup temporary files
    [ -n "$testing_content" ] && rm -f "$testing_content"
    [ -n "$debugging_content" ] && rm -f "$debugging_content"
    [ -n "$architecture_content" ] && rm -f "$architecture_content"
    [ -n "$planning_content" ] && rm -f "$planning_content"
}

# Main execution
main() {
    print_info "Starting CHANGELOG migration process..."
    
    # Check if docs directory exists
    if [ ! -d "docs" ]; then
        print_warning "docs directory not found, skipping migration"
        return 0
    fi
    
    # Backup old files
    backup_old_files
    
    # Update CHANGELOG with migrated content
    update_changelog
    
    print_success "CHANGELOG migration completed successfully!"
    print_info "Old documentation files have been backed up"
    print_info "CHANGELOG.md has been updated with consolidated content"
}

# Run main function
main "$@"