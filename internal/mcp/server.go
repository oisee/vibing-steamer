// Package mcp provides the MCP server implementation for ABAP ADT tools.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// Server wraps the MCP server with ADT client.
type Server struct {
	mcpServer *server.MCPServer
	adtClient *adt.Client
}

// Config holds MCP server configuration.
type Config struct {
	// SAP connection settings
	BaseURL            string
	Username           string
	Password           string
	Client             string
	Language           string
	InsecureSkipVerify bool

	// Cookie authentication (alternative to basic auth)
	Cookies map[string]string

	// Verbose output
	Verbose bool

	// Mode: focused or expert (default: focused)
	Mode string

	// Safety configuration
	ReadOnly        bool
	BlockFreeSQL    bool
	AllowedOps      string
	DisallowedOps   string
	AllowedPackages []string
}

// NewServer creates a new MCP server for ABAP ADT tools.
func NewServer(cfg *Config) *Server {
	// Create ADT client
	opts := []adt.Option{
		adt.WithClient(cfg.Client),
		adt.WithLanguage(cfg.Language),
	}
	if cfg.InsecureSkipVerify {
		opts = append(opts, adt.WithInsecureSkipVerify())
	}
	if len(cfg.Cookies) > 0 {
		opts = append(opts, adt.WithCookies(cfg.Cookies))
	}
	if cfg.Verbose {
		opts = append(opts, adt.WithVerbose())
	}

	// Configure safety settings
	safety := adt.UnrestrictedSafetyConfig() // Default: unrestricted for backwards compatibility
	if cfg.ReadOnly {
		safety.ReadOnly = true
	}
	if cfg.BlockFreeSQL {
		safety.BlockFreeSQL = true
	}
	if cfg.AllowedOps != "" {
		safety.AllowedOps = cfg.AllowedOps
	}
	if cfg.DisallowedOps != "" {
		safety.DisallowedOps = cfg.DisallowedOps
	}
	if len(cfg.AllowedPackages) > 0 {
		safety.AllowedPackages = cfg.AllowedPackages
	}
	opts = append(opts, adt.WithSafety(safety))

	adtClient := adt.NewClient(cfg.BaseURL, cfg.Username, cfg.Password, opts...)

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"mcp-abap-adt-go",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	s := &Server{
		mcpServer: mcpServer,
		adtClient: adtClient,
	}

	// Register tools based on mode
	s.registerTools(cfg.Mode)

	return s
}

// ServeStdio starts the MCP server on stdin/stdout.
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}

