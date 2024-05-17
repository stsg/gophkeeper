// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package server

import (
	"sync"

	"github.com/stsg/gophkeeper/app/status"
)

// StatusMock is a mock implementation of Status.
//
// 	func TestSomethingThatUsesStatus(t *testing.T) {
//
// 		// make and configure a mocked Status
// 		mockedStatus := &StatusMock{
// 			GetFunc: func() (*status.Info, error) {
// 				panic("mock out the Get method")
// 			},
// 		}
//
// 		// use mockedStatus in code that requires Status
// 		// and then make assertions.
//
// 	}
type StatusMock struct {
	// GetFunc mocks the Get method.
	GetFunc func() (*status.Info, error)

	// calls tracks calls to the methods.
	calls struct {
		// Get holds details about calls to the Get method.
		Get []struct {
		}
	}
	lockGet sync.RWMutex
}

// Get calls GetFunc.
func (mock *StatusMock) Get() (*status.Info, error) {
	if mock.GetFunc == nil {
		panic("StatusMock.GetFunc: method is nil but Status.Get was just called")
	}
	callInfo := struct {
	}{}
	mock.lockGet.Lock()
	mock.calls.Get = append(mock.calls.Get, callInfo)
	mock.lockGet.Unlock()
	return mock.GetFunc()
}

// GetCalls gets all the calls that were made to Get.
// Check the length with:
//     len(mockedStatus.GetCalls())
func (mock *StatusMock) GetCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockGet.RLock()
	calls = mock.calls.Get
	mock.lockGet.RUnlock()
	return calls
}
