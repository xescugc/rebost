package storing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/mux"
)

// S3AuthMiddleware returns a gorilla/mux middleware that validates AWS Signature V4.
// If accessKey is empty, the middleware is a no-op and all requests pass through.
// Internal routes (/config, /replicas/) are always skipped.
// authMode controls which methods require auth: "write" skips auth for GET and HEAD;
// any other value (including "") requires auth for all methods.
func S3AuthMiddleware(accessKey, secretKey, authMode string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if accessKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Skip internal routes
			if r.URL.Path == "/config" || strings.HasPrefix(r.URL.Path, "/replicas/") {
				next.ServeHTTP(w, r)
				return
			}

			// In "write" mode, read-only methods bypass auth
			if authMode == "write" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
				next.ServeHTTP(w, r)
				return
			}

			if err := validateSigV4(r, accessKey, secretKey); err != nil {
				encodeS3Error(w, "AccessDenied", err.Error(), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// validateSigV4 validates the AWS Signature V4 on the given request.
// Region is intentionally not validated since Rebost has no geographic concept.
func validateSigV4(r *http.Request, accessKey, secretKey string) error {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return fmt.Errorf("missing Authorization header")
	}

	if !strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256 ") {
		return fmt.Errorf("unsupported authorization scheme")
	}

	// Parse Authorization header parts
	// Format: AWS4-HMAC-SHA256 Credential=<ak>/<date>/<region>/s3/aws4_request, SignedHeaders=<headers>, Signature=<sig>
	parts := strings.SplitN(strings.TrimPrefix(authHeader, "AWS4-HMAC-SHA256 "), ", ", 3)
	if len(parts) != 3 {
		return fmt.Errorf("malformed Authorization header")
	}

	credStr := strings.TrimPrefix(parts[0], "Credential=")
	signedHeadersStr := strings.TrimPrefix(parts[1], "SignedHeaders=")
	signature := strings.TrimPrefix(parts[2], "Signature=")

	credParts := strings.Split(credStr, "/")
	if len(credParts) != 5 {
		return fmt.Errorf("malformed Credential in Authorization header")
	}

	if credParts[0] != accessKey {
		return fmt.Errorf("invalid access key")
	}

	date := credParts[1]
	region := credParts[2]
	// credParts[3] == "s3", credParts[4] == "aws4_request"

	// Timestamp from x-amz-date header
	timestamp := r.Header.Get("x-amz-date")
	if timestamp == "" {
		return fmt.Errorf("missing x-amz-date header")
	}

	// Build canonical request
	canonicalRequest := buildCanonicalRequest(r, signedHeadersStr)

	// Build string to sign
	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", date, region)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		timestamp,
		credentialScope,
		hashSHA256(canonicalRequest),
	)

	// Derive signing key
	signingKey := deriveSigningKey(secretKey, date, region)

	// Compute expected signature
	expectedSig := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

func buildCanonicalRequest(r *http.Request, signedHeadersStr string) string {
	method := r.Method
	canonicalURI := r.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	// Canonical query string: sorted by key
	canonicalQuery := buildCanonicalQueryString(r)

	// Canonical headers: only signed headers, lowercased, trimmed
	signedHeaders := strings.Split(signedHeadersStr, ";")
	canonicalHeaders := buildCanonicalHeaders(r, signedHeaders)

	// Payload hash
	payloadHash := r.Header.Get("x-amz-content-sha256")
	if payloadHash == "" {
		payloadHash = "UNSIGNED-PAYLOAD"
	}

	return strings.Join([]string{
		method,
		canonicalURI,
		canonicalQuery,
		canonicalHeaders,
		signedHeadersStr,
		payloadHash,
	}, "\n")
}

func buildCanonicalQueryString(r *http.Request) string {
	q := r.URL.Query()
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		vals := q[k]
		sort.Strings(vals)
		for _, v := range vals {
			parts = append(parts, fmt.Sprintf("%s=%s", escape(k), escape(v)))
		}
	}
	return strings.Join(parts, "&")
}

func buildCanonicalHeaders(r *http.Request, signedHeaders []string) string {
	var sb strings.Builder
	for _, h := range signedHeaders {
		h = strings.ToLower(strings.TrimSpace(h))
		var val string
		if h == "host" {
			val = r.Host
		} else {
			val = strings.TrimSpace(r.Header.Get(h))
		}
		sb.WriteString(h)
		sb.WriteString(":")
		sb.WriteString(val)
		sb.WriteString("\n")
	}
	return sb.String()
}

func deriveSigningKey(secretKey, date, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, "s3")
	kSigning := hmacSHA256(kService, "aws4_request")
	return kSigning
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func hashSHA256(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// escape percent-encodes a string for use in a canonical query string.
// It encodes everything except unreserved chars: A-Z a-z 0-9 - _ . ~
func escape(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreserved(c) {
			sb.WriteByte(c)
		} else {
			fmt.Fprintf(&sb, "%%%02X", c)
		}
	}
	return sb.String()
}

func isUnreserved(c byte) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_' || c == '.' || c == '~'
}