// registerTools registers ADT tools with the MCP server based on mode.
// Mode "focused" registers 17 essential tools (67% reduction).
// Mode "expert" registers all 45 tools.
func (s *Server) registerTools(mode string) {
	// Define focused mode tool whitelist (17 essential tools)
	focusedTools := map[string]bool{
		// Unified tools (2)
		"GetSource":   true,
		"WriteSource": true,

		// Search tools (3) - foundation
		"GrepObjects":  true, // Multi-object search (replaces GrepObject)
		"GrepPackages": true, // Multi-package + recursive (replaces GrepPackage)
		"SearchObject": true,

		// Primary workflow (1)
		"EditSource": true,

		// Data/Metadata read (5)
		"GetTable":            true,
		"GetTableContents":    true,
		"RunQuery":            true,
		"GetPackage":          true, // Metadata: package contents
		"GetFunctionGroup":    true, // Metadata: function module list
		"GetCDSDependencies":  true, // CDS dependency tree

		// Code intelligence (2)
		"FindDefinition":  true,
		"FindReferences":  true,

		// Development tools (2)
		"SyntaxCheck":   true,
		"RunUnitTests":  true,

		// Advanced/Edge cases (2)
		"LockObject":   true,
		"UnlockObject": true,

		// File-based operations (2)
		"ImportFromFile": true, // File → SAP (replaces DeployFromFile)
		"ExportToFile":   true, // SAP → File (replaces SaveToFile)
	}

	// Helper to check if tool should be registered
	shouldRegister := func(toolName string) bool {
		if mode == "expert" {
			return true // Expert mode: register all tools
		}
		return focusedTools[toolName] // Focused mode: only whitelisted tools
	}

	// Unified Tools (Focused Mode) - NEW
	if shouldRegister("GetSource") {
		s.registerGetSource()
	}
	if shouldRegister("WriteSource") {
		s.registerWriteSource()
	}


	// GetProgram
	if shouldRegister("GetProgram") {
		s.mcpServer.AddTool(mcp.NewTool("GetProgram",
		mcp.WithDescription("Retrieve ABAP program source code"),
		mcp.WithString("program_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP program"),
		),
	), s.handleGetProgram)
	}


	// GetClass
	if shouldRegister("GetClass") {
		s.mcpServer.AddTool(mcp.NewTool("GetClass",
		mcp.WithDescription("Retrieve ABAP class source code"),
		mcp.WithString("class_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP class"),
		),
	), s.handleGetClass)
	}


	// GetInterface
	if shouldRegister("GetInterface") {
		s.mcpServer.AddTool(mcp.NewTool("GetInterface",
		mcp.WithDescription("Retrieve ABAP interface source code"),
		mcp.WithString("interface_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP interface"),
		),
	), s.handleGetInterface)
	}


	// GetFunction
	if shouldRegister("GetFunction") {
		s.mcpServer.AddTool(mcp.NewTool("GetFunction",
		mcp.WithDescription("Retrieve ABAP Function Module source code"),
		mcp.WithString("function_name",
			mcp.Required(),
			mcp.Description("Name of the function module"),
		),
		mcp.WithString("function_group",
			mcp.Required(),
			mcp.Description("Name of the function group"),
		),
	), s.handleGetFunction)
	}


	// GetFunctionGroup
	if shouldRegister("GetFunctionGroup") {
		s.mcpServer.AddTool(mcp.NewTool("GetFunctionGroup",
		mcp.WithDescription("Retrieve ABAP Function Group source code"),
		mcp.WithString("function_group",
			mcp.Required(),
			mcp.Description("Name of the function group"),
		),
	), s.handleGetFunctionGroup)
	}


	// GetInclude
	if shouldRegister("GetInclude") {
		s.mcpServer.AddTool(mcp.NewTool("GetInclude",
		mcp.WithDescription("Retrieve ABAP Include Source Code"),
		mcp.WithString("include_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP Include"),
		),
	), s.handleGetInclude)
	}


	// GetTable
	if shouldRegister("GetTable") {
		s.mcpServer.AddTool(mcp.NewTool("GetTable",
		mcp.WithDescription("Retrieve ABAP table structure"),
		mcp.WithString("table_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP table"),
		),
	), s.handleGetTable)
	}


	// GetTableContents
	if shouldRegister("GetTableContents") {
		s.mcpServer.AddTool(mcp.NewTool("GetTableContents",
		mcp.WithDescription("Retrieve contents of an ABAP table"),
		mcp.WithString("table_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP table"),
		),
		mcp.WithNumber("max_rows",
			mcp.Description("Maximum number of rows to retrieve (default 100)"),
		),
		mcp.WithString("sql_query",
			mcp.Description("Optional full SELECT statement to filter results (e.g., \"SELECT * FROM T000 WHERE MANDT = '001'\")"),
		),
	), s.handleGetTableContents)
	}


	// RunQuery
	if shouldRegister("RunQuery") {
		s.mcpServer.AddTool(mcp.NewTool("RunQuery",
		mcp.WithDescription("Execute a freestyle SQL query against the SAP database"),
		mcp.WithString("sql_query",
			mcp.Required(),
			mcp.Description("SQL query to execute (e.g., \"SELECT * FROM T000 WHERE MANDT = '001'\")"),
		),
		mcp.WithNumber("max_rows",
			mcp.Description("Maximum number of rows to retrieve (default 100)"),
		),
	), s.handleRunQuery)
	}


	// GetCDSDependencies
	if shouldRegister("GetCDSDependencies") {
		s.mcpServer.AddTool(mcp.NewTool("GetCDSDependencies",
		mcp.WithDescription("Retrieve CDS view dependency tree showing all dependent objects (tables, views, associations)"),
		mcp.WithString("ddls_name",
			mcp.Required(),
			mcp.Description("CDS DDL source name (e.g., 'I_SalesOrder', 'ZDDL_MY_VIEW')"),
		),
		mcp.WithString("dependency_level",
			mcp.Description("Level of dependency resolution: 'unit' (direct only) or 'hierarchy' (recursive). Default: 'hierarchy'"),
		),
		mcp.WithBoolean("with_associations",
			mcp.Description("Include modeled associations in dependency tree. Default: false"),
		),
		mcp.WithString("context_package",
			mcp.Description("Filter dependencies to specific package context"),
		),
	), s.handleGetCDSDependencies)
	}


	// GetStructure
	if shouldRegister("GetStructure") {
		s.mcpServer.AddTool(mcp.NewTool("GetStructure",
		mcp.WithDescription("Retrieve ABAP Structure"),
		mcp.WithString("structure_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP Structure"),
		),
	), s.handleGetStructure)
	}


	// GetPackage
	if shouldRegister("GetPackage") {
		s.mcpServer.AddTool(mcp.NewTool("GetPackage",
		mcp.WithDescription("Retrieve ABAP package details"),
		mcp.WithString("package_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP package"),
		),
	), s.handleGetPackage)
	}


	// GetTransaction
	if shouldRegister("GetTransaction") {
		s.mcpServer.AddTool(mcp.NewTool("GetTransaction",
		mcp.WithDescription("Retrieve ABAP transaction details"),
		mcp.WithString("transaction_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP transaction"),
		),
	), s.handleGetTransaction)
	}


	// GetTypeInfo
	if shouldRegister("GetTypeInfo") {
		s.mcpServer.AddTool(mcp.NewTool("GetTypeInfo",
		mcp.WithDescription("Retrieve ABAP type information"),
		mcp.WithString("type_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP type"),
		),
	), s.handleGetTypeInfo)
	}


	// SearchObject
	if shouldRegister("SearchObject") {
		s.mcpServer.AddTool(mcp.NewTool("SearchObject",
		mcp.WithDescription("Search for ABAP objects using quick search"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query string (use * wildcard for partial match)"),
		),
		mcp.WithNumber("maxResults",
			mcp.Description("Maximum number of results to return (default 100)"),
		),
	), s.handleSearchObject)
	}


	// --- Development Tools ---

	// SyntaxCheck
	if shouldRegister("SyntaxCheck") {
		s.mcpServer.AddTool(mcp.NewTool("SyntaxCheck",
		mcp.WithDescription("Check ABAP source code for syntax errors"),
		mcp.WithString("object_url",
			mcp.Required(),
			mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("ABAP source code to check"),
		),
	), s.handleSyntaxCheck)
	}


	// Activate
	if shouldRegister("Activate") {
		s.mcpServer.AddTool(mcp.NewTool("Activate",
		mcp.WithDescription("Activate an ABAP object"),
		mcp.WithString("object_url",
			mcp.Required(),
			mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)"),
		),
		mcp.WithString("object_name",
			mcp.Required(),
			mcp.Description("Technical name of the object (e.g., ZTEST)"),
		),
	), s.handleActivate)
	}


	// RunUnitTests
	if shouldRegister("RunUnitTests") {
		s.mcpServer.AddTool(mcp.NewTool("RunUnitTests",
		mcp.WithDescription("Run ABAP Unit tests for an object"),
		mcp.WithString("object_url",
			mcp.Required(),
			mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST)"),
		),
		mcp.WithBoolean("include_dangerous",
			mcp.Description("Include dangerous risk level tests (default: false)"),
		),
		mcp.WithBoolean("include_long",
			mcp.Description("Include long duration tests (default: false)"),
		),
	), s.handleRunUnitTests)
	}


	// --- CRUD Operations ---

	// LockObject
	if shouldRegister("LockObject") {
		s.mcpServer.AddTool(mcp.NewTool("LockObject",
		mcp.WithDescription("Acquire an edit lock on an ABAP object"),
		mcp.WithString("object_url",
			mcp.Required(),
			mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)"),
		),
		mcp.WithString("access_mode",
			mcp.Description("Access mode: MODIFY (default) or READ"),
		),
	), s.handleLockObject)
	}


	// UnlockObject
	if shouldRegister("UnlockObject") {
		s.mcpServer.AddTool(mcp.NewTool("UnlockObject",
		mcp.WithDescription("Release an edit lock on an ABAP object"),
		mcp.WithString("object_url",
			mcp.Required(),
			mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)"),
		),
		mcp.WithString("lock_handle",
			mcp.Required(),
			mcp.Description("Lock handle from LockObject"),
		),
	), s.handleUnlockObject)
	}


	// UpdateSource
	if shouldRegister("UpdateSource") {
		s.mcpServer.AddTool(mcp.NewTool("UpdateSource",
		mcp.WithDescription("Write source code to an ABAP object (requires lock)"),
		mcp.WithString("object_url",
			mcp.Required(),
			mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)"),
		),
		mcp.WithString("source",
			mcp.Required(),
			mcp.Description("ABAP source code to write"),
		),
		mcp.WithString("lock_handle",
			mcp.Required(),
			mcp.Description("Lock handle from LockObject"),
		),
		mcp.WithString("transport",
			mcp.Description("Transport request number (optional for local packages)"),
		),
	), s.handleUpdateSource)
	}


	// CreateObject
	if shouldRegister("CreateObject") {
		s.mcpServer.AddTool(mcp.NewTool("CreateObject",
		mcp.WithDescription("Create a new ABAP object"),
		mcp.WithString("object_type",
			mcp.Required(),
			mcp.Description("Object type: PROG/P (program), CLAS/OC (class), INTF/OI (interface), PROG/I (include), FUGR/F (function group), FUGR/FF (function module), DEVC/K (package)"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Object name (e.g., ZTEST_PROGRAM)"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Object description"),
		),
		mcp.WithString("package_name",
			mcp.Required(),
			mcp.Description("Package name (e.g., $TMP for local, ZPACKAGE for transportable)"),
		),
		mcp.WithString("transport",
			mcp.Description("Transport request number (required for non-local packages)"),
		),
		mcp.WithString("parent_name",
			mcp.Description("Parent name (required for function modules - the function group name)"),
		),
	), s.handleCreateObject)
	}


	// DeleteObject
	if shouldRegister("DeleteObject") {
		s.mcpServer.AddTool(mcp.NewTool("DeleteObject",
		mcp.WithDescription("Delete an ABAP object (requires lock)"),
		mcp.WithString("object_url",
			mcp.Required(),
			mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)"),
		),
		mcp.WithString("lock_handle",
			mcp.Required(),
			mcp.Description("Lock handle from LockObject"),
		),
		mcp.WithString("transport",
			mcp.Description("Transport request number (optional for local packages)"),
		),
	), s.handleDeleteObject)
	}


	// --- Class Include Operations ---

	// GetClassInclude
	if shouldRegister("GetClassInclude") {
		s.mcpServer.AddTool(mcp.NewTool("GetClassInclude",
		mcp.WithDescription("Retrieve source code of a class include (definitions, implementations, macros, testclasses)"),
		mcp.WithString("class_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP class"),
		),
		mcp.WithString("include_type",
			mcp.Required(),
			mcp.Description("Include type: main, definitions, implementations, macros, testclasses"),
		),
	), s.handleGetClassInclude)
	}


	// CreateTestInclude
	if shouldRegister("CreateTestInclude") {
		s.mcpServer.AddTool(mcp.NewTool("CreateTestInclude",
		mcp.WithDescription("Create the test classes include for a class (required before writing test code)"),
		mcp.WithString("class_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP class"),
		),
		mcp.WithString("lock_handle",
			mcp.Required(),
			mcp.Description("Lock handle from LockObject (lock the parent class first)"),
		),
		mcp.WithString("transport",
			mcp.Description("Transport request number (optional for local packages)"),
		),
	), s.handleCreateTestInclude)
	}


	// UpdateClassInclude
	if shouldRegister("UpdateClassInclude") {
		s.mcpServer.AddTool(mcp.NewTool("UpdateClassInclude",
		mcp.WithDescription("Update source code of a class include (requires lock on parent class)"),
		mcp.WithString("class_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP class"),
		),
		mcp.WithString("include_type",
			mcp.Required(),
			mcp.Description("Include type: main, definitions, implementations, macros, testclasses"),
		),
		mcp.WithString("source",
			mcp.Required(),
			mcp.Description("ABAP source code to write"),
		),
		mcp.WithString("lock_handle",
			mcp.Required(),
			mcp.Description("Lock handle from LockObject (lock the parent class first)"),
		),
		mcp.WithString("transport",
			mcp.Description("Transport request number (optional for local packages)"),
		),
	), s.handleUpdateClassInclude)
	}


	// --- Workflow Tools ---

	// WriteProgram
	if shouldRegister("WriteProgram") {
		s.mcpServer.AddTool(mcp.NewTool("WriteProgram",
		mcp.WithDescription("Update an existing program with syntax check and activation (Lock -> SyntaxCheck -> Update -> Unlock -> Activate)"),
		mcp.WithString("program_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP program"),
		),
		mcp.WithString("source",
			mcp.Required(),
			mcp.Description("ABAP source code"),
		),
		mcp.WithString("transport",
			mcp.Description("Transport request number (optional for local packages)"),
		),
	), s.handleWriteProgram)
	}


	// WriteClass
	if shouldRegister("WriteClass") {
		s.mcpServer.AddTool(mcp.NewTool("WriteClass",
		mcp.WithDescription("Update an existing class with syntax check and activation (Lock -> SyntaxCheck -> Update -> Unlock -> Activate)"),
		mcp.WithString("class_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP class"),
		),
		mcp.WithString("source",
			mcp.Required(),
			mcp.Description("ABAP class source code (definition and implementation)"),
		),
		mcp.WithString("transport",
			mcp.Description("Transport request number (optional for local packages)"),
		),
	), s.handleWriteClass)
	}


	// CreateAndActivateProgram
	if shouldRegister("CreateAndActivateProgram") {
		s.mcpServer.AddTool(mcp.NewTool("CreateAndActivateProgram",
		mcp.WithDescription("Create a new program with source code and activate it (Create -> Lock -> Update -> Unlock -> Activate)"),
		mcp.WithString("program_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP program"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Program description"),
		),
		mcp.WithString("package_name",
			mcp.Required(),
			mcp.Description("Package name (e.g., $TMP for local)"),
		),
		mcp.WithString("source",
			mcp.Required(),
			mcp.Description("ABAP source code"),
		),
		mcp.WithString("transport",
			mcp.Description("Transport request number (required for non-local packages)"),
		),
	), s.handleCreateAndActivateProgram)
	}


	// CreateClassWithTests
	if shouldRegister("CreateClassWithTests") {
		s.mcpServer.AddTool(mcp.NewTool("CreateClassWithTests",
		mcp.WithDescription("Create a new class with unit tests and run them (Create -> Lock -> Update -> CreateTestInclude -> UpdateTest -> Unlock -> Activate -> RunTests)"),
		mcp.WithString("class_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP class"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Class description"),
		),
		mcp.WithString("package_name",
			mcp.Required(),
			mcp.Description("Package name (e.g., $TMP for local)"),
		),
		mcp.WithString("class_source",
			mcp.Required(),
			mcp.Description("ABAP class source code (definition and implementation)"),
		),
		mcp.WithString("test_source",
			mcp.Required(),
			mcp.Description("ABAP unit test source code"),
		),
		mcp.WithString("transport",
			mcp.Description("Transport request number (required for non-local packages)"),
		),
	), s.handleCreateClassWithTests)
	}


	// --- File-Based Deployment Tools ---

	// DeployFromFile (Recommended)
	if shouldRegister("DeployFromFile") {
		s.mcpServer.AddTool(mcp.NewTool("DeployFromFile",
		mcp.WithDescription("✅ RECOMMENDED - Smart deploy from file: auto-detects if object exists and creates/updates accordingly. Solves token limit problem for large generated files (ML models, 3948+ lines). Example: DeployFromFile(file_path=\"/path/to/zcl_ml_iris.clas.abap\", package_name=\"$ZAML_IRIS\") deploys any size file. Workflow: Parse → Check existence → Create or Update → Lock → SyntaxCheck → Write → Unlock → Activate. Supports .clas.abap, .prog.abap, .intf.abap, .fugr.abap, .func.abap. Use this for all file-based deployments."),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Absolute path to ABAP source file"),
		),
		mcp.WithString("package_name",
			mcp.Required(),
			mcp.Description("Package name (required for new objects, e.g., $ZAML_IRIS)"),
		),
		mcp.WithString("transport",
			mcp.Description("Transport request number (optional for local packages)"),
		),
	), s.handleDeployFromFile)
	}


	// SaveToFile
	if shouldRegister("SaveToFile") {
		s.mcpServer.AddTool(mcp.NewTool("SaveToFile",
		mcp.WithDescription("Save ABAP object source to local file (SAP → File). Enables BIDIRECTIONAL SYNC WORKFLOW: (1) SaveToFile downloads object from SAP, (2) edit locally with vim/VS Code/AI assistants, (3) DeployFromFile uploads changes back to SAP. Example: SaveToFile(objType=\"CLAS/OC\", objectName=\"ZCL_ML_IRIS\", outputPath=\"./src/\") creates ./src/zcl_ml_iris.clas.abap. Then edit locally and use DeployFromFile to sync back. Recommended for iterative development. Auto-determines file extension."),
		mcp.WithString("objType",
			mcp.Required(),
			mcp.Description("Object type: CLAS/OC (class), PROG/P (program), INTF/OI (interface), FUGR/F (function group), FUGR/FF (function module)"),
		),
		mcp.WithString("objectName",
			mcp.Required(),
			mcp.Description("Object name (e.g., ZCL_ML_IRIS, ZAML_IRIS_DEMO)"),
		),
		mcp.WithString("outputPath",
			mcp.Description("Output file path or directory. If directory, filename is auto-generated with correct extension. If omitted, saves to current directory."),
		),
	), s.handleSaveToFile)
	}

	// ImportFromFile (alias for DeployFromFile - File → SAP)
	if shouldRegister("ImportFromFile") {
		s.registerImportFromFile()
	}

	// ExportToFile (alias for SaveToFile - SAP → File)
	if shouldRegister("ExportToFile") {
		s.registerExportToFile()
	}


	// RenameObject
	if shouldRegister("RenameObject") {
		s.mcpServer.AddTool(mcp.NewTool("RenameObject",
		mcp.WithDescription("Rename ABAP object by creating copy with new name and deleting old one. Useful for fixing naming conventions. Workflow: GetSource → Replace names → CreateNew → ActivateNew → DeleteOld"),
		mcp.WithString("objType",
			mcp.Required(),
			mcp.Description("Object type: CLAS/OC (class), PROG/P (program), INTF/OI (interface), FUGR/F (function group)"),
		),
		mcp.WithString("oldName",
			mcp.Required(),
			mcp.Description("Current object name"),
		),
		mcp.WithString("newName",
			mcp.Required(),
			mcp.Description("New object name"),
		),
		mcp.WithString("packageName",
			mcp.Required(),
			mcp.Description("Package name for new object (e.g., $ZAML_IRIS)"),
		),
		mcp.WithString("transport",
			mcp.Description("Transport request number (optional for local packages)"),
		),
	), s.handleRenameObject)
	}


	// --- Surgical Edit Tools ---

	// EditSource
	if shouldRegister("EditSource") {
		s.mcpServer.AddTool(mcp.NewTool("EditSource",
		mcp.WithDescription("Surgical string replacement on ABAP source code. Matches the Edit tool pattern for local files. Workflow: GetSource → FindReplace → SyntaxCheck → Lock → Update → Unlock → Activate. Example: EditSource(object_url=\"/sap/bc/adt/programs/programs/ZTEST\", old_string=\"METHOD foo.\\n  ENDMETHOD.\", new_string=\"METHOD foo.\\n  rv_result = 42.\\n  ENDMETHOD.\", replace_all=false, syntax_check=true). Requires unique match if replace_all=false. Use this for incremental edits between syntax checks - no need to download/upload full source!"),
		mcp.WithString("object_url",
			mcp.Required(),
			mcp.Description("ADT URL of object (e.g., /sap/bc/adt/programs/programs/ZTEST, /sap/bc/adt/oo/classes/zcl_test)"),
		),
		mcp.WithString("old_string",
			mcp.Required(),
			mcp.Description("Exact string to find and replace. Must be unique in source if replace_all=false. Include enough context (surrounding lines) to ensure uniqueness."),
		),
		mcp.WithString("new_string",
			mcp.Required(),
			mcp.Description("Replacement string. Can be multiline (use \\n). Length can differ from old_string."),
		),
		mcp.WithBoolean("replace_all",
			mcp.Description("If true, replace all occurrences. If false (default), require unique match. Default: false"),
		),
		mcp.WithBoolean("syntax_check",
			mcp.Description("If true (default), validate syntax before saving. If syntax errors found, changes are NOT saved. Default: true"),
		),
		mcp.WithBoolean("case_insensitive",
			mcp.Description("If true, ignore case when matching old_string. Useful for renaming variables regardless of case. Default: false"),
		),
	), s.handleEditSource)
	}


	// --- Grep/Search Tools ---

	// GrepObject
	if shouldRegister("GrepObject") {
		s.mcpServer.AddTool(mcp.NewTool("GrepObject",
		mcp.WithDescription("Search for regex pattern in a single ABAP object's source code. Returns matches with line numbers and optional context. Use for finding TODO comments, string literals, patterns, or code snippets before editing."),
		mcp.WithString("object_url",
			mcp.Required(),
			mcp.Description("ADT URL of object (e.g., /sap/bc/adt/programs/programs/ZTEST)"),
		),
		mcp.WithString("pattern",
			mcp.Required(),
			mcp.Description("Regular expression pattern (Go regexp syntax). Examples: 'TODO', 'lv_\\w+', 'SELECT.*FROM'"),
		),
		mcp.WithBoolean("case_insensitive",
			mcp.Description("If true, perform case-insensitive matching. Default: false"),
		),
		mcp.WithNumber("context_lines",
			mcp.Description("Number of lines to show before/after each match (like grep -C). Default: 0"),
		),
	), s.handleGrepObject)
	}


	// GrepPackage
	if shouldRegister("GrepPackage") {
		s.mcpServer.AddTool(mcp.NewTool("GrepPackage",
		mcp.WithDescription("Search for regex pattern across all source objects in an ABAP package. Returns matches grouped by object. Use for package-wide analysis, finding patterns across multiple programs/classes."),
		mcp.WithString("package_name",
			mcp.Required(),
			mcp.Description("Package name (e.g., $TMP, ZPACKAGE)"),
		),
		mcp.WithString("pattern",
			mcp.Required(),
			mcp.Description("Regular expression pattern (Go regexp syntax). Examples: 'TODO', 'lv_\\w+', 'SELECT.*FROM'"),
		),
		mcp.WithBoolean("case_insensitive",
			mcp.Description("If true, perform case-insensitive matching. Default: false"),
		),
		mcp.WithString("object_types",
			mcp.Description("Comma-separated object types to search (e.g., 'PROG/P,CLAS/OC'). Empty = search all source objects. Valid: PROG/P, CLAS/OC, INTF/OI, FUGR/F, FUGR/FF, PROG/I"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of matching objects to return. 0 = unlimited. Default: 100"),
		),
	), s.handleGrepPackage)
	}

	// GrepObjects (unified multi-object search)
	if shouldRegister("GrepObjects") {
		s.registerGrepObjects()
	}

	// GrepPackages (unified multi-package search with recursive subpackages)
	if shouldRegister("GrepPackages") {
		s.registerGrepPackages()
	}


	// --- Code Intelligence Tools ---

	// FindDefinition
	if shouldRegister("FindDefinition") {
		s.mcpServer.AddTool(mcp.NewTool("FindDefinition",
		mcp.WithDescription("Navigate to the definition of a symbol at a given position in source code"),
		mcp.WithString("source_url",
			mcp.Required(),
			mcp.Description("ADT URL of the source file (e.g., /sap/bc/adt/programs/programs/ZTEST/source/main)"),
		),
		mcp.WithString("source",
			mcp.Required(),
			mcp.Description("Full source code of the file"),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("Line number (1-based)"),
		),
		mcp.WithNumber("start_column",
			mcp.Required(),
			mcp.Description("Start column of the symbol (1-based)"),
		),
		mcp.WithNumber("end_column",
			mcp.Required(),
			mcp.Description("End column of the symbol (1-based)"),
		),
		mcp.WithBoolean("implementation",
			mcp.Description("Navigate to implementation instead of definition (default: false)"),
		),
		mcp.WithString("main_program",
			mcp.Description("Main program for includes (optional)"),
		),
	), s.handleFindDefinition)
	}


	// FindReferences
	if shouldRegister("FindReferences") {
		s.mcpServer.AddTool(mcp.NewTool("FindReferences",
		mcp.WithDescription("Find all references to an ABAP object or symbol"),
		mcp.WithString("object_url",
			mcp.Required(),
			mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/oo/classes/ZCL_TEST)"),
		),
		mcp.WithNumber("line",
			mcp.Description("Line number for position-based reference search (1-based, optional)"),
		),
		mcp.WithNumber("column",
			mcp.Description("Column number for position-based reference search (1-based, optional)"),
		),
	), s.handleFindReferences)
	}


	// CodeCompletion
	if shouldRegister("CodeCompletion") {
		s.mcpServer.AddTool(mcp.NewTool("CodeCompletion",
		mcp.WithDescription("Get code completion suggestions at a position in source code"),
		mcp.WithString("source_url",
			mcp.Required(),
			mcp.Description("ADT URL of the source file (e.g., /sap/bc/adt/programs/programs/ZTEST/source/main)"),
		),
		mcp.WithString("source",
			mcp.Required(),
			mcp.Description("Full source code of the file"),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("Line number (1-based)"),
		),
		mcp.WithNumber("column",
			mcp.Required(),
			mcp.Description("Column number (1-based)"),
		),
	), s.handleCodeCompletion)
	}


	// PrettyPrint
	if shouldRegister("PrettyPrint") {
		s.mcpServer.AddTool(mcp.NewTool("PrettyPrint",
		mcp.WithDescription("Format ABAP source code using the pretty printer"),
		mcp.WithString("source",
			mcp.Required(),
			mcp.Description("ABAP source code to format"),
		),
	), s.handlePrettyPrint)
	}


	// GetPrettyPrinterSettings
	if shouldRegister("GetPrettyPrinterSettings") {
		s.mcpServer.AddTool(mcp.NewTool("GetPrettyPrinterSettings",
		mcp.WithDescription("Get the current pretty printer (code formatter) settings"),
	), s.handleGetPrettyPrinterSettings)
	}


	// SetPrettyPrinterSettings
	if shouldRegister("SetPrettyPrinterSettings") {
		s.mcpServer.AddTool(mcp.NewTool("SetPrettyPrinterSettings",
		mcp.WithDescription("Update the pretty printer (code formatter) settings"),
		mcp.WithBoolean("indentation",
			mcp.Required(),
			mcp.Description("Enable automatic indentation"),
		),
		mcp.WithString("style",
			mcp.Required(),
			mcp.Description("Keyword style: toLower, toUpper, keywordUpper, keywordLower, keywordAuto, none"),
		),
	), s.handleSetPrettyPrinterSettings)
	}


	// GetTypeHierarchy
	if shouldRegister("GetTypeHierarchy") {
		s.mcpServer.AddTool(mcp.NewTool("GetTypeHierarchy",
		mcp.WithDescription("Get the type hierarchy (supertypes or subtypes) for a class/interface"),
		mcp.WithString("source_url",
			mcp.Required(),
			mcp.Description("ADT URL of the source file"),
		),
		mcp.WithString("source",
			mcp.Required(),
			mcp.Description("Full source code of the file"),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("Line number (1-based)"),
		),
		mcp.WithNumber("column",
			mcp.Required(),
			mcp.Description("Column number (1-based)"),
		),
		mcp.WithBoolean("super_types",
			mcp.Description("Get supertypes instead of subtypes (default: false = subtypes)"),
		),
	), s.handleGetTypeHierarchy)
	}

}

