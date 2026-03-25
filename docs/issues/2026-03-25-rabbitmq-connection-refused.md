# Issue: RabbitMQ Connection Refused — order-service CrashLoopBackOff

**Date:** 2026-03-25
**GitHub Issue:** wilddog64/shopping-cart-order#16
**Fixed in:** wilddog64/shopping-cart-infra PR (fix/rabbitmq-guest-remote-access)
**Status:** Fixed

## Symptoms

- `order-service` pod in `CrashLoopBackOff` in `shopping-cart-apps` namespace
- ArgoCD status: **Degraded**
- Spring Boot startup error: `RabbitMQ Connection refused` at `rabbitmq.shopping-cart-data.svc.cluster.local:5672`
- PostgreSQL connectivity confirmed working

## Root Causes

Two independent issues combined to cause the failure:

### 1. `guest` user restricted to localhost (primary cause)

RabbitMQ restricts the `guest` user to loopback connections (`127.0.0.1`) by default.
Cross-namespace pod connections from `shopping-cart-apps` are rejected at the AMQP layer,
which Spring Boot AMQP reports as "Connection refused".

**Fix:** Added `loopback_users.guest = none` to `data-layer/rabbitmq/configmap.yaml`.
This is dev-only; Stage 2 Vault integration replaces `guest` with dynamic credentials.

### 2. No ArgoCD Application for data-layer (contributing cause)

The `data-layer/` directory (PostgreSQL, RabbitMQ, Redis) had no ArgoCD Application manifest.
After a cluster reset or fresh deploy, RabbitMQ was not deployed automatically — it required
manual `kubectl apply -f data-layer/rabbitmq/`.

**Fix:** Added `argocd/applications/data-layer.yaml` to manage the data-layer via GitOps.

### 3. Resource pressure on t3.medium (contributing cause)

RabbitMQ requested 500m CPU / 1Gi RAM. On a t3.medium (2 vCPU / 4GB) running PostgreSQL,
Redis, Istio, and application pods, this left insufficient headroom and could prevent
RabbitMQ from scheduling.

**Fix:** Reduced requests to 200m CPU / 512Mi RAM in `statefulset.yaml`. Limits unchanged
at 1000m / 1Gi to allow bursting.

## Verification

After applying the infra fix:

```bash
# Confirm RabbitMQ pod is Running
kubectl get pods -n shopping-cart-data -l app=rabbitmq

# Confirm order-service connects
kubectl logs -n shopping-cart-apps deployment/order-service | grep -i rabbitmq

# Test AMQP from within cluster
kubectl run -it --rm amqp-test --image=busybox --restart=Never -n shopping-cart-apps -- \
  nc -zv rabbitmq.shopping-cart-data.svc.cluster.local 5672
```

## Process Note

Any data-layer component (RabbitMQ, PostgreSQL, Redis) must have a corresponding ArgoCD
Application if it is to be managed by GitOps. Without one, it silently disappears after
a cluster reset and causes dependent services to crash-loop.
