# Shopping Cart Order Service - Multi-stage Dockerfile
# Java 21 / Spring Boot 3.2

# =============================================================================
# Stage 1: Build
# =============================================================================
FROM eclipse-temurin:21-jdk-alpine AS builder

WORKDIR /app

# Install Maven
RUN apk add --no-cache maven

# Copy pom.xml and checkstyle config first for dependency caching
COPY pom.xml checkstyle.xml ./

# Download dependencies (cached layer)
# GH_TOKEN secret authenticates against GitHub Packages (rabbitmq-client-java)
RUN --mount=type=secret,id=GH_TOKEN \
    mkdir -p /root/.m2 && \
    printf '<settings><servers><server><id>github-rabbitmq-client</id><username>x-access-token</username><password>%s</password></server></servers></settings>' \
      "$(cat /run/secrets/GH_TOKEN)" > /root/.m2/settings.xml && \
    mvn dependency:go-offline -B && \
    rm -f /root/.m2/settings.xml

# Copy source code
COPY src ./src

# Build the application
RUN mvn package -DskipTests -B

# Extract layers for optimized Docker image
RUN java -Djarmode=layertools -jar target/*.jar extract

# =============================================================================
# Stage 2: Runtime
# =============================================================================
FROM eclipse-temurin:21-jre-alpine

# Security: Run as non-root user
RUN addgroup -g 1000 spring && \
    adduser -u 1000 -G spring -s /bin/sh -D spring

WORKDIR /app

# Copy extracted layers
COPY --from=builder /app/dependencies/ ./
COPY --from=builder /app/spring-boot-loader/ ./
COPY --from=builder /app/snapshot-dependencies/ ./
COPY --from=builder /app/application/ ./

# Set ownership
RUN chown -R spring:spring /app

USER spring:spring

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/actuator/health || exit 1

# JVM options for containers
ENV JAVA_OPTS="-XX:+UseContainerSupport -XX:MaxRAMPercentage=75.0 -XX:InitialRAMPercentage=50.0"

# Run the application
ENTRYPOINT ["sh", "-c", "java $JAVA_OPTS org.springframework.boot.loader.launch.JarLauncher"]
