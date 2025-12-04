# Cookie Authentication Implementation Guide

**Based on:** odata_mcp_go implementation
**Target:** vsp
**Date:** 2025-12-02

---

## Overview

The `odata_mcp_go` project implements cookie-based authentication as an alternative to basic authentication. This allows users to authenticate using:
1. A cookie file (Netscape format)
2. A cookie string (semicolon-separated key=value pairs)

This is useful for:
- SSO (Single Sign-On) scenarios where basic auth isn't available
- Session-based authentication
- Environments where credentials shouldn't be passed directly

---

## Current Authentication in vsp

Currently, `vsp` only supports basic authentication via environment variables:

```go
// pkg/adt/config.go
type Config struct {
    BaseURL            string
    Username           string
    Password           string
    Client             string
    Language           string
    InsecureSkipVerify bool
    StatefulSession    bool
    Timeout            time.Duration
}
```

---

## odata_mcp_go Implementation Analysis

### 1. Configuration Structure

**File:** `internal/config/config.go`

```go
type Config struct {
    // Authentication
    Username     string            `mapstructure:"username"`
    Password     string            `mapstructure:"password"`
    CookieFile   string            `mapstructure:"cookie_file"`
    CookieString string            `mapstructure:"cookie_string"`
    Cookies      map[string]string // Parsed cookies
    // ...
}

// HasBasicAuth returns true if username and password are configured
func (c *Config) HasBasicAuth() bool {
    return c.Username != "" && c.Password != ""
}

// HasCookieAuth returns true if cookies are configured
func (c *Config) HasCookieAuth() bool {
    return len(c.Cookies) > 0
}
```

### 2. Cookie Parsing Functions

**File:** `cmd/odata-mcp/main.go`

#### Loading from Netscape Cookie File

```go
func loadCookiesFromFile(cookieFile string) (map[string]string, error) {
    cookies := make(map[string]string)

    file, err := os.Open(cookieFile)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }

        // Parse Netscape format (7 fields separated by tabs)
        parts := strings.Split(line, "\t")
        if len(parts) >= 7 {
            // domain, flag, path, secure, expiration, name, value
            name := parts[5]
            value := parts[6]
            cookies[name] = value
        } else if strings.Contains(line, "=") {
            // Simple key=value format fallback
            kv := strings.SplitN(line, "=", 2)
            if len(kv) == 2 {
                cookies[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
            }
        }
    }

    return cookies, scanner.Err()
}
```

#### Parsing Cookie String

```go
func parseCookieString(cookieString string) map[string]string {
    cookies := make(map[string]string)
    for _, cookie := range strings.Split(cookieString, ";") {
        cookie = strings.TrimSpace(cookie)
        if strings.Contains(cookie, "=") {
            kv := strings.SplitN(cookie, "=", 2)
            if len(kv) == 2 {
                cookies[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
            }
        }
    }
    return cookies
}
```

### 3. HTTP Client Cookie Handling

**File:** `internal/client/client.go`

```go
type ODataClient struct {
    baseURL        string
    httpClient     *http.Client
    cookies        map[string]string         // User-provided cookies
    username       string
    password       string
    csrfToken      string
    verbose        bool
    sessionCookies []*http.Cookie            // Session cookies from server
    isV4           bool
}

// SetCookies configures cookie authentication
func (c *ODataClient) SetCookies(cookies map[string]string) {
    c.cookies = cookies
}

// buildRequest creates an HTTP request with proper headers and authentication
func (c *ODataClient) buildRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
    // ... create request ...

    // Set authentication - basic auth OR cookies (mutually exclusive)
    if c.username != "" && c.password != "" {
        req.SetBasicAuth(c.username, c.password)
    }

    // Set user-provided cookies
    for name, value := range c.cookies {
        req.AddCookie(&http.Cookie{
            Name:  name,
            Value: value,
        })
    }

    // Add session cookies received from server (important for CSRF)
    for _, cookie := range c.sessionCookies {
        req.AddCookie(cookie)
    }

    // ... CSRF token handling ...
    return req, nil
}
```

### 4. Session Cookie Management

The client tracks session cookies received from the server, which is important for CSRF token handling:

```go
// In fetchCSRFToken:
// Store any session cookies from the response
if cookies := resp.Cookies(); len(cookies) > 0 {
    c.sessionCookies = append(c.sessionCookies, cookies...)
    if c.verbose {
        fmt.Fprintf(os.Stderr, "[VERBOSE] Received %d session cookies during token fetch\n", len(cookies))
    }
}
```

### 5. CLI Flags and Environment Variables

