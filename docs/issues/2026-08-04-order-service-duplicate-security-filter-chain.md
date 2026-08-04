# Order service CrashLoopBackOff from duplicate security filter chains

## Symptom

The Hostinger `order-service` deployment has zero ready replicas and its pod
restarts continuously. Spring Boot fails during startup with
`UnreachableFilterChainException`.

## Root cause

`SecurityConfig` and `OAuth2SecurityConfig` both publish a catch-all
`SecurityFilterChain`. When `oauth2.enabled=true`, Spring Security rejects the
second chain because both match any request.

## Fix

Load the mock/basic `SecurityConfig` only when OAuth2 is disabled (or absent).
The OAuth2 configuration remains the sole filter chain in production.

## Verification

Rely on the GitHub Actions Java CI test and package gates, then deploy the
resulting image and confirm the order-service pod reaches Ready without the
filter-chain exception. Local Maven execution is intentionally avoided because
this repository's agent guidance documents a Java-version mismatch.
