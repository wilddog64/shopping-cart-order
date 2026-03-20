# Issue: Multi-arch workflow pin update

**Date:** 2026-03-17
**Status:** Closed

## Summary
`.github/workflows/ci.yml` referenced infra workflow SHA `8363caf`, so CI produced amd64-only images. Infra SHA `999f8d7` adds `platforms: linux/amd64,linux/arm64`.

## Fix
- Updated the reusable workflow pin to `999f8d70277b92d928412ff694852b05044dbb75`.
- Ensures `build-push-deploy` publishes arm64 images for Ubuntu k3s nodes.

## Follow Up
- Monitor CI and ArgoCD sync to confirm arm64 pods start successfully.
