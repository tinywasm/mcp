package mcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"testing"
)

// Server encapsulates an MCP server and manages resources like pipes and context.
type Server struct {
	name string

	tools             []ServerTool
	prompts           []ServerPrompt
	resources         []ServerResource
	resourceTemplates []ServerResourceTemplate
	clientInfo        Implementation

	cancel func()

	serverReader *io.PipeReader
	serverWriter *io.PipeWriter
	clientReader *io.PipeReader
	clientWriter *io.PipeWriter

	logBuffer bytes.Buffer

	transport Interface
	client    *Client

	wg sync.WaitGroup
}

// NewServer starts a new MCP server with the provided tools and returns the server instance.
func NewServer(t *testing.T, tools ...ServerTool) (*Server, error) {
	server := NewUnstartedServer(t)
	server.AddTools(tools...)

	// TODO: use t.Context() once go.mod is upgraded to go 1.24+
	if err := server.Start(context.TODO()); err != nil {
		return nil, err
	}

	return server, nil
}

// NewUnstartedServer creates a new MCP server instance with the given name, but does not start the 
// Useful for tests where you need to add tools before starting the 
func NewUnstartedServer(t *testing.T) *Server {
	server := &Server{
		name: t.Name(),
	}

	// Set up pipes for client-server communication
	server.serverReader, server.clientWriter = io.Pipe()
	server.clientReader, server.serverWriter = io.Pipe()

	// Return the configured server
	return server
}

// AddTools adds multiple tools to an unstarted 
func (s *Server) AddTools(tools ...ServerTool) {
	s.tools = append(s.tools, tools...)
}

// AddTool adds a tool to an unstarted 
func (s *Server) AddTool(tool Tool, handler ToolHandlerFunc) {
	s.tools = append(s.tools, ServerTool{
		Tool:    tool,
		Handler: handler,
	})
}

// AddPrompt adds a prompt to an unstarted 
func (s *Server) AddPrompt(prompt Prompt, handler PromptHandlerFunc) {
	s.prompts = append(s.prompts, ServerPrompt{
		Prompt:  prompt,
		Handler: handler,
	})
}

// AddPrompts adds multiple prompts to an unstarted 
func (s *Server) AddPrompts(prompts ...ServerPrompt) {
	s.prompts = append(s.prompts, prompts...)
}

// AddResource adds a resource to an unstarted 
func (s *Server) AddResource(resource Resource, handler ResourceHandlerFunc) {
	s.resources = append(s.resources, ServerResource{
		Resource: resource,
		Handler:  handler,
	})
}

// AddResources adds multiple resources to an unstarted 
func (s *Server) AddResources(resources ...ServerResource) {
	s.resources = append(s.resources, resources...)
}

// AddResourceTemplate adds a resource template to an unstarted 
func (s *Server) AddResourceTemplate(template ResourceTemplate, handler ResourceTemplateHandlerFunc) {
	s.resourceTemplates = append(s.resourceTemplates, ServerResourceTemplate{
		Template: template,
		Handler:  handler,
	})
}

// AddResourceTemplates adds multiple resource templates to an unstarted 
func (s *Server) AddResourceTemplates(templates ...ServerResourceTemplate) {
	s.resourceTemplates = append(s.resourceTemplates, templates...)
}

// SetClientInfo sets the client info for the test 
func (s *Server) SetClientInfo(info Implementation) {
	s.clientInfo = info
}

// Start starts the server in a goroutine. Make sure to defer Close() after Start().
// When using NewServer(), the returned server is already started.
func (s *Server) Start(ctx context.Context) error {
	s.wg.Add(1)

	ctx, s.cancel = context.WithCancel(ctx)

	// Start the MCP server in a goroutine
	go func() {
		defer s.wg.Done()

		mcpServer := NewMCPServer(s.name, "1.0.0")

		mcpServer.AddTools(s.tools...)
		mcpServer.AddPrompts(s.prompts...)
		mcpServer.AddResources(s.resources...)
		mcpServer.AddResourceTemplates(s.resourceTemplates...)

		/*
		logger := log.New(&s.logBuffer, "", 0)

		stdioServer := NewStdioServer(mcpServer)
		stdioServer.SetErrorLogger(logger)

		if err := stdioServer.Listen(ctx, s.serverReader, s.serverWriter); err != nil {
			logger.Println("StdioServer.Listen failed:", err)
		}
		*/
	}()

	s.transport = NewIO(s.clientReader, s.clientWriter, io.NopCloser(&s.logBuffer))
	if err := s.transport.Start(ctx); err != nil {
		return fmt.Errorf("Start(): %w", err)
	}

	s.client = NewClient(s.transport)

	var initReq InitializeRequest
	initReq.Params.ProtocolVersion = LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = s.clientInfo
	if _, err := s.client.Initialize(ctx, initReq); err != nil {
		return fmt.Errorf("Initialize(): %w", err)
	}

	return nil
}

// Close stops the server and cleans up resources like temporary directories.
func (s *Server) Close() {
	if s.transport != nil {
		s.transport.Close()
		s.transport = nil
		s.client = nil
	}

	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}

	// Wait for server goroutine to finish
	s.wg.Wait()

	s.serverWriter.Close()
	s.serverReader.Close()
	s.serverReader, s.serverWriter = nil, nil

	s.clientWriter.Close()
	s.clientReader.Close()
	s.clientReader, s.clientWriter = nil, nil
}

// Client returns an MCP client connected to the 
// The client is already initialized, i.e. you do _not_ need to call Client.Initialize().
func (s *Server) Client() *Client {
	return s.client
}
