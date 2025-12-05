# AMDP Debugging: Session Architecture & Solutions

**Date:** 2025-12-05
**Report ID:** 019
**Subject:** Analysis of AMDP debugger session requirements and proposed MCP-compatible solutions

---

## Executive Summary

AMDP (ABAP Managed Database Procedures) debugging via ADT REST API requires **persistent HTTP session context**, which conflicts with the stateless nature of MCP tool calls. This report analyzes the root cause and proposes three solutions ranked by implementation complexity.

| Solution | Complexity | Effort | Recommendation |
|----------|------------|--------|----------------|
| Session Pool Manager | High | 3-5 days | Best long-term |
| Stateful Client Mode | Medium | 1-2 days | Good balance |
| Cookie File Persistence | Low | 0.5 days | Quick workaround |

---

## 1. Technical Findings

### 1.1 ABAP vs AMDP Debugger Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    ABAP Debugger (Works)                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  MCP Call 1              MCP Call 2              MCP Call 3     │
│  ┌──────────┐           ┌──────────┐           ┌──────────┐    │
│  │SetExtBP  │           │DebugListen│          │DebugAttach│    │
│  └────┬─────┘           └────┬─────┘           └────┬─────┘    │
│       │                      │                      │           │
│       ▼                      ▼                      ▼           │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │         SAP Database: External Breakpoints Table        │   │
│  │         (User-level persistence, no session needed)     │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                   AMDP Debugger (Broken)                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  MCP Call 1              MCP Call 2 (FAILS)                     │
│  ┌──────────┐           ┌──────────┐                           │
│  │AMDPStart │           │AMDPResume│                           │
│  └────┬─────┘           └────┬─────┘                           │
│       │                      │                                  │
│       ▼                      ▼                                  │
│  ┌──────────┐           ┌──────────┐                           │
│  │Session A │           │Session B │  ← Different HTTP session │
│  │Cookie: X │           │Cookie: Y │  ← Can't access Session A │
│  └────┬─────┘           └────┬─────┘                           │
│       │                      │                                  │
│       ▼                      ✗                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │      HANA Kernel: Debug Context (Session-bound)         │   │
│  │      Locked to HTTP Session A's cookies                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Session Binding Components

AMDP debug sessions are bound to:

| Component | Storage | Lifetime |
|-----------|---------|----------|
| `SAP_SESSIONID_*` | HTTP Cookie | HTTP session |
| `sap-usercontext` | HTTP Cookie | HTTP session |
| HANA Session ID | Kernel memory | ~5-10 min timeout |
| Debug Context | Work process | Until released |

### 1.3 API Endpoints Verified

| Endpoint | Method | Status | Notes |
|----------|--------|--------|-------|
| `/sap/bc/adt/amdp/debugger/main` | POST | ✅ Works | Returns `<startParameters>` |
| `/sap/bc/adt/amdp/debugger/main/{id}` | GET | ⚠️ Needs session | Resume/wait |
| `/sap/bc/adt/amdp/debugger/main/{id}` | DELETE | ⚠️ Needs session | Stop |
| `/sap/bc/adt/amdp/debugger/main/{id}` | POST | ⚠️ Needs session | Step |

### 1.4 Response Formats

**Start Response:**
```xml
<amdpdbg:startParameters xmlns:amdpdbg="http://www.sap.com/adt/amdp/debugger">
  <amdpdbg:parameter amdpdbg:key="HANA_SESSION_ID"
                     amdpdbg:value="vhcala4hci:30203:300139"/>
</amdpdbg:startParameters>
```

**Error (Session Lost):**
```xml
<exc:exception>
  <type id="AMDB_DBG_Failure"/>
  <properties>
    <entry key="TEXT">Debugging for user "X" already in use</entry>
    <entry key="com.sap.adt.communicationFramework.subType">
      DEBUGGEE_CONTEXT_LOCKED_BY_ME
    </entry>
  </properties>
</exc:exception>
```

---

## 2. Proposed Solutions

### 2.1 Solution A: Cookie File Persistence (Quick Workaround)

**Concept:** Save HTTP session cookies to file after AMDPDebuggerStart, restore for subsequent calls.

**Implementation:**

