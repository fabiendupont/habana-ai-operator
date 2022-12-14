// Code generated by MockGen. DO NOT EDIT.
// Source: labeler.go

// Package labeler is a generated GoMock package.
package labeler

import (
	context "context"
	reflect "reflect"

	v1alpha1 "github.com/HabanaAI/habana-ai-operator/api/v1alpha1"
	gomock "github.com/golang/mock/gomock"
	v1 "k8s.io/api/apps/v1"
)

// MockReconciler is a mock of Reconciler interface.
type MockReconciler struct {
	ctrl     *gomock.Controller
	recorder *MockReconcilerMockRecorder
}

// MockReconcilerMockRecorder is the mock recorder for MockReconciler.
type MockReconcilerMockRecorder struct {
	mock *MockReconciler
}

// NewMockReconciler creates a new mock instance.
func NewMockReconciler(ctrl *gomock.Controller) *MockReconciler {
	mock := &MockReconciler{ctrl: ctrl}
	mock.recorder = &MockReconcilerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockReconciler) EXPECT() *MockReconcilerMockRecorder {
	return m.recorder
}

// DeleteNodeLabeler mocks base method.
func (m *MockReconciler) DeleteNodeLabeler(ctx context.Context, dc *v1alpha1.DeviceConfig) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNodeLabeler", ctx, dc)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNodeLabeler indicates an expected call of DeleteNodeLabeler.
func (mr *MockReconcilerMockRecorder) DeleteNodeLabeler(ctx, dc interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNodeLabeler", reflect.TypeOf((*MockReconciler)(nil).DeleteNodeLabeler), ctx, dc)
}

// DeleteNodeLabelerDaemonSet mocks base method.
func (m *MockReconciler) DeleteNodeLabelerDaemonSet(ctx context.Context, dc *v1alpha1.DeviceConfig) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNodeLabelerDaemonSet", ctx, dc)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNodeLabelerDaemonSet indicates an expected call of DeleteNodeLabelerDaemonSet.
func (mr *MockReconcilerMockRecorder) DeleteNodeLabelerDaemonSet(ctx, dc interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNodeLabelerDaemonSet", reflect.TypeOf((*MockReconciler)(nil).DeleteNodeLabelerDaemonSet), ctx, dc)
}

// ReconcileNodeLabeler mocks base method.
func (m *MockReconciler) ReconcileNodeLabeler(ctx context.Context, dc *v1alpha1.DeviceConfig) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReconcileNodeLabeler", ctx, dc)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReconcileNodeLabeler indicates an expected call of ReconcileNodeLabeler.
func (mr *MockReconcilerMockRecorder) ReconcileNodeLabeler(ctx, dc interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReconcileNodeLabeler", reflect.TypeOf((*MockReconciler)(nil).ReconcileNodeLabeler), ctx, dc)
}

// ReconcileNodeLabelerDaemonSet mocks base method.
func (m *MockReconciler) ReconcileNodeLabelerDaemonSet(ctx context.Context, dc *v1alpha1.DeviceConfig) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReconcileNodeLabelerDaemonSet", ctx, dc)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReconcileNodeLabelerDaemonSet indicates an expected call of ReconcileNodeLabelerDaemonSet.
func (mr *MockReconcilerMockRecorder) ReconcileNodeLabelerDaemonSet(ctx, dc interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReconcileNodeLabelerDaemonSet", reflect.TypeOf((*MockReconciler)(nil).ReconcileNodeLabelerDaemonSet), ctx, dc)
}

// SetDesiredNodeLabelerDaemonSet mocks base method.
func (m *MockReconciler) SetDesiredNodeLabelerDaemonSet(ds *v1.DaemonSet, cr *v1alpha1.DeviceConfig) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetDesiredNodeLabelerDaemonSet", ds, cr)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetDesiredNodeLabelerDaemonSet indicates an expected call of SetDesiredNodeLabelerDaemonSet.
func (mr *MockReconcilerMockRecorder) SetDesiredNodeLabelerDaemonSet(ds, cr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetDesiredNodeLabelerDaemonSet", reflect.TypeOf((*MockReconciler)(nil).SetDesiredNodeLabelerDaemonSet), ds, cr)
}
