#!/bin/bash
set -e

# =============================================================================
# Hermetic Build Workarounds
#
# This script applies temporary fixes required for hermetic (Hermeto/Konflux)
# builds to succeed. Each fix is documented with the root cause and the
# upstream condition that would make it safe to remove.
# =============================================================================

# -----------------------------------------------------------------------------
# Fix #1 — sigstore_models cannot be built via uv-build in a hermetic environment
#
# sigstore_models declares uv-build as its build backend. uv-build depends on
# maturin, which generates a Cargo invalid lockfile causing the hermetic build to fail.
#
# Workaround: Strip the [build-system] section from sigstore_models' pyproject.toml
# before installation. The package is pure Python so it installs cleanly with
# plain pip, bypassing maturin entirely.
#
# Remove when: maturin generates valid lockfiles, or uv-build drops
#              its maturin dependency.
# -----------------------------------------------------------------------------
tar -xzf /cachi2/output/deps/pip/sigstore_models-0.0.6.tar.gz -C /tmp
cd /tmp/sigstore_models-0.0.6
sed -i '/^\[build-system\]$/,/^build-backend = "uv_build"$/d' pyproject.toml
python -m pip install .
