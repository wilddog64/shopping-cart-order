# Shopping Cart Order Service - Makefile
# Java 21 + Spring Boot 3.2

.PHONY: help build test run clean package docker-build docker-run lint format check deps update-deps k8s-build k8s-deploy k8s-delete k8s-status k8s-logs k8s-describe k8s-restart k8s-port-forward k8s-shell argocd-status argocd-sync argocd-refresh argocd-diff argocd-history

# Default target
.DEFAULT_GOAL := help

# Java and Maven settings
JAVA_HOME ?= /home/linuxbrew/.linuxbrew/opt/openjdk@21
MVN := JAVA_HOME=$(JAVA_HOME) PATH=$(JAVA_HOME)/bin:$(PATH) mvn
DOCKER_IMAGE := shopping-cart-order
DOCKER_TAG := latest

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

##@ General

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\n${BLUE}Usage:${NC}\n  make ${GREEN}<target>${NC}\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  ${GREEN}%-20s${NC} %s\n", $$1, $$2 } /^##@/ { printf "\n${YELLOW}%s${NC}\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

build: ## Compile the project
	@echo "${BLUE}Building project...${NC}"
	$(MVN) compile -q

run: ## Run the application locally
	@echo "${BLUE}Starting Order Service...${NC}"
	$(MVN) spring-boot:run

run-dev: ## Run with development profile
	@echo "${BLUE}Starting Order Service (dev mode)...${NC}"
	$(MVN) spring-boot:run -Dspring-boot.run.profiles=dev

debug: ## Run with remote debugging enabled (port 5005)
	@echo "${BLUE}Starting Order Service with debug...${NC}"
	$(MVN) spring-boot:run -Dspring-boot.run.jvmArguments="-agentlib:jdwp=transport=dt_socket,server=y,suspend=n,address=*:5005"

##@ Testing

test: ## Run all tests
	@echo "${BLUE}Running tests...${NC}"
	$(MVN) test

test-unit: ## Run unit tests only
	@echo "${BLUE}Running unit tests...${NC}"
	$(MVN) test -Dtest="*Test" -DfailIfNoTests=false

test-integration: ## Run integration tests only
	@echo "${BLUE}Running integration tests...${NC}"
	$(MVN) verify -Pintegration -DskipUnitTests=true

test-security: ## Run security tests only
	@echo "${BLUE}Running security tests...${NC}"
	$(MVN) test -Dtest="*Security*,*RateLimit*,*Sanitizer*"

test-coverage: ## Run tests with coverage report
	@echo "${BLUE}Running tests with coverage...${NC}"
	$(MVN) test jacoco:report
	@echo "${GREEN}Coverage report: target/site/jacoco/index.html${NC}"

test-watch: ## Run tests in watch mode (requires mvnd)
	@echo "${BLUE}Running tests in watch mode...${NC}"
	$(MVN) fizzed-watcher:run -Dtest=true

##@ Code Quality

lint: ## Run linter (Checkstyle)
	@echo "${BLUE}Running linter...${NC}"
	$(MVN) checkstyle:check || true

format: ## Format code with Spotless
	@echo "${BLUE}Formatting code...${NC}"
	$(MVN) spotless:apply || echo "${YELLOW}Spotless not configured, skipping...${NC}"

format-check: ## Check code formatting
	@echo "${BLUE}Checking code format...${NC}"
	$(MVN) spotless:check || echo "${YELLOW}Spotless not configured, skipping...${NC}"

check: lint test ## Run all checks (lint + test)

##@ Build & Package

package: ## Build JAR package
	@echo "${BLUE}Building JAR package...${NC}"
	$(MVN) package -DskipTests
	@echo "${GREEN}JAR created: target/shopping-cart-order-*.jar${NC}"

package-native: ## Build native image with GraalVM
	@echo "${BLUE}Building native image...${NC}"
	$(MVN) -Pnative native:compile -DskipTests

clean: ## Clean build artifacts
	@echo "${BLUE}Cleaning build artifacts...${NC}"
	$(MVN) clean

rebuild: clean build ## Clean and rebuild

##@ Docker

docker-build: package ## Build Docker image
	@echo "${BLUE}Building Docker image...${NC}"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "${GREEN}Image built: $(DOCKER_IMAGE):$(DOCKER_TAG)${NC}"

docker-run: ## Run Docker container
	@echo "${BLUE}Running Docker container...${NC}"
	docker run --rm -p 8080:8080 \
		-e DB_HOST=host.docker.internal \
		-e RABBITMQ_HOST=host.docker.internal \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-push: ## Push Docker image to registry
	@echo "${BLUE}Pushing Docker image...${NC}"
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-compose-up: ## Start with docker-compose
	docker-compose up -d

docker-compose-down: ## Stop docker-compose
	docker-compose down

##@ Dependencies

deps: ## Display dependency tree
	@echo "${BLUE}Dependency tree:${NC}"
	$(MVN) dependency:tree

deps-updates: ## Check for dependency updates
	@echo "${BLUE}Checking for dependency updates...${NC}"
	$(MVN) versions:display-dependency-updates

deps-plugins: ## Check for plugin updates
	@echo "${BLUE}Checking for plugin updates...${NC}"
	$(MVN) versions:display-plugin-updates

