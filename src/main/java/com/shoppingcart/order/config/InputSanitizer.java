package com.shoppingcart.order.config;

import org.springframework.stereotype.Component;
import org.springframework.web.util.HtmlUtils;

import java.util.regex.Pattern;

/**
 * Utility class for sanitizing user input to prevent XSS attacks.
 *
 * Provides methods to:
 * - Escape HTML entities
 * - Remove potentially dangerous content
 * - Validate input patterns
 */
@Component
public class InputSanitizer {

    // Pattern to detect potential script injection
    private static final Pattern SCRIPT_PATTERN = Pattern.compile(
        "<script[^>]*>.*?</script>|javascript:|on\\w+\\s*=",
        Pattern.CASE_INSENSITIVE | Pattern.DOTALL
    );

    // Pattern for valid alphanumeric with common punctuation
    private static final Pattern SAFE_TEXT_PATTERN = Pattern.compile(
        "^[\\p{L}\\p{N}\\s.,!?@#$%&*()\\-_+=\\[\\]{}|;:'\"/\\\\]*$"
    );

    /**
     * Escapes HTML entities to prevent XSS.
     *
     * @param input Raw user input
     * @return HTML-escaped string
     */
    public String escapeHtml(String input) {
        if (input == null) {
            return null;
        }
        return HtmlUtils.htmlEscape(input);
    }

    /**
     * Removes any potential script content from input.
     *
     * @param input Raw user input
     * @return Sanitized string with scripts removed
     */
    public String removeScripts(String input) {
        if (input == null) {
            return null;
        }
        return SCRIPT_PATTERN.matcher(input).replaceAll("");
    }

    /**
     * Full sanitization: removes scripts and escapes HTML.
     *
     * @param input Raw user input
     * @return Fully sanitized string
     */
    public String sanitize(String input) {
        if (input == null) {
            return null;
        }
        // First remove scripts, then escape remaining HTML
        String noScripts = removeScripts(input);
        return escapeHtml(noScripts);
    }

    /**
     * Validates that input contains only safe characters.
     *
     * @param input String to validate
     * @return true if input is safe, false otherwise
     */
    public boolean isSafeText(String input) {
        if (input == null) {
            return true;
        }
        return SAFE_TEXT_PATTERN.matcher(input).matches();
    }

    /**
     * Checks if input contains potential XSS patterns.
     *
     * @param input String to check
     * @return true if potentially dangerous content detected
     */
    public boolean containsXssPatterns(String input) {
        if (input == null) {
            return false;
        }
        return SCRIPT_PATTERN.matcher(input).find();
    }

    /**
     * Sanitizes for use in JSON response.
     * Escapes special JSON characters.
     *
     * @param input Raw input
     * @return JSON-safe string
     */
    public String sanitizeForJson(String input) {
        if (input == null) {
            return null;
        }
        return input
            .replace("\\", "\\\\")
            .replace("\"", "\\\"")
            .replace("\n", "\\n")
            .replace("\r", "\\r")
            .replace("\t", "\\t");
    }
}
