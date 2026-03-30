package inspection

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	teleAIOriginName = "teleai-cloud-auth-v1"
)

// GenTeleAIHeader generates the Authorization header value for a probe request.
// appID, appKey, region are per-probe values. headers/params come from the probe config.
func GenTeleAIHeader(
	appID, appKey, region, method, rawURL string,
	headers map[string]string,
	params map[string]string,
) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	path := parsedURL.Path
	if path == "" {
		path = "/"
	}
	// Merge URL query params with explicit probe params
	mergedParams := make(map[string]string)
	for k, v := range params {
		mergedParams[k] = v
	}
	for k, v := range parsedURL.Query() {
		if len(v) > 0 {
			mergedParams[k] = v[0]
		}
	}
	signedHeaders := "x-app-id"
	return teleAIGenAuthorization(appID, appKey, region, method, path, mergedParams, headers, signedHeaders, 1800)
}

func teleAIGenAuthorization(
	appID, appKey, region, method, path string,
	queryParams map[string]string,
	headers map[string]string,
	signedHeaders string,
	expirationInSeconds int,
) (string, error) {
	timestamp := time.Now().Unix()
	authStringPrefix := fmt.Sprintf("%s/%s/%s/%d/%d", teleAIOriginName, appID, region, timestamp, expirationInSeconds)
	signingKey, err := teleAIHmacSHA256Hex(authStringPrefix, appKey)
	if err != nil {
		return "", fmt.Errorf("生成 signingKey 失败: %w", err)
	}
	canonicalRequest, err := teleAIBuildCanonicalRequest(method, path, queryParams, headers, signedHeaders)
	if err != nil {
		return "", fmt.Errorf("构建 canonicalRequest 失败: %w", err)
	}
	signature, err := teleAIHmacSHA256Hex(canonicalRequest, signingKey)
	if err != nil {
		return "", fmt.Errorf("生成 signature 失败: %w", err)
	}
	return fmt.Sprintf("%s/%s/%s", authStringPrefix, signedHeaders, signature), nil
}

func teleAIBuildCanonicalRequest(method, path string, queryParams, headers map[string]string, signedHeaders string) (string, error) {
	canonicalMethod := strings.ToUpper(method)
	canonicalURI := teleAICanonicalizeURI(path)
	canonicalQS, err := teleAICanonicalizeQueryString(queryParams)
	if err != nil {
		return "", err
	}
	canonicalHdrs, err := teleAICanonicalizeHeaders(headers, signedHeaders)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s\n%s\n%s\n%s", canonicalMethod, canonicalURI, canonicalQS, canonicalHdrs), nil
}

func teleAICanonicalizeURI(path string) string {
	segments := strings.Split(path, "/")
	encoded := make([]string, len(segments))
	for i, seg := range segments {
		encoded[i] = url.PathEscape(seg)
	}
	return strings.Join(encoded, "/")
}

func teleAICanonicalizeQueryString(params map[string]string) (string, error) {
	if len(params) == 0 {
		return "", nil
	}
	pairs := make([]string, 0, len(params))
	for k, v := range params {
		encodedKey := url.QueryEscape(k)
		encodedVal := ""
		if v != "" {
			encodedVal = url.QueryEscape(v)
		}
		pairs = append(pairs, encodedKey+"="+encodedVal)
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "&"), nil
}

func teleAICanonicalizeHeaders(headers map[string]string, signedHeaders string) (string, error) {
	if len(headers) == 0 {
		return "", nil
	}
	lowerHeaders := make(map[string]string, len(headers))
	for k, v := range headers {
		lowerHeaders[strings.ToLower(k)] = v
	}
	headerNames := strings.Split(signedHeaders, ";")
	pairs := make([]string, 0, len(headerNames))
	for _, name := range headerNames {
		name = strings.TrimSpace(strings.ToLower(name))
		if val, ok := lowerHeaders[name]; ok && val != "" {
			encodedVal := url.QueryEscape(strings.TrimSpace(val))
			pairs = append(pairs, name+":"+encodedVal)
		}
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "\n"), nil
}

func teleAIHmacSHA256Hex(data, key string) (string, error) {
	mac := hmac.New(sha256.New, []byte(key))
	_, err := mac.Write([]byte(data))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}
