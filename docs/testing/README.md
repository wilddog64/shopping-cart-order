# Order Service — Testing Guide

## Overview
Testing relies on Maven with Spring Boot's built-in test framework plus Testcontainers for integration coverage. OWASP Dependency Check runs as part of `mvn verify` to guard supply-chain risk.

## Unit Tests
```bash
# Run unit tests only
mvn test
```
- Tests are under `src/test/java` mirroring the package structure (`controller`, `service`, `repository`, etc.).
- Use `@SpringBootTest` with slices (`@WebMvcTest`, `@DataJpaTest`) for fast feedback.

## Integration Tests
```bash
# Runs integration profile (spins up PostgreSQL + RabbitMQ via Testcontainers)
mvn verify -Pintegration
```
- Testcontainers setups live in `src/test/java/com/shoppingcart/order/integration`.
- Requires Docker available locally/inside CI runner.

## Coverage & Static Analysis
```bash
# Generate Jacoco coverage report
mvn test jacoco:report

# OWASP Dependency Check + unit tests
mvn verify
```
- Jacoco HTML report stored under `target/site/jacoco`.
- OWASP plugin runs during `verify`; `failOnError=false` until `NVD_API_KEY` is configured.

## Formatting & Linting
```bash
# Apply formatting
mvn spotless:apply

# Check formatting
mvn spotless:check
```

## CI Notes
- GitHub Actions workflow runs `mvn test`, `mvn verify`, and builds/pushes the container image.
- Set `PACKAGES_TOKEN` and `GH_TOKEN` secrets so Maven can read GitHub Packages artifacts.
