#!/usr/bin/env bash
# Release the Hivehook Terraform provider to GitHub (goreleaser builds + GPG-signs
# the artifacts; the Terraform Registry ingests them once registered).
# Tag the release first:  git tag -a v0.1.1-beta -m v0.1.1-beta && git push origin v0.1.1-beta
# Then run this from the repo root. Requires: goreleaser, gpg key, gh auth.
set -uo pipefail

read -rsp "GPG Passphrase: " GPG_PASSPHRASE; echo
export GPG_PASSPHRASE
export GPG_FINGERPRINT="37AA881A0C53A2E8FAB6F5B0E759C79FDD4A0454"
export GITHUB_TOKEN="$(gh auth token)"

goreleaser release --clean

unset GPG_PASSPHRASE GITHUB_TOKEN
