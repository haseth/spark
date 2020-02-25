/*
	CircuitBreaker package implements circuit breaker pattern which
	acts as a proxy for a particular remote service. It trips the circuit
	if requests are likely to be failed to remote service and untrips it
	after requests would be successful.

	Trip function is used to open the circuit based on the circuit counters
	Ex. if fail/(fail+success) > 0.5, trip circuit.

	Circuit remains in open state for OpenTime duration and then changes
	to half-open state where the service is monitored.

	UnTrip function is used to close the circuit from half-open state based on
	the circuit counters.
	Ex. if success/(fail+success) > 0.9, untrip circuit if tripped.

	Circuit counters will determine the status of the service.
*/
package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

var (
	errFailed error = errors.New("Failed!! got error")
	errOpen   error = errors.New("Circuit Open")
)

// State defines the state of the circuit
// i.e. Open, Close or half-open
type State int

const (
	stateClose    State = iota
	stateOpen     State = iota
	stateHalfOpen State = iota
)

func (s State) String() string {
	if int(s) < 0 || int(s) > 2 {
		return "incorrect state requested"
	}
	return []string{"close", "open", "half-open"}[int(s)]
}

/*
 	CircuitBreaker acts as proxy for requests to a particular service.
	It opens the circuit if requests are likely to get fail otherwise
	allows the requests to pass the circuit.
*/
type CircuitBreaker struct {
	circuitName string

	currentState State
	currentTime  time.Time
	counters     *CircuitCounters

	// func to transit circuit state from close to open state
	tripCircuit func(CircuitCounters) bool

	// func to transit circuit state from half-open to close state
	untripCircuit func(CircuitCounters) bool

	// time duration for circuit to be in open state before transit
	// to half-open state
	openTime time.Duration

	lock *sync.Mutex
}

/*
	CircuitCounters are counters for the circuit
	which is used to determine/change the state of
	circuit.
*/
// TODO implement Timeout and Rejection counter feedback to circuit
type CircuitCounters struct {
	Failure   int64
	Success   int64
	Timeout   int64
	Rejection int64
}

// NewDefaultCircuitBreaker returns circuitbreaker with default settings
func NewDefaultCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		circuitName:  "Service-B Proxy",
		currentState: stateClose,
		currentTime:  time.Now(),

		tripCircuit: func(counter CircuitCounters) bool {
			fail := float64(counter.Failure)
			success := float64(counter.Success)

			if (fail+success > 0) && fail/(fail+success) >= 0.50 {
				return true
			}
			return false
		},
		untripCircuit: func(counter CircuitCounters) bool {
			fail := float64(counter.Failure)
			success := float64(counter.Success)

			if (fail+success > 0) && success/(fail+success) >= 0.50 {
				return true
			}
			return false
		},

		openTime: 1 * time.Second,
		counters: &CircuitCounters{},
		lock:     &sync.Mutex{},
	}
}

// NewCircuitBreaker returns circuitbreaker with custom settings
func NewCircuitBreaker(circuitName string, tripFunc, untripFunc func(CircuitCounters) bool, openT int) *CircuitBreaker {
	return &CircuitBreaker{
		circuitName:  circuitName,
		currentState: stateClose,
		currentTime:  time.Now(),

		tripCircuit:   tripFunc,
		untripCircuit: untripFunc,

		openTime: time.Duration(openT) * time.Second,
		counters: &CircuitCounters{},
		lock:     &sync.Mutex{},
	}
}

/*
	Spark requests in the circuit of remote service
	if the circuit is in close/half-open state request would be passed
	else if the circuit is in open state request would be failed
*/
func (cb *CircuitBreaker) Spark(request func() (interface{}, error)) (interface{}, error) {
	if isOpen(cb) {
		// create a constant error
		return nil, errOpen
	}

	req, err := request()
	// TODO Need to test with panic function
	defer func() {
		e := recover()
		if e != nil {
			onFail(cb)
			panic(e)
		}
	}()

	if err != nil {
		onFail(cb)
		return req, err
	}
	onSuccess(cb)
	return req, nil
}

func isOpen(cb *CircuitBreaker) bool {
	// isOpen veirifies if circuit is open or not
	cb.lock.Lock()
	defer cb.lock.Unlock()

	updateState(cb)
	state := cb.currentState
	if state == stateOpen {
		return true
	}
	return false
}

func onFail(cb *CircuitBreaker) {
	// increment the failure counter and update state
	cb.lock.Lock()
	defer cb.lock.Unlock()

	cb.counters.Failure++
	updateState(cb)
}

func onSuccess(cb *CircuitBreaker) {
	// increment the success counter and update state
	cb.lock.Lock()
	defer cb.lock.Unlock()

	cb.counters.Success++
	updateState(cb)
}

func updateState(cb *CircuitBreaker) {
	// whenever state changes we reset the counters
	switch cb.currentState {
	case stateClose:
		if cb.tripCircuit(*cb.counters) {
			cb.currentState = stateOpen
			cb.currentTime = time.Now()
			cb.ResetCounters()
		}
	case stateHalfOpen:
		if cb.counters.Failure > 0 {
			cb.currentState = stateOpen
			cb.currentTime = time.Now()
			cb.ResetCounters()
		}
		if cb.untripCircuit(*cb.counters) {
			cb.currentState = stateClose
			cb.currentTime = time.Now()
			cb.ResetCounters()
		}
	case stateOpen:
		if cb.currentTime.Add(cb.openTime).Before(time.Now()) {
			cb.currentState = stateHalfOpen
			cb.currentTime = time.Now()
			cb.ResetCounters()
		}
	}
}

// ResetCounters will reset circuit counters
// It is invoked when state changes
func (cb *CircuitBreaker) ResetCounters() {
	cb.counters.Failure = 0
	cb.counters.Success = 0
	cb.counters.Timeout = 0
	cb.counters.Rejection = 0
}
