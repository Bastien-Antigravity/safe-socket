	if s.listener != nil {
		err := s.listener.Close()
		s.listener = nil
		return err
	}
	return nil
}

// -----------------------------------------------------------------------------

// Bind logger to safe-socket
func (s *SocketServer) SetLogger(logger *interfaces.Logger) {
	s.Logger = logger
}

// -----------------------------------------------------------------------------
// Client Methods (Not Supported for Server)
// -----------------------------------------------------------------------------

func (s *SocketServer) Open() error {
	return errors.New("method Open not supported for Server socket")
}

func (s *SocketServer) Send(data []byte) error {
	return errors.New("method Send not supported for Server socket")
}

func (s *SocketServer) Write(data []byte) (int, error) {
	return 0, errors.New("method Write not supported for Server socket")
}

func (s *SocketServer) Receive() ([]byte, error) {
	return nil, errors.New("method Receive not supported for Server socket")
}

func (s *SocketServer) Read(p []byte) (int, error) {
	return 0, errors.New("method Read not supported for Server socket")
}

func (s *SocketServer) SetDeadline(t time.Time) error {
	return errors.New("method SetDeadline not supported for Server listener (use Config.Deadline for accepted conns)")
}

func (s *SocketServer) SetReadDeadline(t time.Time) error {
	return errors.New("method SetReadDeadline not supported for Server listener")
}

func (s *SocketServer) SetWriteDeadline(t time.Time) error {
	return errors.New("method SetWriteDeadline not supported for Server listener")
}

// SetIdleTimeout updates the internal idle timeout for newly accepted connections.
func (s *SocketServer) SetIdleTimeout(d time.Duration) error {
	s.Config.Deadline = d
	return nil
}