update-deps: ## Update dependencies to latest versions
	@echo "${BLUE}Updating dependencies...${NC}"
	$(MVN) versions:use-latest-releases

##@ Database

db-migrate: ## Run database migrations (Flyway)
	@echo "${BLUE}Running database migrations...${NC}"
	$(MVN) flyway:migrate || echo "${YELLOW}Flyway not configured${NC}"

db-info: ## Show migration info
	$(MVN) flyway:info || echo "${YELLOW}Flyway not configured${NC}"

db-repair: ## Repair migration checksums
	$(MVN) flyway:repair || echo "${YELLOW}Flyway not configured${NC}"

##@ Documentation

docs: ## Generate API documentation
	@echo "${BLUE}Generating documentation...${NC}"
	$(MVN) javadoc:javadoc
	@echo "${GREEN}Docs: target/site/apidocs/index.html${NC}"

##@ Utilities

version: ## Show project version
	@$(MVN) help:evaluate -Dexpression=project.version -q -DforceStdout

java-version: ## Show Java version
	@$(JAVA_HOME)/bin/java -version

tree: ## Show project structure
	@tree -I 'target|node_modules|.git' -L 3

install-local: ## Install to local Maven repository
	@echo "${BLUE}Installing to local repository...${NC}"
	$(MVN) install -DskipTests

##@ Kubernetes

k8s-build: docker-build ## Build and tag for Kubernetes
	@echo "${BLUE}Tagging image for k3s...${NC}"
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) localhost:5000/$(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true

k8s-deploy: ## Deploy to Kubernetes (k3s)
	@echo "${BLUE}Deploying to Kubernetes...${NC}"
	kubectl apply -k k8s/base
	@echo "${GREEN}Deployment complete${NC}"

k8s-delete: ## Delete from Kubernetes
	@echo "${YELLOW}Deleting from Kubernetes...${NC}"
	kubectl delete -k k8s/base --ignore-not-found
	@echo "${GREEN}Resources deleted${NC}"

k8s-status: ## Show deployment status
	@echo "${BLUE}Deployment status:${NC}"
	kubectl get pods,svc,hpa -n shopping-cart-apps -l app.kubernetes.io/name=order-service

k8s-logs: ## Show pod logs
	@echo "${BLUE}Pod logs:${NC}"
	kubectl logs -n shopping-cart-apps -l app.kubernetes.io/name=order-service --tail=100 -f

k8s-describe: ## Describe deployment
	kubectl describe deployment order-service -n shopping-cart-apps

k8s-restart: ## Restart deployment
	@echo "${BLUE}Restarting deployment...${NC}"
	kubectl rollout restart deployment/order-service -n shopping-cart-apps

k8s-port-forward: ## Port forward to local (8080)
	@echo "${BLUE}Port forwarding to localhost:8080...${NC}"
	kubectl port-forward -n shopping-cart-apps svc/order-service 8080:80

k8s-shell: ## Open shell in pod
	@echo "${BLUE}Opening shell in pod...${NC}"
	kubectl exec -n shopping-cart-apps -it $$(kubectl get pods -n shopping-cart-apps -l app.kubernetes.io/name=order-service -o jsonpath='{.items[0].metadata.name}') -- /bin/sh

##@ ArgoCD

argocd-status: ## Show ArgoCD application status
	@echo "${BLUE}ArgoCD Application Status:${NC}"
	@kubectl get application order-service -n argocd -o wide 2>/dev/null || echo "Application not found in ArgoCD"

argocd-sync: ## Trigger ArgoCD sync
	@echo "${BLUE}Triggering ArgoCD sync...${NC}"
	@argocd app sync order-service 2>/dev/null || \
		kubectl patch application order-service -n argocd --type merge \
		-p '{"operation":{"initiatedBy":{"username":"admin"},"sync":{}}}' 2>/dev/null || \
		echo "ArgoCD CLI not available and kubectl patch failed"
	@echo "${GREEN}Sync triggered${NC}"

argocd-refresh: ## Refresh ArgoCD application (fetch latest from Git)
	@echo "${BLUE}Refreshing ArgoCD application...${NC}"
	@argocd app get order-service --refresh 2>/dev/null || \
		kubectl patch application order-service -n argocd --type merge \
		-p '{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"normal"}}}' 2>/dev/null || \
		echo "Refresh failed"

argocd-diff: ## Show ArgoCD diff (what would change)
	@echo "${BLUE}ArgoCD Diff:${NC}"
	@argocd app diff order-service 2>/dev/null || echo "ArgoCD CLI required for diff"

argocd-history: ## Show ArgoCD deployment history
	@echo "${BLUE}Deployment History:${NC}"
	@argocd app history order-service 2>/dev/null || \
		kubectl get application order-service -n argocd -o jsonpath='{.status.history}' | jq . 2>/dev/null || \
		echo "No history available"

##@ Production

prod-build: ## Build for production
	@echo "${BLUE}Building for production...${NC}"
	$(MVN) clean package -Pprod -DskipTests

prod-run: ## Run production JAR
	@echo "${BLUE}Running production build...${NC}"
	java -jar target/shopping-cart-order-*.jar --spring.profiles.active=prod
