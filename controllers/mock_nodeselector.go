// Code generated by MockGen. DO NOT EDIT.
// Source: nodeselector.go

// Package controllers is a generated GoMock package.
package controllers

import (
	context "context"
	reflect "reflect"

	v1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
	gomock "github.com/golang/mock/gomock"
)

// MockNodeSelectorValidator is a mock of NodeSelectorValidator interface.
type MockNodeSelectorValidator struct {
	ctrl     *gomock.Controller
	recorder *MockNodeSelectorValidatorMockRecorder
}

// MockNodeSelectorValidatorMockRecorder is the mock recorder for MockNodeSelectorValidator.
type MockNodeSelectorValidatorMockRecorder struct {
	mock *MockNodeSelectorValidator
}

// NewMockNodeSelectorValidator creates a new mock instance.
func NewMockNodeSelectorValidator(ctrl *gomock.Controller) *MockNodeSelectorValidator {
	mock := &MockNodeSelectorValidator{ctrl: ctrl}
	mock.recorder = &MockNodeSelectorValidatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNodeSelectorValidator) EXPECT() *MockNodeSelectorValidatorMockRecorder {
	return m.recorder
}

// CheckDeviceConfigForConflictingNodeSelector mocks base method.
func (m *MockNodeSelectorValidator) CheckDeviceConfigForConflictingNodeSelector(ctx context.Context, cr *v1alpha1.DeviceConfig) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckDeviceConfigForConflictingNodeSelector", ctx, cr)
	ret0, _ := ret[0].(error)
	return ret0
}

// CheckDeviceConfigForConflictingNodeSelector indicates an expected call of CheckDeviceConfigForConflictingNodeSelector.
func (mr *MockNodeSelectorValidatorMockRecorder) CheckDeviceConfigForConflictingNodeSelector(ctx, cr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckDeviceConfigForConflictingNodeSelector", reflect.TypeOf((*MockNodeSelectorValidator)(nil).CheckDeviceConfigForConflictingNodeSelector), ctx, cr)
}
