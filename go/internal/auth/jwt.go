package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

var (
	// ErrInvalidToken is returned when the token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when the token is expired
	ErrExpiredToken = errors.New("token expired")
	// ErrInvalidIssuer is returned when the issuer doesn't match
	ErrInvalidIssuer = errors.New("invalid issuer")
	// ErrJWKSFetch is returned when JWKS cannot be fetched
	ErrJWKSFetch = errors.New("failed to fetch JWKS")
)

// Claims represents the JWT claims
type Claims struct {
	Subject   string   `json:"sub"`
	Issuer    string   `json:"iss"`
	Audience  []string `json:"aud"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	Roles     []string `json:"roles"`
	Groups    []string `json:"groups"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
}

// JWTValidator validates JWT tokens using Keycloak JWKS
type JWTValidator struct {
	issuerURI  string
	clientID   string
	jwksURL    string
	keys       map[string]*rsa.PublicKey
	keysMutex  sync.RWMutex
	httpClient *http.Client
	logger     *zap.Logger
}

// NewJWTValidator creates a new JWT validator
func NewJWTValidator(issuerURI, clientID string, logger *zap.Logger) *JWTValidator {
	// Normalize issuer URI (remove trailing slash)
	issuerURI = strings.TrimSuffix(issuerURI, "/")

	return &JWTValidator{
		issuerURI: issuerURI,
		clientID:  clientID,
		jwksURL:   issuerURI + "/protocol/openid-connect/certs",
		keys:      make(map[string]*rsa.PublicKey),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// ValidateToken validates a JWT token and returns the claims
func (v *JWTValidator) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	// Parse token without validation first to get the key ID
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Get key ID from header
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: missing key ID", ErrInvalidToken)
	}

	// Get the public key
	publicKey, err := v.getPublicKey(ctx, kid)
	if err != nil {
		return nil, err
	}

	// Parse and validate token
	claims := jwt.MapClaims{}
	token, err = jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	// Validate issuer
	iss, _ := claims["iss"].(string)
	if iss != v.issuerURI {
		return nil, ErrInvalidIssuer
	}

	// Extract claims
	result := &Claims{
		Subject: getStringClaim(claims, "sub"),
		Issuer:  iss,
		Email:   getStringClaim(claims, "email"),
		Name:    getStringClaim(claims, "name"),
	}

	// Extract expiration
	if exp, ok := claims["exp"].(float64); ok {
		result.ExpiresAt = int64(exp)
	}
	if iat, ok := claims["iat"].(float64); ok {
		result.IssuedAt = int64(iat)
	}

	// Extract audience
	result.Audience = getAudienceClaim(claims)

	// Extract roles from Keycloak claims
	result.Roles = v.extractRoles(claims)

	// Extract groups
	result.Groups = getStringArrayClaim(claims, "groups")

	return result, nil
}

// extractRoles extracts roles from Keycloak-specific JWT claims
func (v *JWTValidator) extractRoles(claims jwt.MapClaims) []string {
	var roles []string

	// Extract from realm_access.roles
	if realmAccess, ok := claims["realm_access"].(map[string]interface{}); ok {
		if realmRoles, ok := realmAccess["roles"].([]interface{}); ok {
			for _, role := range realmRoles {
				if roleStr, ok := role.(string); ok {
					roles = append(roles, roleStr)
				}
			}
		}
	}

	// Extract from resource_access.<clientID>.roles
	if resourceAccess, ok := claims["resource_access"].(map[string]interface{}); ok {
		if clientAccess, ok := resourceAccess[v.clientID].(map[string]interface{}); ok {
			if clientRoles, ok := clientAccess["roles"].([]interface{}); ok {
				for _, role := range clientRoles {
					if roleStr, ok := role.(string); ok {
						roles = append(roles, roleStr)
					}
				}
			}
		}
	}

	// Extract from groups claim (remove leading slashes)
	if groups, ok := claims["groups"].([]interface{}); ok {
		for _, group := range groups {
			if groupStr, ok := group.(string); ok {
				// Convert group to role format
				role := strings.TrimPrefix(groupStr, "/")
				roles = append(roles, role)
			}
		}
	}

	return roles
}

// getPublicKey gets the public key for a given key ID
func (v *JWTValidator) getPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Check cache first
	v.keysMutex.RLock()
	if key, ok := v.keys[kid]; ok {
		v.keysMutex.RUnlock()
		return key, nil
	}
	v.keysMutex.RUnlock()

	// Fetch JWKS
	if err := v.fetchJWKS(ctx); err != nil {
		return nil, err
	}

	// Check again after fetch
	v.keysMutex.RLock()
	key, ok := v.keys[kid]
	v.keysMutex.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: key ID not found: %s", ErrInvalidToken, kid)
	}

	return key, nil
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// fetchJWKS fetches the JWKS from the issuer
func (v *JWTValidator) fetchJWKS(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrJWKSFetch, err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrJWKSFetch, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrJWKSFetch, resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("%w: %v", ErrJWKSFetch, err)
	}

	v.keysMutex.Lock()
	defer v.keysMutex.Unlock()

	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" || jwk.Use != "sig" {
			continue
		}

		key, err := jwkToRSAPublicKey(jwk)
		if err != nil {
			v.logger.Warn("failed to parse JWK",
				zap.String("kid", jwk.Kid),
				zap.Error(err),
			)
			continue
		}

		v.keys[jwk.Kid] = key
	}

	return nil
}

// jwkToRSAPublicKey converts a JWK to an RSA public key
func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode N (modulus)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)

	// Decode E (exponent)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}
	e := 0
	for _, b := range eBytes {
		e = e*256 + int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

// Helper functions

func getStringClaim(claims jwt.MapClaims, key string) string {
	if val, ok := claims[key].(string); ok {
		return val
	}
	return ""
}

func getStringArrayClaim(claims jwt.MapClaims, key string) []string {
	var result []string
	if arr, ok := claims[key].([]interface{}); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
	}
	return result
}

func getAudienceClaim(claims jwt.MapClaims) []string {
	switch aud := claims["aud"].(type) {
	case string:
		return []string{aud}
	case []interface{}:
		var result []string
		for _, v := range aud {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}
