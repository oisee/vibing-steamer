// Package mcp provides the MCP server implementation for ABAP ADT tools.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/vibingsteamer/mcp-abap-adt-go/pkg/adt"
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

	// Register all tools
	s.registerTools()

	return s
}

// ServeStdio starts the MCP server on stdin/stdout.
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}

// registerTools registers all ADT tools with the MCP server.
func (s *Server) registerTools() {
	// GetProgram
	s.mcpServer.AddTool(mcp.NewTool("GetProgram",
		mcp.WithDescription("Retrieve ABAP program source code"),
		mcp.WithString("program_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP program"),
		),
	), s.handleGetProgram)

	// GetClass
	s.mcpServer.AddTool(mcp.NewTool("GetClass",
		mcp.WithDescription("Retrieve ABAP class source code"),
		mcp.WithString("class_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP class"),
		),
	), s.handleGetClass)

	// GetInterface
	s.mcpServer.AddTool(mcp.NewTool("GetInterface",
		mcp.WithDescription("Retrieve ABAP interface source code"),
		mcp.WithString("interface_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP interface"),
		),
	), s.handleGetInterface)

	// GetFunction
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

	// GetFunctionGroup
	s.mcpServer.AddTool(mcp.NewTool("GetFunctionGroup",
		mcp.WithDescription("Retrieve ABAP Function Group source code"),
		mcp.WithString("function_group",
			mcp.Required(),
			mcp.Description("Name of the function group"),
		),
	), s.handleGetFunctionGroup)

	// GetInclude
	s.mcpServer.AddTool(mcp.NewTool("GetInclude",
		mcp.WithDescription("Retrieve ABAP Include Source Code"),
		mcp.WithString("include_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP Include"),
		),
	), s.handleGetInclude)

	// GetTable
	s.mcpServer.AddTool(mcp.NewTool("GetTable",
		mcp.WithDescription("Retrieve ABAP table structure"),
		mcp.WithString("table_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP table"),
		),
	), s.handleGetTable)

	// GetTableContents
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

	// RunQuery
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

	// GetStructure
	s.mcpServer.AddTool(mcp.NewTool("GetStructure",
		mcp.WithDescription("Retrieve ABAP Structure"),
		mcp.WithString("structure_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP Structure"),
		),
	), s.handleGetStructure)

	// GetPackage
	s.mcpServer.AddTool(mcp.NewTool("GetPackage",
		mcp.WithDescription("Retrieve ABAP package details"),
		mcp.WithString("package_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP package"),
		),
	), s.handleGetPackage)

	// GetTransaction
	s.mcpServer.AddTool(mcp.NewTool("GetTransaction",
		mcp.WithDescription("Retrieve ABAP transaction details"),
		mcp.WithString("transaction_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP transaction"),
		),
	), s.handleGetTransaction)

	// GetTypeInfo
	s.mcpServer.AddTool(mcp.NewTool("GetTypeInfo",
		mcp.WithDescription("Retrieve ABAP type information"),
		mcp.WithString("type_name",
			mcp.Required(),
			mcp.Description("Name of the ABAP type"),
		),
	), s.handleGetTypeInfo)

	// SearchObject
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

	// --- Development Tools ---

	// SyntaxCheck
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

	// Activate
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

	// RunUnitTests
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

	// --- CRUD Operations ---

	// LockObject
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

	// UnlockObject
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

	// UpdateSource
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

	// CreateObject
	s.mcpServer.AddTool(mcp.NewTool("CreateObject",
		mcp.WithDescription("Create a new ABAP object"),
		mcp.WithString("object_type",
			mcp.Required(),
			mcp.Description("Object type: PROG/P (program), CLAS/OC (class), INTF/OI (interface), PROG/I (include), FUGR/F (function group), FUGR/FF (function module)"),
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

	// DeleteObject
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
