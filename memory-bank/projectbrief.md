# Project Brief: Order Service

## What This Project Does

The Order Service is a Java/Spring Boot microservice that manages the full lifecycle of customer orders in the Shopping Cart platform. It is the authoritative system of record for all orders, from initial placement through payment, fulfillment, shipping, and completion or cancellation.

## Core Responsibilities

- **Order creation**: Accept new orders (from the Basket Service checkout flow via RabbitMQ events or direct API calls) with line items, totals, and shipping address
- **Order lifecycle management**: Progress orders through a defined state machine (PENDING → PAID → PROCESSING → SHIPPED → COMPLETED, with CANCELLED as a terminal state from most steps)
- **Status tracking**: Record precise timestamps for each state transition (paidAt, shippedAt, completedAt, cancelledAt)
- **Shipping tracking**: Store tracking numbers and carrier information on shipment
- **Event publishing**: Publish RabbitMQ events on every order state change so downstream services (payment, inventory, notifications) can react asynchronously

## Goals

- Provide a reliable, transactional system of record for orders
- Enable event-driven communication with payment and fulfillment services
- Enforce valid order state transitions to prevent data integrity issues
- Support high-volume order processing with rate limiting and horizontal scaling

## Scope

**In scope:**
- Order creation and persistence (PostgreSQL)
- Order status lifecycle with state machine enforcement
- RabbitMQ event publishing for all state transitions
- REST API for order management
- JWT authentication via Keycloak
- Rate limiting, input sanitization, security headers
- Prometheus metrics via Spring Actuator
- Kubernetes deployment with ArgoCD integration

**Out of scope:**
- Payment processing (handled by shopping-cart-payment service — it publishes `payment.*` events this service may consume)
- Inventory management (handled by shopping-cart-product-catalog service)
- Cart management (handled by shopping-cart-basket service)
- Customer/user management (handled by Keycloak identity service)
- Email notifications (downstream consumer of order events)

## Service Context in the Platform

The Order Service sits in the middle of the purchase funnel. It receives checkout intent (from the Basket Service `cart.checkout` event or direct API call) and coordinates with Payment Service and fulfillment systems through RabbitMQ events. It is the source of truth for order status that customer-facing UIs query.

## Status

Active development — core order lifecycle is fully implemented and tested. The `rabbitmq-client-java` dependency is currently a local SNAPSHOT (`1.0.0-SNAPSHOT`) and must be installed to the local Maven repository before building.
