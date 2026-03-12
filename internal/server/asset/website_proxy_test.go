// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package asset

import (
	"testing"
)

func TestInjectBaseTag(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		basePath string
		want     string
	}{
		{
			name: "inject into empty head",
			html: `<!DOCTYPE html>
<html>
<head>
<title>Test</title>
</head>
<body></body>
</html>`,
			basePath: "/api/v1/websites/1/proxy/",
			want: `<!DOCTYPE html>
<html>
<head><base href="/api/v1/websites/1/proxy/">
<title>Test</title>
</head>
<body></body>
</html>`,
		},
		{
			name: "replace existing base tag",
			html: `<!DOCTYPE html>
<html>
<head>
<base href="/old/">
<title>Test</title>
</head>
<body></body>
</html>`,
			basePath: "/api/v1/websites/1/proxy/",
			want: `<!DOCTYPE html>
<html>
<head>
<base href="/api/v1/websites/1/proxy/">
<title>Test</title>
</head>
<body></body>
</html>`,
		},
		{
			name: "no head tag - insert after html",
			html: `<!DOCTYPE html>
<html>
<body></body>
</html>`,
			basePath: "/api/v1/websites/1/proxy/",
			want: `<!DOCTYPE html>
<html><head><base href="/api/v1/websites/1/proxy/"></head>
<body></body>
</html>`,
		},
		{
			name:     "no html tag - return original",
			html:     `<body>No HTML tag</body>`,
			basePath: "/api/v1/websites/1/proxy/",
			want:     `<body>No HTML tag</body>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(injectBaseTag([]byte(tt.html), tt.basePath))
			if got != tt.want {
				t.Errorf("injectBaseTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRewriteLocationHeader(t *testing.T) {
	tests := []struct {
		name          string
		location      string
		baseURL       string
		proxyBasePath string
		want          string
	}{
		{
			name:          "relative path",
			location:      "/login",
			baseURL:       "http://internal-app.local",
			proxyBasePath: "/api/v1/websites/1/proxy",
			want:          "/api/v1/websites/1/proxy/login",
		},
		{
			name:          "full URL matching baseURL",
			location:      "http://internal-app.local/dashboard",
			baseURL:       "http://internal-app.local",
			proxyBasePath: "/api/v1/websites/1/proxy",
			want:          "/api/v1/websites/1/proxy/dashboard",
		},
		{
			name:          "external URL - no change",
			location:      "http://external.com/page",
			baseURL:       "http://internal-app.local",
			proxyBasePath: "/api/v1/websites/1/proxy",
			want:          "http://external.com/page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteLocationHeader(tt.location, tt.baseURL, tt.proxyBasePath)
			if got != tt.want {
				t.Errorf("rewriteLocationHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRewriteSetCookieHeader(t *testing.T) {
	tests := []struct {
		name          string
		cookie        string
		proxyBasePath string
		wantContains  []string
		wantNotContains []string
	}{
		{
			name:          "remove domain and set path",
			cookie:        "session=xyz; Domain=internal-app.local; HttpOnly",
			proxyBasePath: "/api/v1/websites/1/proxy",
			wantContains:  []string{"session=xyz", "Path=/api/v1/websites/1/proxy", "HttpOnly"},
			wantNotContains: []string{"Domain="},
		},
		{
			name:          "replace existing path",
			cookie:        "session=xyz; Path=/; HttpOnly",
			proxyBasePath: "/api/v1/websites/1/proxy",
			wantContains:  []string{"session=xyz", "Path=/api/v1/websites/1/proxy", "HttpOnly"},
			wantNotContains: []string{"Path=/;"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteSetCookieHeader(tt.cookie, tt.proxyBasePath)
			for _, want := range tt.wantContains {
				if !contains(got, want) {
					t.Errorf("rewriteSetCookieHeader() = %v, want to contain %v", got, want)
				}
			}
			for _, notWant := range tt.wantNotContains {
				if contains(got, notWant) {
					t.Errorf("rewriteSetCookieHeader() = %v, should not contain %v", got, notWant)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
