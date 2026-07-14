#!/bin/bash
# Local CI pre-push validation script (Track G).
#
# This script aggregates the same validation steps that run in GitHub Actions CI
# so that maintainers can gate pushes locally. It is a best-effort gate: it
# runs on the current host only and cannot fully simulate macOS or Windows
# Wails builds. macOS/Windows coverage is approximated through Go
# cross-compilation and platform-specific test compilation only.
#
# Usage:
#   ./scripts/build/local-ci-check.sh
#
# Exit codes:
#   0  - all validation steps passed
#   1  - one or more validation steps failed (details written to
#        .medmemo/review/local-ci-failure.log)
#
# Steps mirror the Track G table in the v1.1.10 cross-platform packaging plan.
set -euo pipefail

# Match the Go toolchain pinned in the Makefile so every go invocation uses
# the same version required by go.mod (e.g. cross-compilation steps).
export GOTOOLCHAIN=go1.26.4

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FAILURE_LOG="$REPO_ROOT/.medmemo/review/local-ci-failure.log"

TOTAL_STEPS=0
PASSED_STEPS=0
FAILED_STEPS=0
FAILED_NAMES=()

# Ensure the failure log directory exists.
mkdir -p "$(dirname "$FAILURE_LOG")"

# Clear previous failure log.
: > "$FAILURE_LOG"

# Run a named validation step. The remaining arguments form the command.
run_step() {
    local name="$1"
    shift
    local output
    local exit_code=0

    TOTAL_STEPS=$((TOTAL_STEPS + 1))
    printf '[%2d/%2d] %-55s ' "$TOTAL_STEPS" "${STEP_COUNT:-?}" "$name"

    output="$(mktemp)"
    (
        cd "$REPO_ROOT"
        "$@"
    ) > "$output" 2>&1 || exit_code=$?

    if [ "$exit_code" -eq 0 ]; then
        echo "✓ PASS"
        PASSED_STEPS=$((PASSED_STEPS + 1))
    else
        echo "✗ FAIL"
        FAILED_STEPS=$((FAILED_STEPS + 1))
        FAILED_NAMES+=("$name")
        {
            echo "===== $name ====="
            cat "$output"
            echo ""
        } >> "$FAILURE_LOG"
    fi

    rm -f "$output"
}

# Run a best-effort step. A failure is recorded as a warning and does not
# fail the overall validation. Used for environment-dependent checks such as
# cross-compilation of CGO packages from a Linux host.
run_warn_step() {
    local name="$1"
    shift
    local output
    local exit_code=0

    TOTAL_STEPS=$((TOTAL_STEPS + 1))
    printf '[%2d/%2d] %-55s ' "$TOTAL_STEPS" "${STEP_COUNT:-?}" "$name"

    output="$(mktemp)"
    (
        cd "$REPO_ROOT"
        "$@"
    ) > "$output" 2>&1 || exit_code=$?

    if [ "$exit_code" -eq 0 ]; then
        echo "✓ PASS"
        PASSED_STEPS=$((PASSED_STEPS + 1))
    else
        echo "⚠ WARN (best-effort step failed; see $FAILURE_LOG)"
        {
            echo "===== $name (best-effort warning) ====="
            cat "$output"
            echo ""
        } >> "$FAILURE_LOG"
    fi

    rm -f "$output"
}

# Print summary and persist the final failure log.
finish() {
    echo ""
    echo "========================================"
    echo "Local CI validation complete"
    echo "Total:  $TOTAL_STEPS"
    echo "Passed: $PASSED_STEPS"
    echo "Failed: $FAILED_STEPS"

    if [ "$FAILED_STEPS" -gt 0 ]; then
        echo ""
        echo "Failing steps:"
        for name in "${FAILED_NAMES[@]}"; do
            echo "  - $name"
        done
        echo ""
        echo "Full failure log: $FAILURE_LOG"
        exit 1
    fi

    echo "All local CI validation steps passed."
    exit 0
}

# Pre-compute total step count so the progress indicator is accurate.
# 11 core steps + 3 cross-compile checks + 2 platform test compiles + 4 docs checks = 20.
STEP_COUNT=20

# ----------------------------- Core validation -----------------------------
run_step "Go lint"                           make lint
run_step "Frontend lint"                     bash -c 'cd web && npm run lint'
run_step "Frontend typecheck"                bash -c 'cd web && npx tsc --noEmit'
run_step "Provider templates validation"     node scripts/validate-provider-templates.js
run_step "Frontend build"                    bash -c 'cd web && npm run build'
run_step "Go unit tests"                     make test
run_step "Go integration tests"              make test-integration
run_step "Go E2E tests"                      make test-e2e
run_step "Current platform build"            make build
run_step "Wire DI sync"                      bash -c 'make wire && git diff --exit-code wire_gen.go'

# Release asset whitelist check runs only when artifacts have been prepared.
if [ -d "$REPO_ROOT/release-assets" ]; then
    run_step "Release asset whitelist"       ./scripts/build/verify-release-assets.sh release-assets
else
    TOTAL_STEPS=$((TOTAL_STEPS + 1))
    printf '[%2d/%2d] %-55s ' "$TOTAL_STEPS" "$STEP_COUNT" "Release asset whitelist"
    echo "⊘ SKIP (release-assets directory not present)"
fi

# ------------------------- Cross-platform checks ---------------------------
# Cross-compiling the entire project requires CGO cross-compilers for
# sqlcipher, which are usually unavailable on a Linux host. These steps are
# best-effort; platform-specific test compilation below catches OS-gated
# build errors in the updater packages.
run_warn_step "Cross-compile darwin/amd64"        env GOOS=darwin  GOARCH=amd64 go build ./...
run_warn_step "Cross-compile darwin/arm64"        env GOOS=darwin  GOARCH=arm64 go build ./...
run_warn_step "Cross-compile windows/amd64"       env GOOS=windows GOARCH=amd64 go build ./...

# Platform-specific test compilation to catch OS-gated build errors.
run_step "Test compile updater (darwin)"     env GOOS=darwin  GOARCH=arm64 go test -c ./internal/infrastructure/updater
run_step "Test compile updater (windows)"    env GOOS=windows GOARCH=amd64 go test -c ./internal/infrastructure/updater

# Clean up test binaries produced by the platform-specific compile steps.
rm -f "$REPO_ROOT/updater.test" "$REPO_ROOT/updater.test.exe"

# ----------------------------- Documentation checks -----------------------------
# Link check is best-effort locally because lychee is not always installed.
run_warn_step "Docs Markdown link check"        ./scripts/check-doc-links.sh
run_step "Docs Chinese mirror check"            node scripts/check-doc-mirrors.js
run_step "Docs terminology check"               node scripts/check-terminology.js
run_step "Docs version consistency check"       node scripts/check-version-consistency.js

finish
