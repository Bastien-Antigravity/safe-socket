package interfaces

// =============================================================================
// ESSENTIAL PROCESS:
// Defines the universal logging interface contract for safe-socket components,
// decoupling transport, facade, and protocol logging from concrete logger backends.
//
// DATA FLOW:
// 1. Input: Diagnostic and error string messages at varying severity levels.
// 2. Logic: Delegated to the injected ecosystem logger implementation.
// 3. Output: Structured log records dispatched to console, file, or remote sinks.
//
// KEY PARAMETERS:
// - Logger: Interface defining Debug, Info, Warning, Error, Critical, and Close methods.
// =============================================================================

// -----------------------------------------------------------------------------
// Logger is the main interface for logging
type Logger interface {
	// -------------------------------------------------------------------------
	// Debug logs a message at Debug level.
	Debug(msg string)

	// -------------------------------------------------------------------------
	// Info logs a message at Info level.
	Info(msg string)

	// -------------------------------------------------------------------------
	// Warning logs a message at Warning level.
	Warning(msg string)

	// -------------------------------------------------------------------------
	// Error logs a message at Error level.
	Error(msg string)

	// -------------------------------------------------------------------------
	// Critical logs a message at Critical level.
	Critical(msg string)

	// -------------------------------------------------------------------------
	// Close flushes any buffered logs and closes the handler.
	Close()
}