// newToolResultError creates an error result for tool execution failures.
func newToolResultError(message string) *mcp.CallToolResult {
	result := mcp.NewToolResultText(message)
	result.IsError = true
	return result
}

// Tool handlers

func (s *Server) handleGetProgram(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	programName, ok := request.Params.Arguments["program_name"].(string)
	if !ok || programName == "" {
		return newToolResultError("program_name is required"), nil
	}

	source, err := s.adtClient.GetProgram(ctx, programName)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get program: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func (s *Server) handleGetClass(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.Params.Arguments["class_name"].(string)
	if !ok || className == "" {
		return newToolResultError("class_name is required"), nil
	}

	source, err := s.adtClient.GetClassSource(ctx, className)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get class: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func (s *Server) handleGetInterface(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	interfaceName, ok := request.Params.Arguments["interface_name"].(string)
	if !ok || interfaceName == "" {
		return newToolResultError("interface_name is required"), nil
	}

	source, err := s.adtClient.GetInterface(ctx, interfaceName)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get interface: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func (s *Server) handleGetFunction(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	functionName, ok := request.Params.Arguments["function_name"].(string)
	if !ok || functionName == "" {
		return newToolResultError("function_name is required"), nil
	}

	functionGroup, ok := request.Params.Arguments["function_group"].(string)
	if !ok || functionGroup == "" {
		return newToolResultError("function_group is required"), nil
	}

	source, err := s.adtClient.GetFunction(ctx, functionName, functionGroup)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get function: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func (s *Server) handleGetFunctionGroup(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	groupName, ok := request.Params.Arguments["function_group"].(string)
	if !ok || groupName == "" {
		return newToolResultError("function_group is required"), nil
	}

	fg, err := s.adtClient.GetFunctionGroup(ctx, groupName)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get function group: %v", err)), nil
	}

	result, _ := json.MarshalIndent(fg, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleGetInclude(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	includeName, ok := request.Params.Arguments["include_name"].(string)
	if !ok || includeName == "" {
		return newToolResultError("include_name is required"), nil
	}

	source, err := s.adtClient.GetInclude(ctx, includeName)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get include: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func (s *Server) handleGetTable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tableName, ok := request.Params.Arguments["table_name"].(string)
	if !ok || tableName == "" {
		return newToolResultError("table_name is required"), nil
	}

	source, err := s.adtClient.GetTable(ctx, tableName)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get table: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func (s *Server) handleGetTableContents(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tableName, ok := request.Params.Arguments["table_name"].(string)
	if !ok || tableName == "" {
		return newToolResultError("table_name is required"), nil
	}

	maxRows := 100
	if mr, ok := request.Params.Arguments["max_rows"].(float64); ok && mr > 0 {
		maxRows = int(mr)
	}

	sqlQuery := ""
	if sq, ok := request.Params.Arguments["sql_query"].(string); ok {
		sqlQuery = sq
	}

	contents, err := s.adtClient.GetTableContents(ctx, tableName, maxRows, sqlQuery)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get table contents: %v", err)), nil
	}

	result, _ := json.MarshalIndent(contents, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleRunQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sqlQuery, ok := request.Params.Arguments["sql_query"].(string)
	if !ok || sqlQuery == "" {
		return newToolResultError("sql_query is required"), nil
	}

	maxRows := 100
	if mr, ok := request.Params.Arguments["max_rows"].(float64); ok && mr > 0 {
		maxRows = int(mr)
	}

	contents, err := s.adtClient.RunQuery(ctx, sqlQuery, maxRows)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to run query: %v", err)), nil
	}

	result, _ := json.MarshalIndent(contents, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleGetCDSDependencies(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ddlsName, ok := request.Params.Arguments["ddls_name"].(string)
	if !ok || ddlsName == "" {
		return newToolResultError("ddls_name is required"), nil
	}

	opts := adt.CDSDependencyOptions{
		DependencyLevel:  "hierarchy",
		WithAssociations: false,
	}

	if level, ok := request.Params.Arguments["dependency_level"].(string); ok && level != "" {
		opts.DependencyLevel = level
	}

	if assoc, ok := request.Params.Arguments["with_associations"].(bool); ok {
		opts.WithAssociations = assoc
	}

	if pkg, ok := request.Params.Arguments["context_package"].(string); ok && pkg != "" {
		opts.ContextPackage = pkg
	}

	dependencyTree, err := s.adtClient.GetCDSDependencies(ctx, ddlsName, opts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get CDS dependencies: %v", err)), nil
	}

	// Add metadata summary
	summary := map[string]interface{}{
		"ddls_name":       ddlsName,
		"dependency_tree": dependencyTree,
		"statistics": map[string]interface{}{
			"total_dependencies": len(dependencyTree.FlattenDependencies()) - 1, // -1 to exclude root
			"dependency_depth":   dependencyTree.GetDependencyDepth(),
			"by_type":            dependencyTree.CountDependenciesByType(),
			"table_dependencies": len(dependencyTree.GetTableDependencies()),
			"inactive_dependencies": len(dependencyTree.GetInactiveDependencies()),
			"cycles":             dependencyTree.FindCycles(),
		},
	}

	jsonResult, _ := json.MarshalIndent(summary, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

func (s *Server) handleGetStructure(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	structName, ok := request.Params.Arguments["structure_name"].(string)
	if !ok || structName == "" {
		return newToolResultError("structure_name is required"), nil
	}

	source, err := s.adtClient.GetStructure(ctx, structName)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get structure: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func (s *Server) handleGetPackage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	packageName, ok := request.Params.Arguments["package_name"].(string)
	if !ok || packageName == "" {
		return newToolResultError("package_name is required"), nil
	}

	pkg, err := s.adtClient.GetPackage(ctx, packageName)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get package: %v", err)), nil
	}

	result, _ := json.MarshalIndent(pkg, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleGetTransaction(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tcode, ok := request.Params.Arguments["transaction_name"].(string)
	if !ok || tcode == "" {
		return newToolResultError("transaction_name is required"), nil
	}

	tran, err := s.adtClient.GetTransaction(ctx, tcode)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get transaction: %v", err)), nil
	}

	result, _ := json.MarshalIndent(tran, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleGetTypeInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	typeName, ok := request.Params.Arguments["type_name"].(string)
	if !ok || typeName == "" {
		return newToolResultError("type_name is required"), nil
	}

	typeInfo, err := s.adtClient.GetTypeInfo(ctx, typeName)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get type info: %v", err)), nil
	}

	result, _ := json.MarshalIndent(typeInfo, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *Server) handleSearchObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, ok := request.Params.Arguments["query"].(string)
	if !ok || query == "" {
		return newToolResultError("query is required"), nil
	}

	maxResults := 100
	if mr, ok := request.Params.Arguments["maxResults"].(float64); ok && mr > 0 {
		maxResults = int(mr)
	}

	results, err := s.adtClient.SearchObject(ctx, query, maxResults)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to search: %v", err)), nil
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// --- Development Tool Handlers ---

func (s *Server) handleSyntaxCheck(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	content, ok := request.Params.Arguments["content"].(string)
	if !ok || content == "" {
		return newToolResultError("content is required"), nil
	}

	results, err := s.adtClient.SyntaxCheck(ctx, objectURL, content)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Syntax check failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleActivate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	objectName, ok := request.Params.Arguments["object_name"].(string)
	if !ok || objectName == "" {
		return newToolResultError("object_name is required"), nil
	}

	result, err := s.adtClient.Activate(ctx, objectURL, objectName)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Activation failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleRunUnitTests(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	// Build flags from optional parameters
	flags := adt.DefaultUnitTestFlags()

	if includeDangerous, ok := request.Params.Arguments["include_dangerous"].(bool); ok && includeDangerous {
		flags.Dangerous = true
	}

	if includeLong, ok := request.Params.Arguments["include_long"].(bool); ok && includeLong {
		flags.Long = true
	}

	result, err := s.adtClient.RunUnitTests(ctx, objectURL, &flags)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Unit test run failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// --- CRUD Handlers ---

func (s *Server) handleLockObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	accessMode := "MODIFY"
	if am, ok := request.Params.Arguments["access_mode"].(string); ok && am != "" {
		accessMode = am
	}

	result, err := s.adtClient.LockObject(ctx, objectURL, accessMode)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to lock object: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleUnlockObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	lockHandle, ok := request.Params.Arguments["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return newToolResultError("lock_handle is required"), nil
	}

	err := s.adtClient.UnlockObject(ctx, objectURL, lockHandle)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to unlock object: %v", err)), nil
	}

	return mcp.NewToolResultText("Object unlocked successfully"), nil
}

func (s *Server) handleUpdateSource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	lockHandle, ok := request.Params.Arguments["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return newToolResultError("lock_handle is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	// Append /source/main to object URL if not already present
	sourceURL := objectURL
	if !strings.HasSuffix(sourceURL, "/source/main") {
		sourceURL = objectURL + "/source/main"
	}

	err := s.adtClient.UpdateSource(ctx, sourceURL, source, lockHandle, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to update source: %v", err)), nil
	}

	return mcp.NewToolResultText("Source updated successfully"), nil
}

func (s *Server) handleCreateObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.Params.Arguments["object_type"].(string)
	if !ok || objectType == "" {
		return newToolResultError("object_type is required"), nil
	}

	name, ok := request.Params.Arguments["name"].(string)
	if !ok || name == "" {
		return newToolResultError("name is required"), nil
	}

	description, ok := request.Params.Arguments["description"].(string)
	if !ok || description == "" {
		return newToolResultError("description is required"), nil
	}

	packageName, ok := request.Params.Arguments["package_name"].(string)
	if !ok || packageName == "" {
		return newToolResultError("package_name is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	parentName := ""
	if p, ok := request.Params.Arguments["parent_name"].(string); ok {
		parentName = p
	}

	opts := adt.CreateObjectOptions{
		ObjectType:  adt.CreatableObjectType(objectType),
		Name:        name,
		Description: description,
		PackageName: packageName,
		Transport:   transport,
		ParentName:  parentName,
	}

	err := s.adtClient.CreateObject(ctx, opts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to create object: %v", err)), nil
	}

	// Return the object URL for convenience
	objURL := adt.GetObjectURL(opts.ObjectType, opts.Name, opts.ParentName)
	result := map[string]string{
		"status":     "created",
		"object_url": objURL,
	}
	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleDeleteObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	lockHandle, ok := request.Params.Arguments["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return newToolResultError("lock_handle is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	err := s.adtClient.DeleteObject(ctx, objectURL, lockHandle, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to delete object: %v", err)), nil
	}

	return mcp.NewToolResultText("Object deleted successfully"), nil
}

// --- Class Include Handlers ---

func (s *Server) handleGetClassInclude(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.Params.Arguments["class_name"].(string)
	if !ok || className == "" {
		return newToolResultError("class_name is required"), nil
	}

	includeType, ok := request.Params.Arguments["include_type"].(string)
	if !ok || includeType == "" {
		return newToolResultError("include_type is required"), nil
	}

	source, err := s.adtClient.GetClassInclude(ctx, className, adt.ClassIncludeType(includeType))
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to get class include: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

func (s *Server) handleCreateTestInclude(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.Params.Arguments["class_name"].(string)
	if !ok || className == "" {
		return newToolResultError("class_name is required"), nil
	}

	lockHandle, ok := request.Params.Arguments["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return newToolResultError("lock_handle is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	err := s.adtClient.CreateTestInclude(ctx, className, lockHandle, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to create test include: %v", err)), nil
	}

	return mcp.NewToolResultText("Test include created successfully"), nil
}

func (s *Server) handleUpdateClassInclude(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.Params.Arguments["class_name"].(string)
	if !ok || className == "" {
		return newToolResultError("class_name is required"), nil
	}

	includeType, ok := request.Params.Arguments["include_type"].(string)
	if !ok || includeType == "" {
		return newToolResultError("include_type is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	lockHandle, ok := request.Params.Arguments["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return newToolResultError("lock_handle is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	err := s.adtClient.UpdateClassInclude(ctx, className, adt.ClassIncludeType(includeType), source, lockHandle, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to update class include: %v", err)), nil
	}

	return mcp.NewToolResultText("Class include updated successfully"), nil
}

// --- Workflow Handlers ---

func (s *Server) handleWriteProgram(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	programName, ok := request.Params.Arguments["program_name"].(string)
	if !ok || programName == "" {
		return newToolResultError("program_name is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	result, err := s.adtClient.WriteProgram(ctx, programName, source, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("WriteProgram failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleWriteClass(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.Params.Arguments["class_name"].(string)
	if !ok || className == "" {
		return newToolResultError("class_name is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	result, err := s.adtClient.WriteClass(ctx, className, source, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("WriteClass failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleCreateAndActivateProgram(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	programName, ok := request.Params.Arguments["program_name"].(string)
	if !ok || programName == "" {
		return newToolResultError("program_name is required"), nil
	}

	description, ok := request.Params.Arguments["description"].(string)
	if !ok || description == "" {
		return newToolResultError("description is required"), nil
	}

	packageName, ok := request.Params.Arguments["package_name"].(string)
	if !ok || packageName == "" {
		return newToolResultError("package_name is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	result, err := s.adtClient.CreateAndActivateProgram(ctx, programName, description, packageName, source, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("CreateAndActivateProgram failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleCreateClassWithTests(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, ok := request.Params.Arguments["class_name"].(string)
	if !ok || className == "" {
		return newToolResultError("class_name is required"), nil
	}

	description, ok := request.Params.Arguments["description"].(string)
	if !ok || description == "" {
		return newToolResultError("description is required"), nil
	}

	packageName, ok := request.Params.Arguments["package_name"].(string)
	if !ok || packageName == "" {
		return newToolResultError("package_name is required"), nil
	}

	classSource, ok := request.Params.Arguments["class_source"].(string)
	if !ok || classSource == "" {
		return newToolResultError("class_source is required"), nil
	}

	testSource, ok := request.Params.Arguments["test_source"].(string)
	if !ok || testSource == "" {
		return newToolResultError("test_source is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	result, err := s.adtClient.CreateClassWithTests(ctx, className, description, packageName, classSource, testSource, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("CreateClassWithTests failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// --- File-Based Deployment Handlers ---

// Note: CreateFromFile and UpdateFromFile handlers removed - use DeployFromFile instead

func (s *Server) handleDeployFromFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath, ok := request.Params.Arguments["file_path"].(string)
	if !ok || filePath == "" {
		return newToolResultError("file_path is required"), nil
	}

	packageName, ok := request.Params.Arguments["package_name"].(string)
	if !ok || packageName == "" {
		return newToolResultError("package_name is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	result, err := s.adtClient.DeployFromFile(ctx, filePath, packageName, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("DeployFromFile failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleSaveToFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objTypeStr, ok := request.Params.Arguments["objType"].(string)
	if !ok || objTypeStr == "" {
		return newToolResultError("objType is required (e.g., CLAS/OC, PROG/P, INTF/OI, FUGR/F, FUGR/FF)"), nil
	}

	objectName, ok := request.Params.Arguments["objectName"].(string)
	if !ok || objectName == "" {
		return newToolResultError("objectName is required"), nil
	}

	outputPath := ""
	if p, ok := request.Params.Arguments["outputPath"].(string); ok {
		outputPath = p
	}

	// Parse object type
	objType := adt.CreatableObjectType(objTypeStr)

	result, err := s.adtClient.SaveToFile(ctx, objType, objectName, outputPath)
	if err != nil {
		return newToolResultError(fmt.Sprintf("SaveToFile failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleRenameObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objTypeStr, ok := request.Params.Arguments["objType"].(string)
	if !ok || objTypeStr == "" {
		return newToolResultError("objType is required (e.g., CLAS/OC, PROG/P, INTF/OI, FUGR/F)"), nil
	}

	oldName, ok := request.Params.Arguments["oldName"].(string)
	if !ok || oldName == "" {
		return newToolResultError("oldName is required"), nil
	}

	newName, ok := request.Params.Arguments["newName"].(string)
	if !ok || newName == "" {
		return newToolResultError("newName is required"), nil
	}

	packageName, ok := request.Params.Arguments["packageName"].(string)
	if !ok || packageName == "" {
		return newToolResultError("packageName is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	// Parse object type
	objType := adt.CreatableObjectType(objTypeStr)

	result, err := s.adtClient.RenameObject(ctx, objType, oldName, newName, packageName, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("RenameObject failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleEditSource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	oldString, ok := request.Params.Arguments["old_string"].(string)
	if !ok || oldString == "" {
		return newToolResultError("old_string is required"), nil
	}

	newString, ok := request.Params.Arguments["new_string"].(string)
	if !ok {
		return newToolResultError("new_string is required"), nil
	}

	replaceAll := false
	if r, ok := request.Params.Arguments["replace_all"].(bool); ok {
		replaceAll = r
	}

	syntaxCheck := true
	if sc, ok := request.Params.Arguments["syntax_check"].(bool); ok {
		syntaxCheck = sc
	}

	caseInsensitive := false
	if ci, ok := request.Params.Arguments["case_insensitive"].(bool); ok {
		caseInsensitive = ci
	}

	result, err := s.adtClient.EditSource(ctx, objectURL, oldString, newString, replaceAll, syntaxCheck, caseInsensitive)
	if err != nil {
		return newToolResultError(fmt.Sprintf("EditSource failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// --- Grep/Search Handlers ---

func (s *Server) handleGrepObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	pattern, ok := request.Params.Arguments["pattern"].(string)
	if !ok || pattern == "" {
		return newToolResultError("pattern is required"), nil
	}

	caseInsensitive := false
	if ci, ok := request.Params.Arguments["case_insensitive"].(bool); ok {
		caseInsensitive = ci
	}

	contextLines := 0
	if cl, ok := request.Params.Arguments["context_lines"].(float64); ok {
		contextLines = int(cl)
	}

	result, err := s.adtClient.GrepObject(ctx, objectURL, pattern, caseInsensitive, contextLines)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GrepObject failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleGrepPackage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	packageName, ok := request.Params.Arguments["package_name"].(string)
	if !ok || packageName == "" {
		return newToolResultError("package_name is required"), nil
	}

	pattern, ok := request.Params.Arguments["pattern"].(string)
	if !ok || pattern == "" {
		return newToolResultError("pattern is required"), nil
	}

	caseInsensitive := false
	if ci, ok := request.Params.Arguments["case_insensitive"].(bool); ok {
		caseInsensitive = ci
	}

	// Parse object_types (comma-separated string to slice)
	var objectTypes []string
	if ot, ok := request.Params.Arguments["object_types"].(string); ok && ot != "" {
		objectTypes = strings.Split(ot, ",")
		// Trim whitespace from each type
		for i := range objectTypes {
			objectTypes[i] = strings.TrimSpace(objectTypes[i])
		}
	}

	maxResults := 100 // default
	if mr, ok := request.Params.Arguments["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	result, err := s.adtClient.GrepPackage(ctx, packageName, pattern, caseInsensitive, objectTypes, maxResults)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GrepPackage failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// --- Code Intelligence Handlers ---

func (s *Server) handleFindDefinition(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceURL, ok := request.Params.Arguments["source_url"].(string)
	if !ok || sourceURL == "" {
		return newToolResultError("source_url is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	lineF, ok := request.Params.Arguments["line"].(float64)
	if !ok {
		return newToolResultError("line is required"), nil
	}
	line := int(lineF)

	startColF, ok := request.Params.Arguments["start_column"].(float64)
	if !ok {
		return newToolResultError("start_column is required"), nil
	}
	startCol := int(startColF)

	endColF, ok := request.Params.Arguments["end_column"].(float64)
	if !ok {
		return newToolResultError("end_column is required"), nil
	}
	endCol := int(endColF)

	implementation := false
	if impl, ok := request.Params.Arguments["implementation"].(bool); ok {
		implementation = impl
	}

	mainProgram := ""
	if mp, ok := request.Params.Arguments["main_program"].(string); ok {
		mainProgram = mp
	}

	result, err := s.adtClient.FindDefinition(ctx, sourceURL, source, line, startCol, endCol, implementation, mainProgram)
	if err != nil {
		return newToolResultError(fmt.Sprintf("FindDefinition failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleFindReferences(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	line := 0
	column := 0
	if lineF, ok := request.Params.Arguments["line"].(float64); ok {
		line = int(lineF)
	}
	if colF, ok := request.Params.Arguments["column"].(float64); ok {
		column = int(colF)
	}

	results, err := s.adtClient.FindReferences(ctx, objectURL, line, column)
	if err != nil {
		return newToolResultError(fmt.Sprintf("FindReferences failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleCodeCompletion(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceURL, ok := request.Params.Arguments["source_url"].(string)
	if !ok || sourceURL == "" {
		return newToolResultError("source_url is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	lineF, ok := request.Params.Arguments["line"].(float64)
	if !ok {
		return newToolResultError("line is required"), nil
	}
	line := int(lineF)

	colF, ok := request.Params.Arguments["column"].(float64)
	if !ok {
		return newToolResultError("column is required"), nil
	}
	column := int(colF)

	proposals, err := s.adtClient.CodeCompletion(ctx, sourceURL, source, line, column)
	if err != nil {
		return newToolResultError(fmt.Sprintf("CodeCompletion failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(proposals, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handlePrettyPrint(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	formatted, err := s.adtClient.PrettyPrint(ctx, source)
	if err != nil {
		return newToolResultError(fmt.Sprintf("PrettyPrint failed: %v", err)), nil
	}

	return mcp.NewToolResultText(formatted), nil
}

func (s *Server) handleGetPrettyPrinterSettings(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	settings, err := s.adtClient.GetPrettyPrinterSettings(ctx)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GetPrettyPrinterSettings failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(settings, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleSetPrettyPrinterSettings(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	indentation, ok := request.Params.Arguments["indentation"].(bool)
	if !ok {
		return newToolResultError("indentation is required"), nil
	}

	style, ok := request.Params.Arguments["style"].(string)
	if !ok || style == "" {
		return newToolResultError("style is required"), nil
	}

	settings := &adt.PrettyPrinterSettings{
		Indentation: indentation,
		Style:       adt.PrettyPrinterStyle(style),
	}

	err := s.adtClient.SetPrettyPrinterSettings(ctx, settings)
	if err != nil {
		return newToolResultError(fmt.Sprintf("SetPrettyPrinterSettings failed: %v", err)), nil
	}

	return mcp.NewToolResultText("Pretty printer settings updated successfully"), nil
}

func (s *Server) handleGetTypeHierarchy(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sourceURL, ok := request.Params.Arguments["source_url"].(string)
	if !ok || sourceURL == "" {
		return newToolResultError("source_url is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	lineF, ok := request.Params.Arguments["line"].(float64)
	if !ok {
		return newToolResultError("line is required"), nil
	}
	line := int(lineF)

	colF, ok := request.Params.Arguments["column"].(float64)
	if !ok {
		return newToolResultError("column is required"), nil
	}
	column := int(colF)

	superTypes := false
	if st, ok := request.Params.Arguments["super_types"].(bool); ok {
		superTypes = st
	}

	hierarchy, err := s.adtClient.GetTypeHierarchy(ctx, sourceURL, source, line, column, superTypes)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GetTypeHierarchy failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(hierarchy, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// registerGetSource registers the unified GetSource tool
func (s *Server) registerGetSource() {
	s.mcpServer.AddTool(mcp.NewTool("GetSource",
		mcp.WithDescription("Unified tool for reading ABAP source code across different object types. Replaces GetProgram, GetClass, GetInterface, GetFunction, GetInclude, GetFunctionGroup, GetClassInclude."),
		mcp.WithString("object_type",
			mcp.Required(),
			mcp.Description("Object type: PROG (program), CLAS (class), INTF (interface), FUNC (function module), FUGR (function group), INCL (include)"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Object name (e.g., program name, class name, function module name)"),
		),
		mcp.WithString("parent",
			mcp.Description("Function group name (required only for FUNC type)"),
		),
		mcp.WithString("include",
			mcp.Description("Class include type for CLAS: definitions, implementations, macros, testclasses (optional)"),
		),
	), s.handleGetSource)
}

// registerWriteSource registers the unified WriteSource tool
func (s *Server) registerWriteSource() {
	s.mcpServer.AddTool(mcp.NewTool("WriteSource",
		mcp.WithDescription("Unified tool for writing ABAP source code with automatic create/update detection. Replaces WriteProgram, WriteClass, CreateAndActivateProgram, CreateClassWithTests."),
		mcp.WithString("object_type",
			mcp.Required(),
			mcp.Description("Object type: PROG (program), CLAS (class), INTF (interface)"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Object name"),
		),
		mcp.WithString("source",
			mcp.Required(),
			mcp.Description("ABAP source code"),
		),
		mcp.WithString("mode",
			mcp.Description("Operation mode: upsert (default, auto-detect), create (new only), update (existing only)"),
		),
		mcp.WithString("description",
			mcp.Description("Object description (required for create mode)"),
		),
		mcp.WithString("package",
			mcp.Description("Package name (required for create mode)"),
		),
		mcp.WithString("test_source",
			mcp.Description("Test source code for CLAS (auto-creates test include and runs tests)"),
		),
		mcp.WithString("transport",
			mcp.Description("Transport request number"),
		),
	), s.handleWriteSource)
}

// handleGetSource handles the unified GetSource tool call
func (s *Server) handleGetSource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.Params.Arguments["object_type"].(string)
	if !ok || objectType == "" {
		return newToolResultError("object_type is required"), nil
	}

	name, ok := request.Params.Arguments["name"].(string)
	if !ok || name == "" {
		return newToolResultError("name is required"), nil
	}

	parent, _ := request.Params.Arguments["parent"].(string)
	include, _ := request.Params.Arguments["include"].(string)

	opts := &adt.GetSourceOptions{
		Parent:  parent,
		Include: include,
	}

	source, err := s.adtClient.GetSource(ctx, objectType, name, opts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GetSource failed: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

// handleWriteSource handles the unified WriteSource tool call
func (s *Server) handleWriteSource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.Params.Arguments["object_type"].(string)
	if !ok || objectType == "" {
		return newToolResultError("object_type is required"), nil
	}

	name, ok := request.Params.Arguments["name"].(string)
	if !ok || name == "" {
		return newToolResultError("name is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	mode, _ := request.Params.Arguments["mode"].(string)
	description, _ := request.Params.Arguments["description"].(string)
	packageName, _ := request.Params.Arguments["package"].(string)
	testSource, _ := request.Params.Arguments["test_source"].(string)
	transport, _ := request.Params.Arguments["transport"].(string)

	opts := &adt.WriteSourceOptions{
		Description: description,
		Package:     packageName,
		TestSource:  testSource,
		Transport:   transport,
	}

	if mode != "" {
		opts.Mode = adt.WriteSourceMode(mode)
	}

	result, err := s.adtClient.WriteSource(ctx, objectType, name, source, opts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("WriteSource failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// registerGrepObjects registers the unified GrepObjects tool
func (s *Server) registerGrepObjects() {
	s.mcpServer.AddTool(mcp.NewTool("GrepObjects",
		mcp.WithDescription("Unified tool for searching regex patterns in single or multiple ABAP objects. Replaces GrepObject."),
		mcp.WithArray("object_urls",
			mcp.Required(),
			mcp.Description("Array of ADT object URLs to search (e.g., [\"/sap/bc/adt/programs/programs/ZTEST\"])"),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
		mcp.WithString("pattern",
			mcp.Required(),
			mcp.Description("Regular expression pattern (Go regexp syntax)"),
		),
		mcp.WithBoolean("case_insensitive",
			mcp.Description("If true, perform case-insensitive matching (default: false)"),
		),
		mcp.WithNumber("context_lines",
			mcp.Description("Number of context lines before/after each match (default: 0)"),
		),
	), s.handleGrepObjects)
}

// registerGrepPackages registers the unified GrepPackages tool
func (s *Server) registerGrepPackages() {
	s.mcpServer.AddTool(mcp.NewTool("GrepPackages",
		mcp.WithDescription("Unified tool for searching regex patterns across single or multiple packages with optional recursive subpackage search. Replaces GrepPackage."),
		mcp.WithArray("packages",
			mcp.Required(),
			mcp.Description("Array of package names to search (e.g., [\"$TMP\"], [\"Z\"] for namespace search)"),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
		mcp.WithBoolean("include_subpackages",
			mcp.Description("If true, recursively search all subpackages (default: false). Enables namespace-wide searches."),
		),
		mcp.WithString("pattern",
			mcp.Required(),
			mcp.Description("Regular expression pattern (Go regexp syntax)"),
		),
		mcp.WithBoolean("case_insensitive",
			mcp.Description("If true, perform case-insensitive matching (default: false)"),
		),
		mcp.WithArray("object_types",
			mcp.Description("Filter by object types (e.g., [\"CLAS/OC\", \"PROG/P\"]). Empty = search all types."),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of matching objects to return (0 = unlimited, default: 0)"),
		),
	), s.handleGrepPackages)
}

// registerImportFromFile registers the ImportFromFile tool (alias for DeployFromFile)
func (s *Server) registerImportFromFile() {
	s.mcpServer.AddTool(mcp.NewTool("ImportFromFile",
		mcp.WithDescription("Import ABAP object from local file into SAP system. Auto-detects object type, creates or updates, activates. Supports: programs, classes, interfaces, function groups/modules. Renamed from DeployFromFile for clarity."),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Absolute path to ABAP source file (.prog.abap, .clas.abap, .intf.abap, .fugr.abap, .func.abap)"),
		),
		mcp.WithString("package_name",
			mcp.Description("Target package name (required for new objects)"),
		),
		mcp.WithString("transport",
			mcp.Description("Transport request number"),
		),
	), s.handleDeployFromFile) // Reuse existing handler
}

// registerExportToFile registers the ExportToFile tool (alias for SaveToFile)
func (s *Server) registerExportToFile() {
	s.mcpServer.AddTool(mcp.NewTool("ExportToFile",
		mcp.WithDescription("Export ABAP object from SAP system to local file. Saves source code with appropriate file extension. Supports: programs, classes, interfaces, function groups/modules. Renamed from SaveToFile for clarity."),
		mcp.WithString("object_type",
			mcp.Required(),
			mcp.Description("Object type: PROG, CLAS, INTF, FUGR, FUNC"),
		),
		mcp.WithString("object_name",
			mcp.Required(),
			mcp.Description("Object name"),
		),
		mcp.WithString("output_dir",
			mcp.Required(),
			mcp.Description("Output directory path (must exist)"),
		),
		mcp.WithString("parent",
			mcp.Description("Function group name (required for FUNC type)"),
		),
	), s.handleSaveToFile) // Reuse existing handler
}

// handleGrepObjects handles the unified GrepObjects tool call
func (s *Server) handleGrepObjects(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURLsRaw, ok := request.Params.Arguments["object_urls"].([]interface{})
	if !ok || len(objectURLsRaw) == 0 {
		return newToolResultError("object_urls array is required"), nil
	}

	// Convert []interface{} to []string
	objectURLs := make([]string, len(objectURLsRaw))
	for i, v := range objectURLsRaw {
		if s, ok := v.(string); ok {
			objectURLs[i] = s
		} else {
			return newToolResultError(fmt.Sprintf("object_urls[%d] must be a string", i)), nil
		}
	}

	pattern, ok := request.Params.Arguments["pattern"].(string)
	if !ok || pattern == "" {
		return newToolResultError("pattern is required"), nil
	}

	caseInsensitive := false
	if ci, ok := request.Params.Arguments["case_insensitive"].(bool); ok {
		caseInsensitive = ci
	}

	contextLines := 0
	if cl, ok := request.Params.Arguments["context_lines"].(float64); ok {
		contextLines = int(cl)
	}

	result, err := s.adtClient.GrepObjects(ctx, objectURLs, pattern, caseInsensitive, contextLines)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GrepObjects failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// handleGrepPackages handles the unified GrepPackages tool call
func (s *Server) handleGrepPackages(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	packagesRaw, ok := request.Params.Arguments["packages"].([]interface{})
	if !ok || len(packagesRaw) == 0 {
		return newToolResultError("packages array is required"), nil
	}

	// Convert []interface{} to []string
	packages := make([]string, len(packagesRaw))
	for i, v := range packagesRaw {
		if s, ok := v.(string); ok {
			packages[i] = s
		} else {
			return newToolResultError(fmt.Sprintf("packages[%d] must be a string", i)), nil
		}
	}

	includeSubpackages := false
	if is, ok := request.Params.Arguments["include_subpackages"].(bool); ok {
		includeSubpackages = is
	}

	pattern, ok := request.Params.Arguments["pattern"].(string)
	if !ok || pattern == "" {
		return newToolResultError("pattern is required"), nil
	}

	caseInsensitive := false
	if ci, ok := request.Params.Arguments["case_insensitive"].(bool); ok {
		caseInsensitive = ci
	}

	var objectTypes []string
	if ot, ok := request.Params.Arguments["object_types"].([]interface{}); ok {
		objectTypes = make([]string, len(ot))
		for i, v := range ot {
			if s, ok := v.(string); ok {
				objectTypes[i] = s
			}
		}
	}

	maxResults := 0
	if mr, ok := request.Params.Arguments["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	result, err := s.adtClient.GrepPackages(ctx, packages, includeSubpackages, pattern, caseInsensitive, objectTypes, maxResults)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GrepPackages failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
