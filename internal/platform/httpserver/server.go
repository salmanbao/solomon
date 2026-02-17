package httpserver

// Server represents the HTTP transport adapter.
// It should only map transport DTOs to application commands/queries.
type Server struct {}

func New() *Server {
	return &Server{}
}

func (s *Server) Start() error {
	// TODO: start router, middleware, and handlers.
	return nil
}
