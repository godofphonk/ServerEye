package agent

// Common test helpers and mocks

// MockUpdateFunc returns a no-op update function for testing
// This prevents tests from executing real system commands (wget, mv, chmod)
func MockUpdateFunc() func(string) error {
	return func(version string) error {
		return nil
	}
}
