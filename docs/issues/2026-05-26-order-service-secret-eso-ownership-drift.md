# Issue: `order-service-secrets` is still placeholder-managed in `shopping-cart-order` while the live cluster secret is ESO-owned

**Date:** 2026-05-26
**Repository:** `shopping-cart-order`
**Status:** Open

## Symptoms

After rebuilding the remote `ubuntu-k3s` cluster, Argo CD reported:

```text
shopping-cart-order             OutOfSync     Progressing
```

Argo CD showed the drift specifically on `Secret/order-service-secrets` in `shopping-cart-apps`.

The live secret is already owned by External Secrets Operator (ESO):

```text
ownerReferences:
- apiVersion: external-secrets.io/v1
  kind: ExternalSecret
  name: order-service-secrets
```

The live secret also carries ESO-managed markers:

```text
labels:
  reconcile.external-secrets.io/managed: "true"
annotations:
  reconcile.external-secrets.io/data-hash: 830dbe5fe5d0c1081d3d6aab5455024a2369037d036251b67535879c
```

But the app source in `shopping-cart-order` still includes a placeholder secret manifest:

```yaml
# k8s/base/kustomization.yaml
resources:
- serviceaccount.yaml
- configmap.yaml
- secret.yaml
- deployment.yaml
- service.yaml
```

and the placeholder data is still committed in `k8s/base/secret.yaml`.

## Actual Output

From the live cluster:

```text
$ kubectl --context k3d-k3d-cluster -n cicd get application shopping-cart-order -o yaml
...
status:
  sync:
    status: OutOfSync
  resources:
  - kind: Secret
    name: order-service-secrets
    namespace: shopping-cart-apps
    status: OutOfSync
```

```text
$ kubectl --context ubuntu-k3s -n shopping-cart-apps get secret order-service-secrets -o yaml
...
ownerReferences:
- apiVersion: external-secrets.io/v1
  kind: ExternalSecret
  name: order-service-secrets
...
```

```text
$ kubectl --context ubuntu-k3s -n shopping-cart-apps get externalsecret order-service-secrets -o yaml
...
status:
  conditions:
  - type: Ready
    status: "True"
    reason: SecretSynced
    message: secret synced
```

## Root Cause

This is a GitOps ownership mismatch.

- `shopping-cart-order` still renders a placeholder `Secret` named `order-service-secrets` from `k8s/base/secret.yaml`.
- The live cluster already expects ESO to own and refresh that secret.
- Argo CD compares the rendered placeholder against the ESO-mutated live secret data and keeps the application `OutOfSync`.

This is the same class of problem that was fixed for the payment service: the repo must stop claiming ownership of a secret that is now sourced from Vault through ESO.

## Recommended Follow-up

- Remove `k8s/base/secret.yaml` from `k8s/base/kustomization.yaml`.
- Delete the placeholder `k8s/base/secret.yaml` manifest from `shopping-cart-order`.
- Keep the source of truth for `order-service-secrets` in `shopping-cart-infra/data-layer/secrets/postgres-orders-apps-externalsecret.yaml`.
- After the repo fix merges, refresh Argo CD and confirm `shopping-cart-order` returns to `Synced`.

## Verification

After the repo fix lands, verify:

```bash
kubectl --context k3d-k3s-cluster -n cicd get application shopping-cart-order -o wide
kubectl --context ubuntu-k3s -n shopping-cart-apps get secret order-service-secrets -o yaml
kubectl --context ubuntu-k3s -n shopping-cart-apps get externalsecret order-service-secrets -o yaml
```

Expected outcome:

- `shopping-cart-order` is `Synced`
- `order-service-secrets` remains ESO-owned
- no manual cluster patching is required after a remote rebuild
