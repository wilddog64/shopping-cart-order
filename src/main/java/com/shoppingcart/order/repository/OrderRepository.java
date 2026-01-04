package com.shoppingcart.order.repository;

import com.shoppingcart.order.entity.Order;
import com.shoppingcart.order.entity.OrderStatus;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.time.Instant;
import java.util.List;
import java.util.UUID;

@Repository
public interface OrderRepository extends JpaRepository<Order, UUID> {

    List<Order> findByCustomerId(String customerId);

    List<Order> findByStatus(OrderStatus status);

    List<Order> findByCustomerIdAndStatus(String customerId, OrderStatus status);

    List<Order> findByCreatedAtBetween(Instant start, Instant end);

    List<Order> findByStatusAndCreatedAtBefore(OrderStatus status, Instant before);
}
