# Issue: `order-service` is degraded after remote rebuild because PostgreSQL authentication fails

**Date:** 2026-05-26
**Repository:** `shopping-cart-order`
**Status:** Open

## Symptoms

After the remote `ubuntu-k3s` cluster was rebuilt, Argo CD showed:

```text
shopping-cart-order             Synced     Degraded
```

The `order-service` pod in `shopping-cart-apps` was stuck in `CrashLoopBackOff` and never became healthy enough to pass its startup probe.

## Actual Output

From the live cluster:

```text
$ kubectl --context k3d-k3d-cluster -n cicd get application shopping-cart-order -o yaml
...
status:
  sync:
    status: Synced
  health:
    status: Degraded
```

```text
$ kubectl --context ubuntu-k3s -n shopping-cart-apps get pods -l app.kubernetes.io/name=order-service -o wide
NAME                             READY   STATUS             RESTARTS       AGE
order-service-5545485d98-rxf7g   0/1     CrashLoopBackOff   9 (4m5s ago)   26m
```

```text
$ kubectl --context ubuntu-k3s -n shopping-cart-apps describe pod order-service-5545485d98-rxf7g
...
Last State:     Terminated
  Reason:       Error
  Exit Code:    1
Events:
  Warning  Unhealthy  ...  Startup probe failed: dial tcp 10.42.0.14:8080: connect: connection refused
```

The previous container logs show the application failing while connecting to PostgreSQL:

```text
org.postgresql.util.PSQLException: FATAL: password authentication failed for user "postgres"
```

The application source still consumes `order-service-secrets` via `envFrom` in `k8s/base/deployment.yaml`, and the database host still points at `postgresql-orders.shopping-cart-data.svc.cluster.local` in `k8s/base/configmap.yaml`.

The live cluster data layer is healthy:

```text
$ kubectl --context ubuntu-k3s -n shopping-cart-data get pods -o wide
minio-0                 1/1     Running
postgresql-orders-0     1/1     Running
postgresql-payment-0    1/1     Running
postgresql-products-0   1/1     Running
rabbitmq-0              1/1     Running
redis-cart-0            1/1     Running
```

## Root Cause

This is a runtime credential mismatch, not an Argo CD drift issue.

- `shopping-cart-order` is already `Synced`.
- The `order-service` container is starting, but it fails during PostgreSQL initialization.
- PostgreSQL rejects the password for user `postgres`, so Spring Boot never reaches a healthy state and the startup probe keeps failing.

The most likely explanation is that the password material consumed by `order-service-secrets` no longer matches the live `postgresql-orders` credential expected by the database.

## Recommended Follow-up

- Verify the live PostgreSQL admin password and the `order-service-secrets.DB_PASSWORD` value are aligned.
- Confirm the Vault / ESO source of truth for the orders database credentials is being seeded consistently after sandbox rebuilds.
- After the source is corrected, refresh Argo CD and verify `shopping-cart-order` returns to `Healthy`.

## Verification

After the repo fix lands, verify:

```bash
kubectl --context k3d-k3d-cluster -n cicd get application shopping-cart-order -o wide
kubectl --context ubuntu-k3s -n shopping-cart-apps get pods -l app.kubernetes.io/name=order-service -o wide
kubectl --context ubuntu-k3s -n shopping-cart-apps logs deployment/order-service --tail=100
```

Expected outcome:

- `shopping-cart-order` is `Synced` and `Healthy`
- `order-service` reaches `Running`
- PostgreSQL authentication succeeds on startup
