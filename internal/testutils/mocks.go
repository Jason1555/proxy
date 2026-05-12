package testutils

type MockLogger struct{}

func (l *MockLogger) Debugf(format string, args ...any) {}
func (l *MockLogger) Infof(format string, args ...any)  {}
func (l *MockLogger) Warnf(format string, args ...any)  {}
func (l *MockLogger) Errorf(format string, args ...any) {}
func (l *MockLogger) Fatalf(format string, args ...any) {}
