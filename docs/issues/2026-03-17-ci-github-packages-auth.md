# CI Failure: Maven Cannot Authenticate to GitHub Packages

**Date:** 2026-03-17
**Severity:** CI blocking — Docker build fails, image never pushed to ghcr.io
**Status:** Open — assigned to Codex

---

## Symptoms

`Build, Scan & Push` job fails in CI at `RUN mvn dependency:go-offline -B` with exit code 1.
Maven attempts to resolve dependencies from `maven.pkg.github.com/wilddog64/rabbitmq-client-java`
but receives a 401/403 — no credentials are available inside the Docker build context.

```
[INFO] Downloading from github-rabbitmq-client: https://maven.pkg.github.com/wilddog64/rabbitmq-client-java/...
[ERROR] Failed to execute goal ...
ERROR: process "/bin/sh -c mvn dependency:go-offline -B" did not complete successfully: exit code: 1
```

## Root Cause

`pom.xml` declares a Maven repository `github-rabbitmq-client` pointing to
`https://maven.pkg.github.com/wilddog64/rabbitmq-client-java`. GitHub Packages requires
authentication for all reads. The `Dockerfile` has no mechanism to pass credentials into
the Maven build.

The infra reusable workflow (`build-push-deploy.yml@8363caf`) already forwards
`PACKAGES_TOKEN` as a Docker build secret (`GH_TOKEN=${{ secrets.PACKAGES_TOKEN }}`), but
the `Dockerfile` does not mount or use this secret.

## Fix

Use Docker BuildKit secret mounts to pass `GH_TOKEN` into the Maven build without
embedding credentials in an image layer.

**In `Dockerfile` — Stage 1 (builder):**

1. Add a Maven `settings.xml` that reads credentials from the build secret:
   ```dockerfile
   RUN --mount=type=secret,id=GH_TOKEN \
       mkdir -p /root/.m2 && \
       printf '<settings>\n  <servers>\n    <server>\n      <id>github-rabbitmq-client</id>\n      <username>x-token-auth</username>\n      <password>%s</password>\n    </server>\n  </servers>\n</settings>\n' \
         "$(cat /run/secrets/GH_TOKEN)" > /root/.m2/settings.xml
   ```

2. Keep the existing `RUN mvn dependency:go-offline -B` and `RUN mvn package -DskipTests -B`
   lines unchanged — Maven will pick up `/root/.m2/settings.xml` automatically.

   Alternatively, combine into the same `--mount` layer:
   ```dockerfile
   RUN --mount=type=secret,id=GH_TOKEN \
       mkdir -p /root/.m2 && \
       printf '<settings>...</settings>' "$(cat /run/secrets/GH_TOKEN)" > /root/.m2/settings.xml && \
       mvn dependency:go-offline -B
   ```

3. Also copy `checkstyle.xml` so `mvn package` can find it:
   ```dockerfile
   # Change
   COPY pom.xml .
   # To
   COPY pom.xml checkstyle.xml .
   ```

## Status

- Fixed on `main` (commit `cb663a2`). CI run 23175038080 completed successfully; image push verified via workflow logs.

## Constraints

- Only touch `Dockerfile` — do NOT modify `pom.xml`, `ci.yaml`, or infra workflow
- The `GH_TOKEN` secret is already forwarded by the infra workflow — no workflow changes needed
- Do NOT use `--no-verify` on commits
- Commit on `main` branch

## Verification

```bash
gh run list -R wilddog64/shopping-cart-order --limit 1 --json status,conclusion,headSha
gh api repos/wilddog64/shopping-cart-order/packages/container/shopping-cart-order/versions \
  --jq '.[0] | {version: .name, updated: .updated_at}'
```

## Definition of Done

- [ ] CI run green (all jobs pass)
- [ ] Image `ghcr.io/wilddog64/shopping-cart-order:latest` exists in ghcr.io
- [ ] Commit SHA reported back
