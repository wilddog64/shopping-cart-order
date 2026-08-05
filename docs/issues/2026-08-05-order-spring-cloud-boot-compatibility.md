# Order service fails to start after Spring Boot 3.5 upgrade

**Date:** 2026-08-05
**Repository:** `shopping-cart-order`
**Status:** Fixed in PR #60; awaiting merge and live verification

## What Was Tested

The live `order-service` pod was checked after the image built from main was
deployed to the ubuntu-hostinger cluster.

## Actual Output

```text
Spring Boot [3.5.16] is not compatible with this Spring Cloud release train

Change Spring Boot version to one of the following versions [3.2.x].
```

The pod exited with status 1 and entered CrashLoopBackOff. The image itself
was present and successfully pulled:

```text
ghcr.io/wilddog64/shopping-cart-order:sha-02840413b66773431519cc193d899b18fa9c3032
sha256:f8f8827b52f7ca76f36c4339f18f3fcca8b647be0da88aa0f4525560c6db804f
```

## Root Cause

Dependabot commit `3fb5207` changed the Spring Boot parent from `3.2.0` to
`3.5.16`, while the Spring Cloud release train brought in by the service only
supports Spring Boot 3.2.x.

## Resolution

Restore the Spring Boot parent to `3.2.0`. Do not disable Spring Cloud's
compatibility verifier: it correctly prevents an unsupported runtime
combination. Validate the change through the repository CI and redeploy the
new SHA image after merge.
