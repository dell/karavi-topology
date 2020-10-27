// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/dell/karavi-topology/internal/entrypoint (interfaces: ServiceRunner)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockServiceRunner is a mock of ServiceRunner interface
type MockServiceRunner struct {
	ctrl     *gomock.Controller
	recorder *MockServiceRunnerMockRecorder
}

// MockServiceRunnerMockRecorder is the mock recorder for MockServiceRunner
type MockServiceRunnerMockRecorder struct {
	mock *MockServiceRunner
}

// NewMockServiceRunner creates a new mock instance
func NewMockServiceRunner(ctrl *gomock.Controller) *MockServiceRunner {
	mock := &MockServiceRunner{ctrl: ctrl}
	mock.recorder = &MockServiceRunnerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockServiceRunner) EXPECT() *MockServiceRunnerMockRecorder {
	return m.recorder
}

// Run mocks base method
func (m *MockServiceRunner) Run() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run")
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run
func (mr *MockServiceRunnerMockRecorder) Run() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockServiceRunner)(nil).Run))
}