**Command-line flags:**
```go
rootCmd.Flags().StringVar(&cfg.CookieFile, "cookie-file", "", "Path to cookie file in Netscape format")
rootCmd.Flags().StringVar(&cfg.CookieString, "cookie-string", "", "Cookie string (key1=val1; key2=val2)")
```

**Environment variables:**
- `ODATA_COOKIE_FILE` - Path to cookie file
- `ODATA_COOKIE_STRING` - Cookie string directly

### 6. Authentication Priority and Validation

```go
func processAuthentication(cfg *config.Config) error {
    // Check for mutually exclusive authentication options
    authMethods := 0
    if cfg.CookieFile != "" {
        authMethods++
    }
    if cfg.CookieString != "" {
        authMethods++
    }
    if cfg.Username != "" {
        authMethods++
    }

    if authMethods > 1 {
        return fmt.Errorf("only one authentication method can be used at a time")
    }
    // ... process authentication ...
}
```

---

## Implementation Plan for vsp

### Phase 1: Configuration Changes

**File:** `pkg/adt/config.go`

```go
type Config struct {
    BaseURL            string
    Username           string
    Password           string
    Client             string
    Language           string
    InsecureSkipVerify bool
    StatefulSession    bool
    Timeout            time.Duration

    // New cookie auth fields
    CookieFile   string            // Path to Netscape-format cookie file
    CookieString string            // Raw cookie string (key1=val1; key2=val2)
    Cookies      map[string]string // Parsed cookies
}

// Add helper methods
func (c *Config) HasBasicAuth() bool {
    return c.Username != "" && c.Password != ""
}

func (c *Config) HasCookieAuth() bool {
    return len(c.Cookies) > 0
}
```

### Phase 2: Cookie Parsing Functions

**New file:** `pkg/adt/cookies.go`

```go
package adt

import (
    "bufio"
    "os"
    "strings"
)

// LoadCookiesFromFile loads cookies from a Netscape-format cookie file
func LoadCookiesFromFile(cookieFile string) (map[string]string, error) {
    // Implementation from odata_mcp_go
}

// ParseCookieString parses a cookie string (key1=val1; key2=val2)
func ParseCookieString(cookieString string) map[string]string {
    // Implementation from odata_mcp_go
}
```

### Phase 3: HTTP Transport Changes

**File:** `pkg/adt/http.go`

Add cookie support to the Transport struct:

```go
type Transport struct {
    baseURL    string
    httpClient *http.Client
    username   string
    password   string
    cookies    map[string]string    // User-provided cookies
    csrfToken  string
    stateful   bool
    sessionCookies []*http.Cookie   // Server-provided session cookies
    // ...
}

// SetCookies sets user-provided cookies for authentication
func (t *Transport) SetCookies(cookies map[string]string) {
    t.cookies = cookies
}
```

Update `Request` method to include cookies:

```go
func (t *Transport) Request(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
    // ... create request ...

    // Set authentication
    if t.username != "" && t.password != "" {
        req.SetBasicAuth(t.username, t.password)
    }

    // Set user-provided cookies
    for name, value := range t.cookies {
        req.AddCookie(&http.Cookie{Name: name, Value: value})
    }

    // Set session cookies
    for _, cookie := range t.sessionCookies {
        req.AddCookie(cookie)
    }

    // ... rest of request handling ...
}
```

### Phase 4: Environment Variables

**File:** `cmd/vsp/main.go`

Add new environment variables:
- `SAP_COOKIE_FILE` - Path to cookie file
- `SAP_COOKIE_STRING` - Cookie string

```go
cfg := &mcp.Config{
    BaseURL:      getEnv("SAP_URL", ""),
    Username:     getEnv("SAP_USER", ""),
    Password:     getEnv("SAP_PASSWORD", ""),
    Client:       getEnv("SAP_CLIENT", "001"),
    Language:     getEnv("SAP_LANGUAGE", "EN"),
    CookieFile:   getEnv("SAP_COOKIE_FILE", ""),
    CookieString: getEnv("SAP_COOKIE_STRING", ""),
    // ...
}

// Process cookies
if cfg.CookieFile != "" {
    cookies, err := adt.LoadCookiesFromFile(cfg.CookieFile)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: failed to load cookie file: %v\n", err)
        os.Exit(1)
    }
    cfg.Cookies = cookies
} else if cfg.CookieString != "" {
    cfg.Cookies = adt.ParseCookieString(cfg.CookieString)
}
```

---

## Netscape Cookie File Format

The Netscape cookie file format has 7 tab-separated fields per line:

