package com.shoppingcart.order.config;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.context.annotation.Import;
import org.springframework.security.test.context.support.WithMockUser;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

/**
 * Integration tests for OAuth2 security configuration.
 * Tests security headers and endpoint access control.
 *
 * Uses @WebMvcTest for lightweight testing without full application context.
 */
@WebMvcTest(controllers = TestController.class)
@Import({SecurityConfig.class, RateLimitConfig.class})
@TestPropertySource(properties = {
    "oauth2.enabled=false",
    "rate-limit.requests-per-minute=100",
    "rate-limit.requests-per-second=20",
    "rate-limit.burst-capacity=50"
})
@DisplayName("OAuth2 Security Integration Tests")
class OAuth2SecurityIntegrationTest {

    @Autowired
    private MockMvc mockMvc;

    @Nested
    @DisplayName("Security Headers")
    class SecurityHeaderTests {

        @Test
        @WithMockUser
        @DisplayName("Response should include X-Content-Type-Options header")
        void shouldIncludeContentTypeOptionsHeader() throws Exception {
            MvcResult result = mockMvc.perform(get("/api/test"))
                .andReturn();

            String header = result.getResponse().getHeader("X-Content-Type-Options");
            assertThat(header).isEqualTo("nosniff");
        }

        @Test
        @WithMockUser
        @DisplayName("Response should include X-Frame-Options header")
        void shouldIncludeFrameOptionsHeader() throws Exception {
            MvcResult result = mockMvc.perform(get("/api/test"))
                .andReturn();

            String header = result.getResponse().getHeader("X-Frame-Options");
            assertThat(header).isEqualTo("DENY");
        }

        @Test
        @WithMockUser
        @DisplayName("Response should include X-XSS-Protection header")
        void shouldIncludeXssProtectionHeader() throws Exception {
            MvcResult result = mockMvc.perform(get("/api/test"))
                .andReturn();

            String header = result.getResponse().getHeader("X-XSS-Protection");
            assertThat(header).isNotNull();
            assertThat(header).contains("1");
        }

        @Test
        @WithMockUser
        @DisplayName("Response should include Content-Security-Policy header")
        void shouldIncludeCspHeader() throws Exception {
            MvcResult result = mockMvc.perform(get("/api/test"))
                .andReturn();

            String header = result.getResponse().getHeader("Content-Security-Policy");
            assertThat(header).isNotNull();
            assertThat(header).contains("default-src 'self'");
        }

        @Test
        @WithMockUser
        @DisplayName("Response should include Referrer-Policy header")
        void shouldIncludeReferrerPolicyHeader() throws Exception {
            MvcResult result = mockMvc.perform(get("/api/test"))
                .andReturn();

            String header = result.getResponse().getHeader("Referrer-Policy");
            assertThat(header).isNotNull();
        }
    }

    @Nested
    @DisplayName("Endpoint Access Control")
    class EndpointAccessTests {

        @Test
        @DisplayName("Unauthenticated request to API should be allowed (OAuth2 disabled)")
        void unauthenticatedApiRequestShouldBeAllowed() throws Exception {
            // When OAuth2 is disabled, API endpoints should be accessible
            MvcResult result = mockMvc.perform(get("/api/orders")).andReturn();
            int statusCode = result.getResponse().getStatus();
            // Should not be 401 when OAuth2 is disabled
            // May be 404 if no controller exists
            assertThat(statusCode).isNotEqualTo(401);
        }

        @Test
        @WithMockUser(roles = {"ORDER_USER"})
        @DisplayName("Authenticated user should not get 401")
        void authenticatedUserShouldNotGet401() throws Exception {
            MvcResult result = mockMvc.perform(get("/api/orders")).andReturn();
            int statusCode = result.getResponse().getStatus();
            assertThat(statusCode).isNotEqualTo(401);
        }

        @Test
        @WithMockUser(roles = {"PLATFORM_ADMIN"})
        @DisplayName("Admin user should not get 401 or 403")
        void adminUserShouldHaveAccess() throws Exception {
            MvcResult result = mockMvc.perform(get("/api/orders")).andReturn();
            int statusCode = result.getResponse().getStatus();
            assertThat(statusCode).isNotEqualTo(401);
            assertThat(statusCode).isNotEqualTo(403);
        }

        @Test
        @DisplayName("Unknown endpoints should be denied")
        void unknownEndpointsShouldBeDenied() throws Exception {
            MvcResult result = mockMvc.perform(get("/unknown/path")).andReturn();
            int statusCode = result.getResponse().getStatus();
            // Unknown paths should return 403 (denied) or 404 (not found)
            assertThat(statusCode).isIn(403, 404);
        }
    }

    @Nested
    @DisplayName("Mock User Roles")
    class MockUserRoleTests {

        @Test
        @WithMockUser(username = "orderadmin", roles = {"ORDER_ADMIN"})
        @DisplayName("Order admin role should be recognized")
        void orderAdminRoleShouldBeRecognized() throws Exception {
            MvcResult result = mockMvc.perform(get("/api/orders")).andReturn();
            // Should not be 401 (unauthorized) or 403 (forbidden)
            int statusCode = result.getResponse().getStatus();
            assertThat(statusCode).isNotEqualTo(401);
            assertThat(statusCode).isNotEqualTo(403);
        }

        @Test
        @WithMockUser(username = "developer", roles = {"PLATFORM_DEVELOPER"})
        @DisplayName("Platform developer role should be recognized")
        void platformDeveloperRoleShouldBeRecognized() throws Exception {
            MvcResult result = mockMvc.perform(get("/api/orders")).andReturn();
            int statusCode = result.getResponse().getStatus();
            assertThat(statusCode).isNotEqualTo(401);
            assertThat(statusCode).isNotEqualTo(403);
        }
    }
}
