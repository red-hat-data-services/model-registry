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
# Fix #1 — Cargo git source redirect missing from Hermeto-generated config
#
# Hermeto dynamically generates .cargo/config.toml to redirect crates.io
# sources to the local vendor directory. However, it does not apply the same
# redirect for git-sourced dependencies (only registry sources are handled).
# This causes cargo to attempt a live network fetch during a hermetic build.
#
# Workaround: Overwrite the generated config with one that also redirects the
# known git source (pyca/cryptography) to the local vendor directory.
#
# Remove when: Hermeto supports vendoring and redirecting git-sourced Cargo deps.
# -----------------------------------------------------------------------------
cat <<EOF > /cachi2/output/.cargo/config.toml

[source.crates-io]
replace-with = "local"

[source."git+https://github.com/pyca/cryptography.git?tag=45.0.4"]
git = "https://github.com/pyca/cryptography.git"
tag = "45.0.4"
replace-with = "local"

[source.local]
directory = "/cachi2/output/deps/cargo"
EOF

# -----------------------------------------------------------------------------
# Fix #2 — sigstore_models cannot be built via uv-build in a hermetic environment
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

# -----------------------------------------------------------------------------
# Fix #3 — hf-xet requires Rust ≥ 1.89.0, but the UBI9 base image ships 1.88.0
#
# hf-xet (and its transitive dependencies) enforce a minimum Rust toolchain
# version of 1.89.0. The UBI9 base image currently provides only 1.88.0,
# causing the build to abort with a toolchain version error.
#
# Workaround: Pass --ignore-rust-version to maturin via MATURIN_PEP517_ARGS so
# the version check is skipped. The build proceeds successfully at 1.88.0.
#
# Remove when: the UBI9 base image is updated to Rust ≥ 1.89.0.
# -----------------------------------------------------------------------------
MATURIN_PEP517_ARGS="--ignore-rust-version" pip install hf-xet