```
# Netscape HTTP Cookie File
# domain    flag    path    secure  expiration  name    value
.example.com    TRUE    /    FALSE    1234567890    session_id    abc123
.example.com    TRUE    /    FALSE    0    sap-usercontext    xyz789
```

Fields:
1. **domain** - The domain that created the cookie
2. **flag** - TRUE if all machines within a given domain can access the cookie
3. **path** - The path within the domain for which the cookie is valid
4. **secure** - TRUE if a secure connection is required
5. **expiration** - Unix timestamp when the cookie expires (0 = session cookie)
6. **name** - The name of the cookie
7. **value** - The cookie value

---

## Usage Examples

### Cookie File Authentication

```bash
# Create cookie file
cat > cookies.txt << 'EOF'
# Netscape HTTP Cookie File
.sap-system.com    TRUE    /    FALSE    0    sap-usercontext    abc123
.sap-system.com    TRUE    /    FALSE    0    SAP_SESSIONID    xyz789
EOF

# Use with vsp
SAP_URL=https://sap-system.com:44300 \
SAP_COOKIE_FILE=cookies.txt \
SAP_CLIENT=001 \
./vsp
```

### Cookie String Authentication

```bash
SAP_URL=https://sap-system.com:44300 \
SAP_COOKIE_STRING="sap-usercontext=abc123; SAP_SESSIONID=xyz789" \
SAP_CLIENT=001 \
./vsp
```

### Claude Desktop Config with Cookies

```json
{
  "mcpServers": {
    "abap-adt": {
      "command": "/path/to/vsp",
      "env": {
        "SAP_URL": "https://sap-system.com:44300",
        "SAP_COOKIE_STRING": "sap-usercontext=abc123; SAP_SESSIONID=xyz789",
        "SAP_CLIENT": "001",
        "SAP_LANGUAGE": "EN"
      }
    }
  }
}
```

---

## Important Considerations

### 1. Session Cookie Management

SAP ADT requires session cookies for stateful operations (CRUD). The implementation should:
- Store session cookies received from the server
- Send them with subsequent requests
- Handle cookie expiration gracefully

### 2. CSRF Token with Cookies

The CSRF token flow works the same with cookie auth:
1. Send request with `x-csrf-token: Fetch` header
2. Server returns CSRF token in response header
3. Server may also set session cookies
4. Use both token and cookies for subsequent modifying requests

### 3. Authentication Priority

Suggested priority (only one should be active):
1. Cookie file (`SAP_COOKIE_FILE`)
2. Cookie string (`SAP_COOKIE_STRING`)
3. Basic auth (`SAP_USER` + `SAP_PASSWORD`)
4. Anonymous (no auth)

### 4. Security Notes

- Cookie files should have restricted permissions (600)
- Cookie strings in environment variables are visible in process lists
- Consider clearing cookie values from verbose logs

---

## Testing

### Unit Test for Cookie Parsing

```go
func TestParseCookieString(t *testing.T) {
    tests := []struct {
        input    string
        expected map[string]string
    }{
        {
            input: "session=abc123; token=xyz789",
            expected: map[string]string{
                "session": "abc123",
                "token":   "xyz789",
            },
        },
        {
            input: "single=value",
            expected: map[string]string{
                "single": "value",
            },
        },
    }

    for _, tt := range tests {
        result := ParseCookieString(tt.input)
        if !reflect.DeepEqual(result, tt.expected) {
            t.Errorf("ParseCookieString(%q) = %v, want %v", tt.input, result, tt.expected)
        }
    }
}
```

### Integration Test

Test against SAP system with cookies exported from browser:
1. Login to SAP via browser
2. Export cookies using browser extension
3. Run integration tests with cookie file

---

## Files to Modify

| File | Changes |
|------|---------|
| `pkg/adt/config.go` | Add CookieFile, CookieString, Cookies fields and helper methods |
| `pkg/adt/cookies.go` | New file with cookie parsing functions |
| `pkg/adt/http.go` | Add cookie support to Transport |
| `cmd/vsp/main.go` | Add SAP_COOKIE_FILE and SAP_COOKIE_STRING env vars |
| `internal/mcp/server.go` | Pass cookies through to ADT client |
| `README.md` | Document cookie authentication |
| `CLAUDE.md` | Update with cookie auth info |

---

## Estimated Effort

- Configuration changes: 1 hour
- Cookie parsing functions: 1 hour
- HTTP transport changes: 2 hours
- Main.go changes: 1 hour
- Documentation: 1 hour
- Testing: 2 hours

**Total: ~8 hours**