```go
// pkg/adt/config.go - Add session persistence
type Config struct {
    // ... existing fields ...
    SessionFile string // Path to persist session cookies
}

// pkg/adt/http.go - Save/restore session
func (t *Transport) SaveSession(path string) error {
    cookies := t.client.Jar.Cookies(t.baseURL)
    // Serialize cookies to file
}

func (t *Transport) RestoreSession(path string) error {
    // Load cookies from file and set in jar
}
```

**MCP Tool Changes:**
```go
// AMDPDebuggerStart - save session after start
func handleAMDPDebuggerStart(...) {
    session, err := client.AMDPDebuggerStart(...)
    client.SaveSession("/tmp/amdp-session-" + session.MainID + ".cookies")
    // Return session info including cookie file path
}

// AMDPDebuggerResume - restore session before call
func handleAMDPDebuggerResume(...) {
    client.RestoreSession(sessionFile)
    result, err := client.AMDPDebuggerResume(...)
}
```

**Pros:**
- Minimal code changes
- Works with existing architecture
- User can manually manage session files

**Cons:**
- Session files on disk (security consideration)
- User must pass session file path in subsequent calls
- No automatic cleanup

---

### 2.2 Solution B: Stateful Client Mode (Recommended)

**Concept:** Add `--stateful` mode where vsp maintains a single HTTP client instance across all MCP calls.

**Implementation:**

```go
// internal/mcp/server.go - Add stateful mode
type Server struct {
    mcpServer    *server.MCPServer
    adtClient    *adt.Client
    stateful     bool
    debugSession *DebugSessionState // Persists across calls
}

type DebugSessionState struct {
    AMDPMainID  string
    AMDPUser    string
    StartTime   time.Time
    HTTPClient  *http.Client // Preserved client with cookies
}

// NewServer with stateful option
func NewServer(cfg *Config) *Server {
    s := &Server{
        stateful: cfg.Stateful,
    }
    if cfg.Stateful {
        // Create single HTTP client that persists
        s.adtClient = adt.NewClient(..., adt.WithPersistentClient())
    }
    return s
}
```

**CLI Flag:**
```bash
./vsp --stateful  # Maintains HTTP session across calls
```

**Pros:**
- Clean solution
- No files on disk
- Automatic session management
- Works for any session-bound operation

**Cons:**
- Requires architecture change
- Only works in single-user mode
- Session lost on vsp restart

---

### 2.3 Solution C: Session Pool Manager (Best Long-term)

**Concept:** Dedicated session manager that maintains pools of authenticated sessions per user/operation type.

**Implementation:**

```go
// pkg/session/manager.go
type SessionManager struct {
    mu       sync.RWMutex
    sessions map[string]*ManagedSession
    config   ManagerConfig
}

type ManagedSession struct {
    ID         string
    User       string
    Type       SessionType // AMDP, Debug, etc.
    HTTPClient *http.Client
    Created    time.Time
    LastUsed   time.Time
    State      interface{} // Type-specific state
}

type SessionType string
const (
    SessionTypeAMDP  SessionType = "amdp"
    SessionTypeDebug SessionType = "debug"
)

// Acquire session for AMDP debugging
func (m *SessionManager) AcquireAMDP(user string) (*ManagedSession, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    key := fmt.Sprintf("amdp:%s", user)
    if sess, ok := m.sessions[key]; ok {
        sess.LastUsed = time.Now()
        return sess, nil
    }

    // Create new session
    sess := &ManagedSession{
        ID:         uuid.New().String(),
        User:       user,
        Type:       SessionTypeAMDP,
        HTTPClient: createHTTPClient(),
        Created:    time.Now(),
    }
    m.sessions[key] = sess
    return sess, nil
}

// Release session
func (m *SessionManager) Release(id string) {
    // Mark for cleanup or immediate release
}

// Background cleanup of expired sessions
func (m *SessionManager) StartCleanup(interval, maxAge time.Duration) {
    go func() {
        ticker := time.NewTicker(interval)
        for range ticker.C {
            m.cleanupExpired(maxAge)
        }
    }()
}
```

