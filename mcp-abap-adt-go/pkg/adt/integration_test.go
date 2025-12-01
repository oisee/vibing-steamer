//go:build integration

package adt

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// Integration tests require SAP_URL, SAP_USER, SAP_PASSWORD environment variables.
// Run with: go test -tags=integration -v ./pkg/adt/

func getIntegrationClient(t *testing.T) *Client {
	url := os.Getenv("SAP_URL")
	user := os.Getenv("SAP_USER")
	pass := os.Getenv("SAP_PASSWORD")

	if url == "" || user == "" || pass == "" {
		t.Skip("SAP_URL, SAP_USER, SAP_PASSWORD required for integration tests")
	}

	client := os.Getenv("SAP_CLIENT")
	if client == "" {
		client = "001"
	}
	lang := os.Getenv("SAP_LANGUAGE")
	if lang == "" {
		lang = "EN"
	}

	opts := []Option{
		WithClient(client),
		WithLanguage(lang),
		WithTimeout(30 * time.Second),
	}

	if os.Getenv("SAP_INSECURE") == "true" {
		opts = append(opts, WithInsecureSkipVerify())
	}

	return NewClient(url, user, pass, opts...)
}

func TestIntegration_SearchObject(t *testing.T) {
	client := getIntegrationClient(t)
	ctx := context.Background()

	results, err := client.SearchObject(ctx, "CL_*", 10)
	if err != nil {
		t.Fatalf("SearchObject failed: %v", err)
	}

	if len(results) == 0 {
		t.Log("No results found for CL_* search")
	} else {
		t.Logf("Found %d results", len(results))
		for i, r := range results {
			if i >= 3 {
				break
			}
			t.Logf("  %s (%s) - %s", r.Name, r.Type, r.Description)
		}
	}
}

func TestIntegration_GetProgram(t *testing.T) {
	client := getIntegrationClient(t)
	ctx := context.Background()

	// Try to get a standard SAP program
	source, err := client.GetProgram(ctx, "SAPMSSY0")
	if err != nil {
		t.Logf("Could not get SAPMSSY0: %v", err)
		// Try another common program
		source, err = client.GetProgram(ctx, "RS_ABAP_SOURCE_SCAN")
		if err != nil {
			t.Skipf("Could not retrieve any standard program: %v", err)
		}
	}

	if len(source) == 0 {
		t.Error("Program source is empty")
	} else {
		t.Logf("Retrieved %d characters of source code", len(source))
		// Show first 200 chars
		preview := source
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		t.Logf("Preview:\n%s", preview)
	}
}

func TestIntegration_GetClass(t *testing.T) {
	client := getIntegrationClient(t)
	ctx := context.Background()

	// Try to get a standard SAP class
	sources, err := client.GetClass(ctx, "CL_ABAP_TYPEDESCR")
	if err != nil {
		t.Skipf("Could not get CL_ABAP_TYPEDESCR: %v", err)
	}

	mainSource, ok := sources["main"]
	if !ok {
		t.Error("No main source in class")
	} else if len(mainSource) == 0 {
		t.Error("Main source is empty")
	} else {
		t.Logf("Retrieved %d characters of class source", len(mainSource))
	}
}

func TestIntegration_GetTableContents(t *testing.T) {
	client := getIntegrationClient(t)
	ctx := context.Background()

	// Get contents of T000 (clients table - should exist in any system)
	contents, err := client.GetTableContents(ctx, "T000", 5, "")
	if err != nil {
		t.Skipf("Could not get T000 contents: %v", err)
	}

	t.Logf("Retrieved %d columns, %d rows", len(contents.Columns), len(contents.Rows))

	if len(contents.Columns) == 0 {
		t.Error("No columns returned")
	}
	if len(contents.Rows) == 0 {
		t.Error("No rows returned")
	} else {
		t.Logf("First row: %v", contents.Rows[0])
	}
}

func TestIntegration_GetTableContentsWithQuery(t *testing.T) {
	client := getIntegrationClient(t)
	ctx := context.Background()

	// Get contents of T000 with SQL query (must be full SELECT statement)
	contents, err := client.GetTableContents(ctx, "T000", 10, "SELECT * FROM T000 WHERE MANDT = '001'")
	if err != nil {
		t.Skipf("Could not get T000 contents with query: %v", err)
	}

	t.Logf("Retrieved %d columns, %d rows (filtered)", len(contents.Columns), len(contents.Rows))

	// All rows should have MANDT = '001'
	for i, row := range contents.Rows {
		if mandt, ok := row["MANDT"].(string); ok && mandt != "001" {
			t.Errorf("Row %d has MANDT = %s, expected 001", i, mandt)
		}
	}
}

