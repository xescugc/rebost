// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/xescugc/rebost/storing (interfaces: Membership)

// Package mock is a generated GoMock package.
package mock

import (
	gomock "github.com/golang/mock/gomock"
	storing "github.com/xescugc/rebost/storing"
	volume "github.com/xescugc/rebost/volume"
	reflect "reflect"
)

// Membership is a mock of Membership interface
type Membership struct {
	ctrl     *gomock.Controller
	recorder *MembershipMockRecorder
}

// MembershipMockRecorder is the mock recorder for Membership
type MembershipMockRecorder struct {
	mock *Membership
}

// NewMembership creates a new mock instance
func NewMembership(ctrl *gomock.Controller) *Membership {
	mock := &Membership{ctrl: ctrl}
	mock.recorder = &MembershipMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *Membership) EXPECT() *MembershipMockRecorder {
	return m.recorder
}

// Leave mocks base method
func (m *Membership) Leave() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Leave")
}

// Leave indicates an expected call of Leave
func (mr *MembershipMockRecorder) Leave() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Leave", reflect.TypeOf((*Membership)(nil).Leave))
}

// LocalVolumes mocks base method
func (m *Membership) LocalVolumes() []volume.Local {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LocalVolumes")
	ret0, _ := ret[0].([]volume.Local)
	return ret0
}

// LocalVolumes indicates an expected call of LocalVolumes
func (mr *MembershipMockRecorder) LocalVolumes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LocalVolumes", reflect.TypeOf((*Membership)(nil).LocalVolumes))
}

// Nodes mocks base method
func (m *Membership) Nodes() []storing.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Nodes")
	ret0, _ := ret[0].([]storing.Service)
	return ret0
}

// Nodes indicates an expected call of Nodes
func (mr *MembershipMockRecorder) Nodes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Nodes", reflect.TypeOf((*Membership)(nil).Nodes))
}
