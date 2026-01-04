package com.shoppingcart.order.config;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.springframework.security.core.GrantedAuthority;
import org.springframework.security.oauth2.jwt.Jwt;

import java.time.Instant;
import java.util.Collection;
import java.util.List;
import java.util.Map;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Tests for OAuth2SecurityConfig and Keycloak JWT conversion.
 */
@DisplayName("OAuth2 Security Configuration Tests")
class OAuth2SecurityConfigTest {

    private OAuth2SecurityConfig.KeycloakGrantedAuthoritiesConverter converter;

    @BeforeEach
    void setUp() {
        converter = new OAuth2SecurityConfig.KeycloakGrantedAuthoritiesConverter();
    }

    @Nested
    @DisplayName("Keycloak JWT Authority Extraction")
    class KeycloakJwtAuthorityTests {

        @Test
        @DisplayName("Should extract realm roles from JWT")
        void shouldExtractRealmRoles() {
            // Given: JWT with realm_access.roles
            Jwt jwt = createJwtWithClaims(Map.of(
                "realm_access", Map.of(
                    "roles", List.of("platform-admin", "order-user")
                )
            ));

            // When
            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            // Then
            assertThat(authorities)
                .extracting(GrantedAuthority::getAuthority)
                .contains("ROLE_PLATFORM_ADMIN", "ROLE_ORDER_USER");
        }

        @Test
        @DisplayName("Should extract resource roles from JWT")
        void shouldExtractResourceRoles() {
            // Given: JWT with resource_access roles
            Jwt jwt = createJwtWithClaims(Map.of(
                "resource_access", Map.of(
                    "order-service", Map.of(
                        "roles", List.of("manage-orders", "view-orders")
                    ),
                    "account", Map.of(
                        "roles", List.of("manage-account")
                    )
                )
            ));

            // When
            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            // Then
            assertThat(authorities)
                .extracting(GrantedAuthority::getAuthority)
                .contains("ROLE_MANAGE_ORDERS", "ROLE_VIEW_ORDERS", "ROLE_MANAGE_ACCOUNT");
        }

        @Test
        @DisplayName("Should extract groups from JWT")
        void shouldExtractGroups() {
            // Given: JWT with groups claim
            Jwt jwt = createJwtWithClaims(Map.of(
                "groups", List.of("/platform-admins", "/order-admins", "developers")
            ));

            // When
            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            // Then
            assertThat(authorities)
                .extracting(GrantedAuthority::getAuthority)
                .contains("ROLE_PLATFORM_ADMINS", "ROLE_ORDER_ADMINS", "ROLE_DEVELOPERS");
        }

        @Test
        @DisplayName("Should combine realm roles, resource roles, and groups")
        void shouldCombineAllAuthorities() {
            // Given: JWT with all claim types
            Jwt jwt = createJwtWithClaims(Map.of(
                "realm_access", Map.of(
                    "roles", List.of("platform-developer")
                ),
                "resource_access", Map.of(
                    "order-service", Map.of(
                        "roles", List.of("order-admin")
                    )
                ),
                "groups", List.of("/argocd-developers")
            ));

            // When
            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            // Then
            assertThat(authorities)
                .extracting(GrantedAuthority::getAuthority)
                .containsExactlyInAnyOrder(
                    "ROLE_PLATFORM_DEVELOPER",
                    "ROLE_ORDER_ADMIN",
                    "ROLE_ARGOCD_DEVELOPERS"
                );
        }

        @Test
        @DisplayName("Should handle missing realm_access claim")
        void shouldHandleMissingRealmAccess() {
            // Given: JWT without realm_access
            Jwt jwt = createJwtWithClaims(Map.of(
                "groups", List.of("users")
            ));

            // When
            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            // Then
            assertThat(authorities)
                .extracting(GrantedAuthority::getAuthority)
                .contains("ROLE_USERS");
        }

        @Test
        @DisplayName("Should handle missing resource_access claim")
        void shouldHandleMissingResourceAccess() {
            // Given: JWT without resource_access
            Jwt jwt = createJwtWithClaims(Map.of(
                "realm_access", Map.of(
                    "roles", List.of("user")
                )
            ));

            // When
            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            // Then
            assertThat(authorities)
                .extracting(GrantedAuthority::getAuthority)
                .contains("ROLE_USER");
        }

        @Test
        @DisplayName("Should handle missing groups claim")
        void shouldHandleMissingGroups() {
            // Given: JWT without groups
            Jwt jwt = createJwtWithClaims(Map.of(
                "realm_access", Map.of(
                    "roles", List.of("admin")
                )
            ));

            // When
            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            // Then
            assertThat(authorities)
                .extracting(GrantedAuthority::getAuthority)
                .contains("ROLE_ADMIN");
        }