func TestIntegration_RunQuery(t *testing.T) {
	client := getIntegrationClient(t)
	ctx := context.Background()

	// Run a simple query
	contents, err := client.RunQuery(ctx, "SELECT MANDT, MTEXT FROM T000", 10)
	if err != nil {
		t.Skipf("Could not run query: %v", err)
	}

	t.Logf("Query returned %d columns, %d rows", len(contents.Columns), len(contents.Rows))

	// Should have exactly 2 columns (MANDT and MTEXT)
	if len(contents.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(contents.Columns))
	}

	if len(contents.Rows) > 0 {
		t.Logf("First row: %v", contents.Rows[0])
	}
}

func TestIntegration_GetTable(t *testing.T) {
	client := getIntegrationClient(t)
	ctx := context.Background()

	source, err := client.GetTable(ctx, "T000")
	if err != nil {
		t.Skipf("Could not get T000 source: %v", err)
	}

	if len(source) == 0 {
		t.Error("Table source is empty")
	} else {
		t.Logf("Retrieved %d characters of table source", len(source))
		// Show first 200 chars
		preview := source
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		t.Logf("Preview:\n%s", preview)
	}
}

func TestIntegration_GetPackage(t *testing.T) {
	client := getIntegrationClient(t)
	ctx := context.Background()

	pkg, err := client.GetPackage(ctx, "BASIS")
	if err != nil {
		t.Skipf("Could not get BASIS package: %v", err)
	}

	t.Logf("Package: %s", pkg.Name)
	t.Logf("Sub-packages: %d, Objects: %d", len(pkg.SubPackages), len(pkg.Objects))
}

// --- Development Tools Integration Tests ---

func TestIntegration_SyntaxCheck(t *testing.T) {
	client := getIntegrationClient(t)
	ctx := context.Background()

	// Test with valid ABAP code - using a simple report
	validCode := `REPORT ztest_syntax.
WRITE 'Hello World'.`

	results, err := client.SyntaxCheck(ctx, "/sap/bc/adt/programs/programs/ZTEST_SYNTAX", validCode)
	if err != nil {
		t.Logf("Syntax check call failed (might be expected if program doesn't exist): %v", err)
		// Try with invalid code to at least test the endpoint
		invalidCode := `REPORT ztest_syntax.
WRITEE 'Hello World'.` // intentional typo

		results, err = client.SyntaxCheck(ctx, "/sap/bc/adt/programs/programs/ZTEST_SYNTAX", invalidCode)
		if err != nil {
			t.Skipf("Syntax check endpoint not accessible: %v", err)
		}
	}

	t.Logf("Syntax check returned %d messages", len(results))
	for i, r := range results {
		if i >= 5 {
			break
		}
		t.Logf("  [%s] Line %d: %s", r.Severity, r.Line, r.Text)
	}
}

func TestIntegration_RunUnitTests(t *testing.T) {
	client := getIntegrationClient(t)
	ctx := context.Background()

	// Try to run unit tests on CL_ABAP_UNIT_ASSERT (which might have tests)
	flags := DefaultUnitTestFlags()
	result, err := client.RunUnitTests(ctx, "/sap/bc/adt/oo/classes/CL_ABAP_UNIT_ASSERT", &flags)
	if err != nil {
		// Try another common test class
		result, err = client.RunUnitTests(ctx, "/sap/bc/adt/oo/classes/CL_ABAP_TYPEDESCR", &flags)
		if err != nil {
			t.Skipf("Could not run unit tests: %v", err)
		}
	}

	t.Logf("Unit test result: %d test classes", len(result.Classes))
	for _, class := range result.Classes {
		t.Logf("  Class: %s (%s)", class.Name, class.RiskLevel)
		for _, method := range class.TestMethods {
			status := "PASS"
			if len(method.Alerts) > 0 {
				status = "FAIL"
			}
			t.Logf("    [%s] %s (%d µs)", status, method.Name, method.ExecutionTime)
		}
	}
}

// --- CRUD Integration Tests ---

