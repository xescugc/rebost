// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/xescugc/rebost/membership (interfaces: Membership)

// Package mock is a generated GoMock package.
package mock

import (
	gomock "github.com/golang/mock/gomock"
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
	m.ctrl.Call(m, "Leave")
}

// Leave indicates an expected call of Leave
func (mr *MembershipMockRecorder) Leave() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Leave", reflect.TypeOf((*Membership)(nil).Leave))
}

// Volumes mocks base method
func (m *Membership) Volumes() []volume.Volume {
	ret := m.ctrl.Call(m, "Volumes")
	ret0, _ := ret[0].([]volume.Volume)
	return ret0
}

// Volumes indicates an expected call of Volumes
func (mr *MembershipMockRecorder) Volumes() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Volumes", reflect.TypeOf((*Membership)(nil).Volumes))
}