        @Test
        @DisplayName("Should handle empty JWT claims")
        void shouldHandleEmptyClaims() {
            // Given: JWT with no role/group claims
            Jwt jwt = createJwtWithClaims(Map.of());

            // When
            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            // Then
            assertThat(authorities).isEmpty();
        }

        @Test
        @DisplayName("Should normalize role names to uppercase with underscores")
        void shouldNormalizeRoleNames() {
            // Given: JWT with various role formats
            Jwt jwt = createJwtWithClaims(Map.of(
                "realm_access", Map.of(
                    "roles", List.of("platform-admin", "order_user", "CatalogViewer")
                )
            ));

            // When
            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            // Then
            assertThat(authorities)
                .extracting(GrantedAuthority::getAuthority)
                .contains("ROLE_PLATFORM_ADMIN", "ROLE_ORDER_USER", "ROLE_CATALOGVIEWER");
        }

        @Test
        @DisplayName("Should remove leading slash from group paths")
        void shouldRemoveLeadingSlashFromGroups() {
            // Given: JWT with groups having leading slashes
            Jwt jwt = createJwtWithClaims(Map.of(
                "groups", List.of("/platform-admins", "/nested/group", "no-slash")
            ));

            // When
            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            // Then
            assertThat(authorities)
                .extracting(GrantedAuthority::getAuthority)
                .contains("ROLE_PLATFORM_ADMINS", "ROLE_NESTED/GROUP", "ROLE_NO_SLASH");
        }

        @Test
        @DisplayName("Should handle null roles list in realm_access")
        void shouldHandleNullRolesInRealmAccess() {
            // Given: JWT with realm_access but null roles
            Jwt jwt = createJwtWithClaims(Map.of(
                "realm_access", Map.of(
                    "other_key", "value"
                )
            ));

            // When
            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            // Then
            assertThat(authorities).isEmpty();
        }

        @Test
        @DisplayName("Should handle empty roles list")
        void shouldHandleEmptyRolesList() {
            // Given: JWT with empty roles list
            Jwt jwt = createJwtWithClaims(Map.of(
                "realm_access", Map.of(
                    "roles", List.of()
                ),
                "groups", List.of()
            ));

            // When
            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            // Then
            assertThat(authorities).isEmpty();
        }
    }

    @Nested
    @DisplayName("Role-Based Access Control")
    class RoleBasedAccessTests {

        @Test
        @DisplayName("Admin role should be extracted from platform-admin")
        void adminRoleShouldBeExtracted() {
            Jwt jwt = createJwtWithClaims(Map.of(
                "realm_access", Map.of(
                    "roles", List.of("platform-admin")
                )
            ));

            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            assertThat(authorities)
                .extracting(GrantedAuthority::getAuthority)
                .contains("ROLE_PLATFORM_ADMIN");
        }

        @Test
        @DisplayName("Order admin role should be extracted")
        void orderAdminRoleShouldBeExtracted() {
            Jwt jwt = createJwtWithClaims(Map.of(
                "realm_access", Map.of(
                    "roles", List.of("order-admin")
                )
            ));

            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            assertThat(authorities)
                .extracting(GrantedAuthority::getAuthority)
                .contains("ROLE_ORDER_ADMIN");
        }

        @Test
        @DisplayName("Multiple service roles should be extracted")
        void multipleServiceRolesShouldBeExtracted() {
            Jwt jwt = createJwtWithClaims(Map.of(
                "resource_access", Map.of(
                    "order-service", Map.of("roles", List.of("admin")),
                    "catalog-service", Map.of("roles", List.of("viewer"))
                )
            ));

            Collection<GrantedAuthority> authorities = converter.convert(jwt);

            assertThat(authorities)
                .extracting(GrantedAuthority::getAuthority)
                .contains("ROLE_ADMIN", "ROLE_VIEWER");
        }
    }

    /**
     * Helper method to create a JWT with specific claims.
     */
    private Jwt createJwtWithClaims(Map<String, Object> claims) {
        Instant now = Instant.now();
        return Jwt.withTokenValue("test-token")
            .header("alg", "RS256")
            .header("typ", "JWT")
            .claim("sub", "test-user-id")
            .claim("iss", "http://keycloak.identity.svc.cluster.local/realms/shopping-cart")
            .claim("iat", now)
            .claim("exp", now.plusSeconds(3600))
            .claims(c -> c.putAll(claims))
            .build();
    }
}
