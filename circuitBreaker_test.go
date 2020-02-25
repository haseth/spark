package circuitbreaker

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCircuitBreaker(t *testing.T) {
	/*
		Testing Circuit Breaker with default values
	*/
	cb := NewCircuitBreaker()

	assert.Equal(t, cb.openTime, 1*time.Second, "Got correct open timeout")
	assert.Equal(t, cb.circuitName, "Service-B Proxy", "Got correct service name")
	assert.Equal(t, cb.currentState, stateClose, "Deafult state is closed")
}

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

func TestSpark(t *testing.T) {
	// Default circuit breaker with default trip function
	cb := NewCircuitBreaker()

	_, err := cb.Spark(doSuccessCall)
	assert.Nil(t, err, "Received no error")
	assert.NotZero(t, cb.counters, &CircuitCounters{Failure: 0, Success: 1, Timeout: 0, Rejection: 0}, "Success calls should be incremented")
	assert.Equal(t, cb.currentState, stateClose, "State should be closed after successful requests")

	_, err = cb.Spark(doFailCall)
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")

	// Circuit tripped success/ fail request should be tripped without beign sent to server
	_, err = cb.Spark(doFailCall)
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")

	// TODO check if there is a good way to do
	// currently in open state, we need to wait for circuit to come in half-close so that we can panic
	time.Sleep(1 * time.Second)

	// TODO check the errror messages
	// currently in half-open state
	// Success call which should untrip the circuit
	_, err = cb.Spark(doSuccessCall)
	assert.Nil(t, err, "Received error from fail call")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 1, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateHalfOpen, "Circuit should trip if error rate exceeds 50%")

	// do fail call and it should be tripped back to open state
	_, err = cb.Spark(doFailCall)
	assert.NotNil(t, err, "Received error from fail call")
	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")

	time.Sleep(1 * time.Second)

	// state should be in half-state
	// make atleast 2 calls to make circuit closed based on default untrip method

	_, err = cb.Spark(doSuccessCall)
	_, err = cb.Spark(doSuccessCall)

	assert.Equal(t, cb.counters, &CircuitCounters{Failure: 0, Success: 0, Timeout: 0, Rejection: 0}, "State should be closed after successful requests")
	assert.Equal(t, cb.currentState, stateClose, "Circuit should trip if error rate exceeds 50%")

	// assert.Panics(t, func() { cb.Spark(doPanicCall) }, "Received error from fail call")
	// //assert.Equal(t, cb.currentState, stateOpen, "Circuit should trip if error rate exceeds 50%")
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
