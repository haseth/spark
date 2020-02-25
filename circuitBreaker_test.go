package circuitbreaker

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultCircuitBreaker(t *testing.T) {
	/*
		Testing Circuit Breaker with default values
	*/
	cb := NewDefaultCircuitBreaker()

	assert.Equal(t, cb.openTime, 1*time.Second, "Got correct open timeout")
	assert.Equal(t, cb.circuitName, "Service-B Proxy", "Got correct service name")
	assert.Equal(t, cb.currentState, stateClose, "Deafult state is closed")
}

func TestNewCircuitBreaker(t *testing.T) {
	/*
		Testing Circuit Breaker with custom settings
	*/
	cb := NewCircuitBreaker("my-circuit", testTripFunc, testUntripFunc, 1)

	assert.Equal(t, cb.openTime, 1*time.Second, "Got correct open timeout")
	assert.Equal(t, cb.circuitName, "my-circuit", "Got correct service name")
	assert.Equal(t, cb.currentState, stateClose, "Deafult state is closed")
}

func TestSpark_DefaultSettings(t *testing.T) {
	// Default circuit breaker with default trip function
	cb := NewDefaultCircuitBreaker()

	_, err := cb.Spark(doSuccessCall)
	assert.Nil(t, err, "Received no error")
	assert.NotZero(t, cb.counters, &CircuitCounters{Failure: 0, Success: 1, Timeout: 0, Rejection: 0}, "Success calls should be incremented")
	assert.Equal(t, cb.currentState, stateClose, "State should be closed after successful requests")

	_, err = cb.Spark(doFailCall)
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")

	// Circuit tripped success/fail request should be tripped without beign sent to server
	_, err = cb.Spark(doFailCall)
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")

	// TODO check if there is a good way to do
	// currently in open state, we need to wait for circuit to come in half-close so that we can panic
	time.Sleep(1 * time.Second)

	// TODO check the errror messages

	// do fail call and it should be tripped back to open state
	_, err = cb.Spark(doFailCall)
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")

	time.Sleep(1 * time.Second)

	// state should be in half-state
	// make atleast 2 calls to make circuit closed based on default untrip method

	_, err = cb.Spark(doSuccessCall)
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateClose, "Circuit should trip if error rate exceeds 50%")

	// assert.Panics(t, func() { cb.Spark(doPanicCall) }, "Received error from fail call")
	// assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")
	// assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
}

func TestSpark_CustomSettings(t *testing.T) {
	openTime := 2
	// Default circuit breaker with default trip function
	cb := NewCircuitBreaker("Service-A", testTripFunc, testUntripFunc, openTime)

	_, err := cb.Spark(doSuccessCall)
	assert.Nil(t, err, "Received no error")
	assert.NotZero(t, cb.counters, &CircuitCounters{Failure: 0, Success: 1, Timeout: 0, Rejection: 0}, "Success calls should be incremented")
	assert.Equal(t, cb.currentState, stateClose, "State should be closed after successful requests")

	_, err = cb.Spark(doFailCall)
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 1, Success: 1, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateClose, "Circuit should trip if error rate exceeds 50%")

	// Circuit tripped success/fail request should be tripped without beign sent to server
	_, err = cb.Spark(doFailCall)
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")

	// sleeping for less than open time still state should be open
	time.Sleep(time.Duration(openTime-1) * time.Second)
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")

	time.Sleep(time.Duration(openTime) * time.Second)

	// TODO check the errror messages
	// do fail call and it should be tripped back to open state
	_, err = cb.Spark(doFailCall)
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")

	time.Sleep(time.Duration(openTime) * time.Second)

	// state should be in half-state
	// make atleast 1 calls to make circuit closed based on default untrip method

	_, err = cb.Spark(doSuccessCall)
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateClose, "Circuit should trip if error rate exceeds 50%")

	// assert.Panics(t, func() { cb.Spark(doPanicCall) }, "Received error from fail call")
	// assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")
	// assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
}

func TestState(t *testing.T) {
	s := State(0)
	assert.Equal(t, s.String(), "close", "Received state name close correctly")
	assert.Equal(t, s, stateClose, "Close state correctly numbered")

	s = State(1)
	assert.Equal(t, s.String(), "open", "Received state name open correctly")
	assert.Equal(t, s, stateOpen, "Open state correctly numbered")

	s = State(2)
	assert.Equal(t, s.String(), "half-open", "Received state name half-open correctly")
	assert.Equal(t, s, stateHalfOpen, "Half-state correctly numbered")

	s = State(3)
	assert.Equal(t, s.String(), "incorrect state requested", "Received state name half-open correctly")
}

// Helper functions
func doSuccessCall() (interface{}, error) {
	// heavy load function
	return nil, nil
}

func doFailCall() (interface{}, error) {
	// heavy load function with error
	return nil, errors.New("Some Error")
}

func doPanicCall() (interface{}, error) {
	// heavy load with panic call
	panic("Error: in calll")
	return nil, nil
}

func testTripFunc(counter CircuitCounters) bool {
	// func to trip circuit from close state to open state
	fail := float64(counter.Failure)
	success := float64(counter.Success)

	if (fail+success > 0) && fail/(fail+success) > 0.50 {
		return true
	}
	return false
}
func testUntripFunc(counter CircuitCounters) bool {
	fail := float64(counter.Failure)
	success := float64(counter.Success)

	if (fail+success > 0) && success/(fail+success) > 0.50 {
		return true
	}
	return false
}