// TestIntegration_CRUD_FullWorkflow tests the complete CRUD workflow:
// Create -> Lock -> Update -> Activate -> Unlock -> Delete
func TestIntegration_CRUD_FullWorkflow(t *testing.T) {
	client := getIntegrationClient(t)
	ctx := context.Background()

	// Use a unique test program name with timestamp to avoid conflicts
	timestamp := time.Now().Unix() % 100000 // Last 5 digits
	programName := fmt.Sprintf("ZMCP_%05d", timestamp)
	packageName := "$TMP" // Local package, no transport needed
	t.Logf("Test program name: %s", programName)

	// Step 1: Create a new program
	t.Log("Step 1: Creating program...")
	err := client.CreateObject(ctx, CreateObjectOptions{
		ObjectType:  ObjectTypeProgram,
		Name:        programName,
		Description: "Test program for MCP CRUD integration test",
		PackageName: packageName,
	})
	if err != nil {
		t.Fatalf("Failed to create program: %v", err)
	}
	t.Logf("Created program: %s", programName)

	// Cleanup: ensure we delete the program at the end
	defer func() {
		t.Log("Cleanup: Deleting program...")
		objectURL := GetObjectURL(ObjectTypeProgram, programName, "")

		// Lock for delete
		lock, err := client.LockObject(ctx, objectURL, "MODIFY")
		if err != nil {
			t.Logf("Cleanup: Failed to lock for delete: %v", err)
			return
		}

		err = client.DeleteObject(ctx, objectURL, lock.LockHandle, "")
		if err != nil {
			t.Logf("Cleanup: Failed to delete: %v", err)
			// Try to unlock
			client.UnlockObject(ctx, objectURL, lock.LockHandle)
		} else {
			t.Log("Cleanup: Program deleted successfully")
		}
	}()

	objectURL := GetObjectURL(ObjectTypeProgram, programName, "")
	t.Logf("Object URL: %s", objectURL)

	// Step 2: Lock the object
	t.Log("Step 2: Locking object...")
	lock, err := client.LockObject(ctx, objectURL, "MODIFY")
	if err != nil {
		t.Fatalf("Failed to lock object: %v", err)
	}
	t.Logf("Lock acquired: %s (local: %v)", lock.LockHandle, lock.IsLocal)

	// Step 3: Update the source
	t.Log("Step 3: Updating source...")
	newSource := `REPORT ztest_mcp_crud.
* Test program created by MCP CRUD integration test
WRITE 'Hello from MCP!'.`

	sourceURL := GetSourceURL(ObjectTypeProgram, programName, "")
	err = client.UpdateSource(ctx, sourceURL, newSource, lock.LockHandle, "")
	if err != nil {
		// Unlock before failing
		client.UnlockObject(ctx, objectURL, lock.LockHandle)
		t.Fatalf("Failed to update source: %v", err)
	}
	t.Log("Source updated successfully")

	// Step 4: Unlock the object (must unlock before activation)
	t.Log("Step 4: Unlocking object...")
	err = client.UnlockObject(ctx, objectURL, lock.LockHandle)
	if err != nil {
		t.Fatalf("Failed to unlock: %v", err)
	}
	t.Log("Object unlocked successfully")

	// Step 5: Activate the object
	t.Log("Step 5: Activating object...")
	activateResult, err := client.Activate(ctx, objectURL, programName)
	if err != nil {
		t.Fatalf("Failed to activate: %v", err)
	}
	t.Logf("Activation result: success=%v, messages=%d", activateResult.Success, len(activateResult.Messages))

	// Step 6: Verify the source was saved
	t.Log("Step 6: Verifying source...")
	source, err := client.GetProgram(ctx, programName)
	if err != nil {
		t.Fatalf("Failed to read back source: %v", err)
	}

	if !strings.Contains(source, "Hello from MCP") {
		t.Errorf("Source doesn't contain expected content")
	} else {
		t.Log("Source verified successfully")
	}

	t.Log("CRUD workflow completed successfully!")
}

// TestIntegration_LockUnlock tests just the lock/unlock cycle
func TestIntegration_LockUnlock(t *testing.T) {
	client := getIntegrationClient(t)
	ctx := context.Background()

	// Try to lock a standard program (should exist in any system)
	objectURL := "/sap/bc/adt/programs/programs/SAPMSSY0"

	lock, err := client.LockObject(ctx, objectURL, "MODIFY")
	if err != nil {
		t.Skipf("Could not lock SAPMSSY0: %v", err)
	}
	t.Logf("Lock acquired: handle=%s, isLocal=%v", lock.LockHandle, lock.IsLocal)

	// Immediately unlock
	err = client.UnlockObject(ctx, objectURL, lock.LockHandle)
	if err != nil {
		t.Fatalf("Failed to unlock: %v", err)
	}
	t.Log("Object unlocked successfully")
}
