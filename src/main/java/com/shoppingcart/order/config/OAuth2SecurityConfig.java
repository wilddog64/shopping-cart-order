package com.shoppingcart.order.config;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.annotation.Order;
import org.springframework.core.convert.converter.Converter;
import org.springframework.security.config.annotation.method.configuration.EnableMethodSecurity;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity;
import org.springframework.security.config.annotation.web.configurers.AbstractHttpConfigurer;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.core.authority.SimpleGrantedAuthority;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.server.resource.authentication.JwtAuthenticationConverter;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.security.web.header.writers.ReferrerPolicyHeaderWriter;
import org.springframework.security.web.header.writers.XXssProtectionHeaderWriter;

import java.util.Collection;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;
import java.util.stream.Stream;

/**
 * OAuth2/OIDC Security configuration for the Order Service.
 *
 * Integrates with Keycloak for SSO authentication.
 * Enabled when oauth2.enabled=true in configuration.
 */
@Configuration
@EnableWebSecurity
@EnableMethodSecurity(prePostEnabled = true)
@ConditionalOnProperty(name = "oauth2.enabled", havingValue = "true")
@Order(1)
public class OAuth2SecurityConfig {

    @Value("${oauth2.resource-server.jwt.issuer-uri:}")
    private String issuerUri;

    @Bean
    public SecurityFilterChain oauth2SecurityFilterChain(HttpSecurity http) throws Exception {
        http
            // Disable CSRF for stateless REST API
            .csrf(AbstractHttpConfigurer::disable)

            // Stateless session management
            .sessionManagement(session -> session
                .sessionCreationPolicy(SessionCreationPolicy.STATELESS))

            // Security headers
            .headers(headers -> headers
                .xssProtection(xss -> xss
                    .headerValue(XXssProtectionHeaderWriter.HeaderValue.ENABLED_MODE_BLOCK))
                .frameOptions(frame -> frame.deny())
                .contentTypeOptions(contentType -> {})
                .contentSecurityPolicy(csp -> csp
                    .policyDirectives("default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; frame-ancestors 'none'; form-action 'self'"))
                .referrerPolicy(referrer -> referrer
                    .policy(ReferrerPolicyHeaderWriter.ReferrerPolicy.STRICT_ORIGIN_WHEN_CROSS_ORIGIN))
                .httpStrictTransportSecurity(hsts -> hsts
                    .includeSubDomains(true)
                    .maxAgeInSeconds(31536000))
                .permissionsPolicy(permissions -> permissions
                    .policy("geolocation=(), microphone=(), camera=()")))

            // Authorization rules
            .authorizeHttpRequests(auth -> auth
                // Public endpoints
                .requestMatchers("/actuator/health", "/actuator/info", "/actuator/prometheus").permitAll()
                .requestMatchers("/api/public/**").permitAll()

                // Admin endpoints require admin role
                .requestMatchers("/api/admin/**").hasAnyRole("ADMIN", "ORDER_ADMIN", "PLATFORM_ADMIN")

                // Order management requires order roles
                .requestMatchers("/api/orders/**").hasAnyRole("ORDER_USER", "ORDER_ADMIN", "PLATFORM_ADMIN", "PLATFORM_DEVELOPER")

                // All other API endpoints require authentication
                .requestMatchers("/api/**").authenticated()

                // Deny everything else
                .anyRequest().denyAll())

            // OAuth2 Resource Server with JWT
            .oauth2ResourceServer(oauth2 -> oauth2
                .jwt(jwt -> jwt
                    .jwtAuthenticationConverter(jwtAuthenticationConverter())));

        return http.build();
    }

    /**
     * Converts Keycloak JWT claims to Spring Security authorities.
     * Extracts roles from both realm_access.roles and resource_access claims.
     */
    @Bean
    public JwtAuthenticationConverter jwtAuthenticationConverter() {
        JwtAuthenticationConverter converter = new JwtAuthenticationConverter();
        converter.setJwtGrantedAuthoritiesConverter(new KeycloakGrantedAuthoritiesConverter());
        return converter;
    }

    /**
     * Custom converter for Keycloak JWT tokens.
     * Extracts roles and groups from Keycloak-specific claims.
     */
    static class KeycloakGrantedAuthoritiesConverter implements Converter<Jwt, Collection<GrantedAuthority>> {

        @Override
        public Collection<GrantedAuthority> convert(Jwt jwt) {
            // Extract realm roles
            Stream<String> realmRoles = extractRealmRoles(jwt);

            // Extract resource roles (client-specific)
            Stream<String> resourceRoles = extractResourceRoles(jwt);

            // Extract groups
            Stream<String> groups = extractGroups(jwt);

            // Combine all authorities
            return Stream.concat(Stream.concat(realmRoles, resourceRoles), groups)
                .map(role -> new SimpleGrantedAuthority("ROLE_" + role.toUpperCase().replace("-", "_")))
                .collect(Collectors.toList());
        }

        @SuppressWarnings("unchecked")
        private Stream<String> extractRealmRoles(Jwt jwt) {
            Map<String, Object> realmAccess = jwt.getClaim("realm_access");
            if (realmAccess == null) {
                return Stream.empty();
            }

            List<String> roles = (List<String>) realmAccess.get("roles");
            if (roles == null) {
                return Stream.empty();
            }

            return roles.stream();
        }

        @SuppressWarnings("unchecked")
        private Stream<String> extractResourceRoles(Jwt jwt) {
            Map<String, Object> resourceAccess = jwt.getClaim("resource_access");
            if (resourceAccess == null) {
                return Stream.empty();
            }

            // Extract roles from all resources
            return resourceAccess.values().stream()
                .filter(Map.class::isInstance)
                .map(Map.class::cast)
                .flatMap(resource -> {
                    List<String> roles = (List<String>) resource.get("roles");
                    return roles != null ? roles.stream() : Stream.empty();
                });
        }

        @SuppressWarnings("unchecked")
        private Stream<String> extractGroups(Jwt jwt) {
            List<String> groups = jwt.getClaim("groups");
            if (groups == null) {
                return Stream.empty();
            }

            // Remove leading slash from group paths if present
            return groups.stream()
                .map(group -> group.startsWith("/") ? group.substring(1) : group);
        }
    }
}
