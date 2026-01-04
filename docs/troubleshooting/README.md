# Order Service Troubleshooting Guide

## Common Issues

### Connection Issues

#### Database Connection Failed

**Symptoms:**
```
HikariPool-1 - Connection is not available, request timed out after 30000ms
```

**Causes:**
- PostgreSQL not running
- Incorrect connection URL
- Network connectivity issues
- Connection pool exhausted

**Solutions:**

1. Verify PostgreSQL is running:
   ```bash
   kubectl get pods -n shopping-cart -l app=postgresql
   ```

2. Check connection settings:
   ```bash
   kubectl get secret order-service-secrets -o jsonpath='{.data.SPRING_DATASOURCE_URL}' | base64 -d
   ```

3. Test connectivity:
   ```bash
   kubectl exec -it order-service-xxx -- nc -zv postgresql 5432
   ```

4. Check pool settings in `application.yml`:
   ```yaml
   spring:
     datasource:
       hikari:
         maximum-pool-size: 10
         connection-timeout: 30000
   ```

#### RabbitMQ Connection Failed

**Symptoms:**
```
Unable to connect to RabbitMQ: Connection refused
```

**Solutions:**

1. Verify RabbitMQ is running:
   ```bash
   kubectl get pods -n rabbitmq -l app=rabbitmq
   ```

2. Check credentials from Vault:
   ```bash
   vault read rabbitmq/creds/order-service
   ```

3. Test connectivity:
   ```bash
   kubectl exec -it order-service-xxx -- nc -zv rabbitmq 5672
   ```

### Authentication Issues

#### 401 Unauthorized

**Symptoms:**
```json
{
  "status": 401,
  "error": "Unauthorized",
  "message": "Full authentication is required"
}
```

**Causes:**
- Missing Authorization header
- Expired JWT token
- Invalid token signature

**Solutions:**

1. Verify OAuth2 is configured:
   ```bash
   kubectl get configmap order-service-config -o yaml | grep OAUTH2
   ```

2. Check token expiration:
   ```bash
   # Decode JWT (without verification)
   echo $TOKEN | cut -d. -f2 | base64 -d | jq .exp
   ```

3. Verify Keycloak issuer is accessible:
   ```bash
   curl -s $OAUTH2_ISSUER_URI/.well-known/openid-configuration
   ```

#### 403 Forbidden

**Symptoms:**
```json
{
  "status": 403,
  "error": "Forbidden",
  "message": "Access Denied"
}
```

**Causes:**
- User lacks required role
- Endpoint requires specific permission

**Solutions:**

1. Check user roles in JWT:
   ```bash
   echo $TOKEN | cut -d. -f2 | base64 -d | jq .realm_access.roles
   ```

2. Verify required roles for endpoint in `SecurityConfig.java`

3. Add role in Keycloak admin console

### Rate Limiting

#### 429 Too Many Requests

**Symptoms:**
```json
{
  "status": 429,
  "error": "Too Many Requests",
  "message": "Rate limit exceeded"
}
```

**Solutions:**

1. Check rate limit headers:
   ```bash
   curl -i http://localhost:8080/api/orders?customerId=test
   # Look for: X-Rate-Limit-Remaining, X-Rate-Limit-Retry-After
   ```

2. Adjust rate limits in configuration:
   ```yaml
   rate-limit:
     requests-per-minute: 100
     requests-per-second: 20
     burst-capacity: 50
   ```

3. For testing, exclude paths from rate limiting

### Data Issues

#### Order Not Found

**Symptoms:**
```json
{
  "status": 404,
  "error": "Not Found",
  "message": "Order not found: xxx"
}
```

**Solutions:**

1. Verify order exists in database:
   ```sql
   SELECT * FROM orders WHERE id = 'xxx';
   ```

2. Check if order belongs to correct customer (if multi-tenant)

#### Invalid Status Transition

**Symptoms:**
```json
{
  "status": 409,
  "error": "Conflict",
  "message": "Cannot transition from DELIVERED to CANCELLED"
}
```

**Solutions:**

1. Check current order status
2. Review valid status transitions in [API docs](../api/README.md#valid-status-transitions)

### Performance Issues

#### Slow Response Times

**Symptoms:**
- API latency > 500ms
- Timeouts on requests

**Diagnosis:**

1. Check database query performance:
   ```sql
   EXPLAIN ANALYZE SELECT * FROM orders WHERE customer_id = 'xxx';
   ```

2. Check connection pool metrics:
   ```bash
   curl localhost:8080/actuator/metrics/hikaricp.connections.active
   ```

3. Review application logs for slow queries:
   ```bash
   kubectl logs -f order-service-xxx | grep -i slow
   ```

**Solutions:**

1. Add missing indexes:
   ```sql
   CREATE INDEX idx_orders_customer_id ON orders(customer_id);
   CREATE INDEX idx_orders_status ON orders(status);
   ```

2. Increase connection pool size
3. Enable query caching

#### High Memory Usage

**Symptoms:**
- OOMKilled pods
- Frequent garbage collection

**Solutions:**

1. Check memory metrics:
   ```bash
   kubectl top pod order-service-xxx
   ```

2. Analyze heap dump:
   ```bash
   kubectl exec order-service-xxx -- jcmd 1 GC.heap_dump /tmp/heap.hprof
   kubectl cp order-service-xxx:/tmp/heap.hprof ./heap.hprof
   ```

3. Adjust JVM settings:
   ```yaml
   env:
     - name: JAVA_OPTS
       value: "-Xmx512m -Xms256m -XX:+UseG1GC"
   ```

## Logging

### Enable Debug Logging

```yaml
logging:
  level:
    com.shoppingcart.order: DEBUG
    org.springframework.security: DEBUG
    org.hibernate.SQL: DEBUG
```

### Log Locations

- Kubernetes: `kubectl logs -f deployment/order-service`
- Local: `./logs/order-service.log`

### Useful Log Patterns

```bash
# Security events
kubectl logs order-service-xxx | grep -E "(WARN|ERROR).*security"

# Database queries
kubectl logs order-service-xxx | grep "Hibernate:"

# RabbitMQ events
kubectl logs order-service-xxx | grep -i rabbit
```

## Health Checks

### Verify Service Health

```bash
# Liveness
curl http://localhost:8080/actuator/health/liveness

# Readiness
curl http://localhost:8080/actuator/health/readiness

# Full health
curl http://localhost:8080/actuator/health
```

### Check Dependencies

```bash
# Database
curl http://localhost:8080/actuator/health/db

# RabbitMQ (if enabled)
curl http://localhost:8080/actuator/health/rabbit
```

## Metrics

### Key Metrics to Monitor

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `http_server_requests_seconds` | Request latency | p99 > 1s |
| `hikaricp.connections.active` | Active DB connections | > 80% of max |
| `jvm_memory_used_bytes` | Memory usage | > 80% of limit |
| `order_events_published_total` | Events sent to RabbitMQ | - |

### Prometheus Queries

```promql
# Request rate
rate(http_server_requests_seconds_count{application="order-service"}[5m])

# Error rate
rate(http_server_requests_seconds_count{application="order-service",status=~"5.."}[5m])

# P99 latency
histogram_quantile(0.99, rate(http_server_requests_seconds_bucket{application="order-service"}[5m]))
```

## Support

For issues not covered here:
1. Check application logs
2. Review [Architecture docs](../architecture/README.md)
3. Open an issue in GitHub repository