**MCP Integration:**
```go
// internal/mcp/server.go
type Server struct {
    sessionMgr *session.SessionManager
}

func (s *Server) handleAMDPDebuggerStart(args map[string]interface{}) {
    user := getString(args, "user")

    // Acquire managed session
    sess, _ := s.sessionMgr.AcquireAMDP(user)

    // Create client with session's HTTP client
    client := adt.NewClientWithHTTP(sess.HTTPClient, s.config)

    // Start debug and store state
    result, _ := client.AMDPDebuggerStart(ctx, user, cascadeMode)
    sess.State = result

    return result
}
```

**Pros:**
- Production-ready solution
- Supports multiple users
- Automatic session lifecycle
- Extensible for other session-bound operations
- Memory-efficient with cleanup

**Cons:**
- Most complex implementation
- Requires careful concurrency handling
- Memory overhead for session storage

---

## 3. Implementation Roadmap

### Phase 1: Quick Fix (Solution A)
- Add cookie save/restore to Transport
- Update AMDP tools to use session files
- Document manual workflow
- **Timeline:** 0.5 days

### Phase 2: Stateful Mode (Solution B)
- Add `--stateful` flag
- Modify Server to preserve HTTP client
- Add session state tracking
- **Timeline:** 1-2 days

### Phase 3: Session Manager (Solution C)
- Implement SessionManager package
- Add session lifecycle management
- Integrate with MCP server
- Add cleanup and monitoring
- **Timeline:** 3-5 days

---

## 4. Recommended Approach

**Short-term (v2.11):** Implement Solution A (Cookie Persistence)
- Quick to implement
- Unblocks AMDP debugging
- Provides learning for better solution

**Medium-term (v2.12):** Implement Solution B (Stateful Mode)
- Add `--stateful` flag for debug workflows
- Clean up Solution A code

**Long-term (v3.0):** Implement Solution C (Session Pool)
- Full session management
- Multi-user support
- Production-ready debugging

---

## 5. Testing Strategy

### Test Class Created
```abap
CLASS zcl_adt_amdp_test DEFINITION PUBLIC.
  INTERFACES if_amdp_marker_hdb.

  CLASS-METHODS calc_sum
    IMPORTING iv_n TYPE i
    EXPORTING ev_sum TYPE i.
ENDCLASS.

CLASS zcl_adt_amdp_test IMPLEMENTATION.
  METHOD calc_sum BY DATABASE PROCEDURE FOR HDB
                  LANGUAGE SQLSCRIPT.
    DECLARE lv_i INTEGER;
    lv_total = 0;
    WHILE lv_i <= :iv_n DO
      lv_total = :lv_total + :lv_i;
      lv_i = :lv_i + 1;
    END WHILE;
    ev_sum = :lv_total;
  ENDMETHOD.
ENDCLASS.
```

### Test Workflow
1. Start AMDP debug session
2. Set breakpoint in `calc_sum` method
3. Run unit tests to trigger AMDP execution
4. Verify debugger catches breakpoint
5. Step through SQLScript code
6. Inspect variables
7. Stop session cleanly

---

## 6. References

### SAP Classes
| Class | Description |
|-------|-------------|
| `CL_AMDP_DBG_MAIN` | Main debugger controller |
| `CL_AMDP_DBG_ADT_RES_MAIN` | REST resource handler |
| `CL_AMDP_DBG_ADMIN` | Work process reservation |
| `CL_AMDP_DBG_SYS_DEBUG` | SQLScript debugger calls |

### ADT Endpoints
| Endpoint | Content-Type |
|----------|--------------|
| POST `/amdp/debugger/main` | `application/vnd.sap.adt.amdp.dbg.startmain.v1+xml` |
| GET `/amdp/debugger/main/{id}` | `application/xml` |
| DELETE `/amdp/debugger/main/{id}` | - |

---

## Appendix: Error Codes

| Error | Meaning | Resolution |
|-------|---------|------------|
| `DEBUGGEE_CONTEXT_LOCKED_BY_ME` | Session already exists | Wait for timeout or use same HTTP session |
| `ExceptionParameterNotFound: mainId` | Missing session ID | Start new session |
| `406 Not Acceptable` | Wrong Accept header | Use SAP-specific content type |
