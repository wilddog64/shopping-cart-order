# Issue: Rate Limiter is Per-Pod (Not Cluster-Wide)

**Date:** 2026-03-18
**Status:** Open

## Problem

`RateLimitConfig` uses an in-memory Caffeine cache to store per-IP token buckets.
With 2+ replicas, each pod has independent counters — effective limit is
`requestsPerSecond * replicas` per IP. An attacker hitting multiple pods
round-robin bypasses the intended limit entirely.

Current limits:
- 20 req/s per IP per pod (50 burst)
- 100 req/min per IP per pod

With 2 replicas: effectively 40 req/s / 200 req/min per IP.

## Fix: Replace Caffeine cache with Redis-backed Bucket4j

Bucket4j supports Redis as a distributed state backend via `bucket4j-redisson` (Redisson integration).

### 1. Add dependencies

```xml
<!-- pom.xml -->
<dependency>
    <groupId>com.bucket4j</groupId>
    <artifactId>bucket4j-redisson</artifactId>
    <version>8.10.1</version>
</dependency>
<dependency>
    <groupId>org.redisson</groupId>
    <artifactId>redisson</artifactId>
    <version>3.27.2</version>
</dependency>
```

### 2. Replace in-memory cache with Redis proxy

In `RateLimitConfig.java`, replace:

```java
// REMOVE
@Bean
public Cache<String, Bucket> rateLimitBucketCache() {
    return Caffeine.newBuilder()
        .expireAfterAccess(1, TimeUnit.HOURS)
        .maximumSize(100_000)
        .build();
}
```

With:

```java
// ADD
@Bean
public RedissonClient redissonClient(
        @Value("${spring.data.redis.host}") String host,
        @Value("${spring.data.redis.port}") int port) {
    Config config = new Config();
    config.useSingleServer().setAddress("redis://" + host + ":" + port);
    return Redisson.create(config);
}

@Bean
public ProxyManager<String> bucketProxyManager(RedissonClient redissonClient) {
    return Bucket4jRedisson.casBasedBuilder(redissonClient.getMap("rate-limit-buckets"))
        .build();
}
```

### 3. Update RateLimitFilter to use ProxyManager

```java
// Replace Cache<String, Bucket> injection with ProxyManager<String>
private final ProxyManager<String> bucketProxyManager;
private final RateLimitConfig rateLimitConfig;

// Replace bucket lookup — preserve all 3 existing rate bands:
BucketConfiguration config = BucketConfiguration.builder()
    .addLimit(rateLimitConfig.createBurstBandwidth())
    .addLimit(rateLimitConfig.createPerSecondBandwidth())
    .addLimit(rateLimitConfig.createPerMinuteBandwidth())
    .build();
Bucket bucket = bucketProxyManager.builder().build(clientIp, config);
```

### 4. Environment variables

Add `REDIS_HOST` and `REDIS_PORT` to `k8s/base/configmap.yaml` and `application.yml` — this service does not currently have Redis config. The Redis instance is in the `shopping-cart-data` namespace.

## Definition of Done

- [ ] `bucket4j-redisson` + `redisson` added to `pom.xml`
- [ ] `RedissonClient` bean configured from `REDIS_HOST` / `REDIS_PORT`
- [ ] `ProxyManager<String>` replaces `Cache<String, Bucket>` in `RateLimitFilter`
- [ ] Rate limit is enforced cluster-wide (test: 2 pods, same IP, combined count hits limit)
- [ ] Unit tests updated — mock `ProxyManager` instead of Caffeine cache
- [ ] No changes to k8s manifests, Dockerfiles, or other services

## What NOT to Do

- Do NOT change the rate limit values (20 req/s, 100 req/min) — only the backend
- Do NOT add rate limiting to health/actuator endpoints (already excluded)
- Do NOT modify RabbitMQ or database code
