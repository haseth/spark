package circuitbreaker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultCircuitBreaker(t *testing.T) {
	/*
		Testing Circuit Breaker with default settings
	*/
	cb := NewDefaultCircuitBreaker()

	assert.Equal(t, cb.openTime, 1*time.Second, "correct open timeout")
	assert.Equal(t, cb.circuitName, "Service-B Proxy", "correct service name")
	assert.Equal(t, cb.currentState, stateClose, "correct current state")
}

func TestNewCircuitBreaker(t *testing.T) {
	/*
		Testing Circuit Breaker with custom settings
	*/
	cb := NewCircuitBreaker("my-circuit", testTripFunc, testUntripFunc, 2)

	assert.Equal(t, cb.openTime, 2*time.Second, "correct open timeout")
	assert.Equal(t, cb.circuitName, "my-circuit", "correct service name")
	assert.Equal(t, cb.currentState, stateClose, "correct circuit state")
}

func TestSpark_DefaultSettings(t *testing.T) {
	/*
		Testing CircuitBreaker's state transition with default
		settings.
	*/
	// TEST-1
	// Circuit in initial close state and would try a request
	// Request should be successful and state should remain close.

	// setup
	cb := NewDefaultCircuitBreaker()

	// test with success call
	_, err := cb.Spark(doSuccessCall)

	// validate if success counter increases
	assert.Nil(t, err, "no error in success call")
	assert.NotZero(t, cb.counters, &CircuitCounters{Failure: 0, Success: 1, Timeout: 0, Rejection: 0}, "Success circuitcounter should be incremented")
	assert.Equal(t, cb.currentState, stateClose, "State should remain closed after successful requests")

	// TEST-2
	// Circuit in close state with 1 success counter circuit.
	// Requests would pass as circuit is in closed state.
	// As requests would fail circuit will trip to open state
	// based on default trip function.

	// test with fail call
	_, err = cb.Spark(doFailCall)

	// validate that circuit tripped
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, err, errFailed, "Error in sending request")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "Counters should be reset after state change")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 49%")

	// TEST-3
	// Request should fail if circuit in open state.

	// test with fail/success call
	_, err = cb.Spark(doSuccessCall)

	// validate request should fail as circuit in open state
	assert.NotNil(t, err, "no error in successful calls")
	assert.Equal(t, err, errOpen, "Circuit in open state should fail request")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "Counters should be reset after state change")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should be in open state for openTimeout Duration")

	// TEST-4
	// Circuit should change state from open to half-open
	// state after openTime duration and request should be
	// allowed without error

	// TODO check if there is a good way to do
	time.Sleep(1 * time.Second)

	// test with fail call and it should be tripped back to open state
	_, err = cb.Spark(doFailCall)

	// validate if circuit tripped to open state with even
	// 1 fail request in half-open state
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, err, errFailed, "error from failed request")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "Counters should be reset after state change")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if requests in half-open state fail")

	// TEST-5
	// Circuit should change state from open to half-open
	// state after openTime duration and success request should
	// be allowed without error. Circuit should be untrip to closed
	// state based on untrip function.

	// TODO check if there is a good way to do
	time.Sleep(1 * time.Second)

	// test by passing success call in half-open state
	_, err = cb.Spark(doSuccessCall)

	// validate if success request transit state from half-open to close
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "Counters should be reset after state change")
	assert.Equal(t, cb.currentState, stateClose, "Circuit should trip if error rate exceeds 50%")

	// TEST-6
	// Circuit should be able to handle panic calls
	// and increment failure counters and update state

	// assert.Panics(t, func() { cb.Spark(doPanicCall) }, "Received error from fail call")
	// assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")
	// assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "Counters should be reset after state change")
}

func TestSpark_CustomSettings(t *testing.T) {
	/*
		Testing CircuitBreaker's state transition with custom
		settings.
	*/

	// setup
	openTime := 2
	// Circuit Breaker with user-defined custom settings
	cb := NewCircuitBreaker("Service-A", testTripFunc, testUntripFunc, openTime)

	// TEST-1
	// Circuit in initial close state and would try a request
	// Request should be successful and state should remain close.

	// test with success call
	_, err := cb.Spark(doSuccessCall)

	// validate if success counter increases
	assert.Nil(t, err, "no error in success call")
	assert.NotZero(t, cb.counters, &CircuitCounters{Failure: 0, Success: 1, Timeout: 0, Rejection: 0}, "Success calls should be incremented")
	assert.Equal(t, cb.currentState, stateClose, "State should be closed after successful requests")

	// TEST-2
	// Circuit in close state with 1 success counter circuit.
	// Requests would pass as circuit is in closed state.
	// As requests would fail circuit will trip to open state
	// based on custom defined trip function.

	// test with fail call
	_, err = cb.Spark(doFailCall)

	// validate circuit state based on custom defined trip function
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, err, errFailed, "Request to service failed")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 1, Success: 1, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateClose, "Circuit should not trip if error rate did not exceeds 50%")

	// circuit will trip based on custom defined trip function if one more fail request is passed
	_, err = cb.Spark(doFailCall)
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, err, errFailed, "Request to service failed")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")

	// TEST-3
	// Request should fail if circuit in open state.

	// test with fail/success call
	_, err = cb.Spark(doSuccessCall)

	// validate request should fail as circuit in open state
	assert.NotNil(t, err, "no error in successful calls")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")

	// sleeping for less than open time state still should be open
	time.Sleep(time.Duration(openTime-1) * time.Second)
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")

	// TEST-4
	// Circuit should change state from open to half-open
	// state after openTime duration and request should be
	// allowed without error if any error state circuit would
	// be tripped again.

	time.Sleep(time.Duration(openTime) * time.Second)

	// TODO check the errror messages
	// do fail call and it should be tripped back to open state
	_, err = cb.Spark(doFailCall)

	// validate if circuit tripped to open state with even
	// 1 fail request in half-open state
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, err, errFailed, "request to service failed")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")

	// TEST-5
	// Circuit should change state from open to half-open
	// state after openTime duration and success request should
	// be allowed without error. Circuit should be untrip to closed
	// state based on untrip function.

	time.Sleep(time.Duration(openTime) * time.Second)

	// test by passing success call in half-open state
	_, err = cb.Spark(doSuccessCall)

	// validate if success request transit state from half-open to close
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateClose, "Circuit should trip if error rate exceeds 50%")

	// TEST-6
	// Circuit should be able to handle panic calls
	// and increment failure counters and update state

	// assert.Panics(t, func() { cb.Spark(doPanicCall) }, "Received error from fail call")
	// assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")
	// assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
}

func TestState(t *testing.T) {
	// Close State Test
	s := State(0)
	assert.Equal(t, s.String(), "close", "Received state name close correctly")
	assert.Equal(t, s, stateClose, "Close state correctly numbered")

	// Open State Test
	s = State(1)
	assert.Equal(t, s.String(), "open", "Received state name open correctly")
	assert.Equal(t, s, stateOpen, "Open state correctly numbered")

	// HalfOpen State Test
	s = State(2)
	assert.Equal(t, s.String(), "half-open", "Received state name half-open correctly")
	assert.Equal(t, s, stateHalfOpen, "Half-state correctly numbered")

	// Some other State Test
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
	return nil, errFailed
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
