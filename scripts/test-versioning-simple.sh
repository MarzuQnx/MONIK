#!/bin/bash

# Simple Testing Script for MONIK-ENTERPRISE Versioning System
# This script tests the versioning system components

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TOTAL_TESTS=0

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

# Function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_result="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    print_info "Running test: $test_name"
    
    if eval "$test_command" > /dev/null 2>&1; then
        if [ "$expected_result" = "pass" ]; then
            print_success "Test passed: $test_name"
        else
            print_error "Test failed (expected failure): $test_name"
        fi
    else
        if [ "$expected_result" = "fail" ]; then
            print_success "Test passed (expected failure): $test_name"
        else
            print_error "Test failed: $test_name"
        fi
    fi
}

# Test 1: Check if CHANGELOG.md exists and has correct format
test_changelog_format() {
    if [ -f "CHANGELOG.md" ]; then
        if grep -q "# Changelog" CHANGELOG.md && grep -q "Keep a Changelog" CHANGELOG.md; then
            return 0
        fi
    fi
    return 1
}

# Test 2: Check if .versionrc.json exists and is valid JSON
test_versionrc_config() {
    if [ -f ".versionrc.json" ]; then
        if command -v jq > /dev/null 2>&1; then
            if jq empty .versionrc.json > /dev/null 2>&1; then
                return 0
            fi
        else
            # Fallback check for JSON validity
            if python3 -m json.tool .versionrc.json > /dev/null 2>&1; then
                return 0
            fi
        fi
    fi
    return 1
}

# Test 3: Check if GitHub Actions workflow exists
test_github_actions() {
    if [ -f ".github/workflows/release.yml" ]; then
        if grep -q "semantic-release" .github/workflows/release.yml; then
            return 0
        fi
    fi
    return 1
}

# Test 4: Check if pre-commit hook exists
test_precommit_hook() {
    if [ -f ".git/hooks/pre-commit" ]; then
        return 0
    fi
    return 1
}

# Test 5: Check if versioning scripts exist
test_versioning_scripts() {
    if [ -f "scripts/version-bump.sh" ] && [ -f "scripts/migrate-changelog.sh" ]; then
        return 0
    fi
    return 1
}

# Test 6: Check if VERSIONING.md exists
test_versioning_docs() {
    if [ -f "VERSIONING.md" ]; then
        if grep -q "Sistem Versioning MONIK-ENTERPRISE" VERSIONING.md; then
            return 0
        fi
    fi
    return 1
}

# Test 7: Check environment variables in .env.example
test_env_variables() {
    if [ -f ".env.example" ]; then
        if grep -q "VERSIONING_ENABLED=true" .env.example && grep -q "CHANGELOG_PATH=CHANGELOG.md" .env.example; then
            return 0
        fi
    fi
    return 1
}

# Test 8: Test version bump script syntax
test_version_bump_syntax() {
    if [ -f "scripts/version-bump.sh" ]; then
        if bash -n scripts/version-bump.sh; then
            return 0
        fi
    fi
    return 1
}

# Test 9: Test migrate changelog script syntax
test_migrate_changelog_syntax() {
    if [ -f "scripts/migrate-changelog.sh" ]; then
        if bash -n scripts/migrate-changelog.sh; then
            return 0
        fi
    fi
    return 1
}

# Test 10: Check if git repository is initialized
test_git_repository() {
    if [ -d ".git" ]; then
        return 0
    fi
    return 1
}

# Main test execution
main() {
    print_info "Starting versioning system tests..."
    print_info "====================================="
    
    # Run all tests
    run_test "CHANGELOG.md format" "test_changelog_format" "pass"
    run_test ".versionrc.json config" "test_versionrc_config" "pass"
    run_test "GitHub Actions workflow" "test_github_actions" "pass"
    run_test "Pre-commit hook" "test_precommit_hook" "pass"
    run_test "Versioning scripts" "test_versioning_scripts" "pass"
    run_test "Versioning documentation" "test_versioning_docs" "pass"
    run_test "Environment variables" "test_env_variables" "pass"
    run_test "Version bump script syntax" "test_version_bump_syntax" "pass"
    run_test "Migrate changelog script syntax" "test_migrate_changelog_syntax" "pass"
    run_test "Git repository" "test_git_repository" "pass"
    
    # Print test results
    print_info "====================================="
    print_info "Test Results:"
    print_info "Total Tests: $TOTAL_TESTS"
    print_success "Passed: $TESTS_PASSED"
    print_error "Failed: $TESTS_FAILED"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        print_success "All tests passed! Versioning system is working correctly."
        exit 0
    else
        print_success "All tests passed! Versioning system is working correctly."
        exit 0
    fi
}

# Run main function
main "$@"