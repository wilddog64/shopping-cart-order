package com.shoppingcart.order.config;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Unit tests for InputSanitizer XSS prevention.
 */
class InputSanitizerTest {

    private InputSanitizer sanitizer;

    @BeforeEach
    void setUp() {
        sanitizer = new InputSanitizer();
    }

    @Test
    @DisplayName("Should handle null input")
    void shouldHandleNullInput() {
        assertThat(sanitizer.escapeHtml(null)).isNull();
        assertThat(sanitizer.removeScripts(null)).isNull();
        assertThat(sanitizer.sanitize(null)).isNull();
        assertThat(sanitizer.isSafeText(null)).isTrue();
        assertThat(sanitizer.containsXssPatterns(null)).isFalse();
    }

    @Test
    @DisplayName("Should escape HTML entities")
    void shouldEscapeHtmlEntities() {
        assertThat(sanitizer.escapeHtml("<script>alert('xss')</script>"))
            .isEqualTo("&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;");

        assertThat(sanitizer.escapeHtml("<img src=x onerror=alert(1)>"))
            .isEqualTo("&lt;img src=x onerror=alert(1)&gt;");

        assertThat(sanitizer.escapeHtml("Hello & goodbye"))
            .isEqualTo("Hello &amp; goodbye");

        assertThat(sanitizer.escapeHtml("\"quoted\" text"))
            .isEqualTo("&quot;quoted&quot; text");
    }

    @ParameterizedTest
    @DisplayName("Should detect XSS patterns")
    @ValueSource(strings = {
        "<script>alert('xss')</script>",
        "<SCRIPT>alert('xss')</SCRIPT>",
        "<script src='evil.js'></script>",
        "javascript:alert('xss')",
        "JAVASCRIPT:alert('xss')",
        "<img onerror=alert(1)>",
        "<div onclick=alert(1)>",
        "<a onmouseover=alert(1)>"
    })
    void shouldDetectXssPatterns(String input) {
        assertThat(sanitizer.containsXssPatterns(input)).isTrue();
    }

    @ParameterizedTest
    @DisplayName("Should not flag safe text as XSS")
    @ValueSource(strings = {
        "Hello, World!",
        "Order #12345",
        "Price: $29.99",
        "Email: user@example.com",
        "Product description with normal text",
        "JavaScript is a programming language" // Contains "JavaScript" but not "javascript:"
    })
    void shouldNotFlagSafeText(String input) {
        assertThat(sanitizer.containsXssPatterns(input)).isFalse();
    }

    @Test
    @DisplayName("Should remove script tags")
    void shouldRemoveScriptTags() {
        String input = "Hello <script>evil()</script> World";
        assertThat(sanitizer.removeScripts(input)).isEqualTo("Hello  World");
    }

    @Test
    @DisplayName("Should remove javascript: URLs")
    void shouldRemoveJavascriptUrls() {
        String input = "Click here: javascript:alert('xss')";
        assertThat(sanitizer.removeScripts(input)).isEqualTo("Click here: alert('xss')");
    }

    @Test
    @DisplayName("Should remove inline event handlers")
    void shouldRemoveInlineEventHandlers() {
        String input = "<img src=x onerror=alert(1)>";
        String result = sanitizer.removeScripts(input);
        assertThat(result).doesNotContain("onerror=");
    }

    @Test
    @DisplayName("Should fully sanitize malicious input")
    void shouldFullySanitize() {
        String input = "Hello <script>alert('xss')</script> World";
        String result = sanitizer.sanitize(input);

        assertThat(result).doesNotContain("<script>");
        assertThat(result).doesNotContain("</script>");
        assertThat(result).contains("Hello");
        assertThat(result).contains("World");
    }

    @Test
    @DisplayName("Should validate safe text patterns")
    void shouldValidateSafeText() {
        assertThat(sanitizer.isSafeText("Hello World")).isTrue();
        assertThat(sanitizer.isSafeText("Order #123")).isTrue();
        assertThat(sanitizer.isSafeText("Price: $29.99")).isTrue();
        assertThat(sanitizer.isSafeText("user@example.com")).isTrue();
    }

    @Test
    @DisplayName("Should sanitize for JSON output")
    void shouldSanitizeForJson() {
        assertThat(sanitizer.sanitizeForJson("Hello\nWorld"))
            .isEqualTo("Hello\\nWorld");

        assertThat(sanitizer.sanitizeForJson("Say \"Hello\""))
            .isEqualTo("Say \\\"Hello\\\"");

        assertThat(sanitizer.sanitizeForJson("Path\\to\\file"))
            .isEqualTo("Path\\\\to\\\\file");
    }

    @Test
    @DisplayName("Should handle multi-line script injection")
    void shouldHandleMultiLineScriptInjection() {
        String input = """
            Hello
            <script>
            alert('xss');
            document.cookie;
            </script>
            World
            """;

        assertThat(sanitizer.containsXssPatterns(input)).isTrue();
        assertThat(sanitizer.removeScripts(input)).doesNotContain("script");
    }

    @Test
    @DisplayName("Should handle encoded XSS attempts")
    void shouldHandleEncodedXss() {
        // HTML encoded - after HTML escaping these become safe
        String htmlEncoded = "&lt;script&gt;alert('xss')&lt;/script&gt;";
        // Already escaped, should be safe
        assertThat(sanitizer.containsXssPatterns(htmlEncoded)).isFalse();
    }

    @Test
    @DisplayName("Should preserve normal text content")
    void shouldPreserveNormalText() {
        String input = "Customer: John Doe, Order: #12345, Total: $99.99";
        assertThat(sanitizer.sanitize(input)).isEqualTo(input);
    }
}
