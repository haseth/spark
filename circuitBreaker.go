// CircuitBreaker package implements circuit breaker pattern which
// acts as a proxy for a particular service. It trips the circuit
// if the request is likely to fail. In circuit tripped state the
// requests would be failed. After some halt time the circuit is
// partially closed and allows the request to pass, if fails then
// circuit trips again. Circuit counters will determine the status
// of the service.

package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

var (
	errFailed error = errors.New("Failed!! got error")
)

/*
 State defines the state of the service proxy
*/
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
 	CircuitBreaker acts a proxy for the request to a particular service
	and opens the circuit if request is likely to fail
	else lets the request pass the circuit if mostly successful
	Half-open state acts as a transition state from Open to Close State
*/
type CircuitBreaker struct {
	// UNIQUELY IDENTIFY CIRCUIT
	circuitName string

	// CURRENT CIRCUIT DATA
	currentState State
	currentTime  time.Time
	counters     *CircuitCounters

	// CLOSE TO OPEN FUNCTION
	tripCircuit func(CircuitCounters) bool

	// HALF-OPEN TO CLOSE FUNCTION
	untripCircuit func(CircuitCounters) bool

	// OPEN TO HALF_OPEN DURATION
	openTime time.Duration

	lock *sync.Mutex
}

/*
	CircuitCounters defines all the counters
	which is used to determine the health of
	circuit.
*/
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
	Spark the request in the circuit
	if the circuit is close/half-open request would be passed
	if the circuit is open request would be failed
*/
func (cb *CircuitBreaker) Spark(request func() (interface{}, error)) (interface{}, error) {
	if isOpen(cb) {
		// create a constant error
		return nil, errors.New("error circuit open")
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

// Whenever state changes we reset the counters
func updateState(cb *CircuitBreaker) {
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

// ResetCounters reset the counters for the circuit
// It is invoked when state changes.
func (cb *CircuitBreaker) ResetCounters() {
	cb.counters.Failure = 0
	cb.counters.Success = 0
	cb.counters.Timeout = 0
	cb.counters.Rejection = 0
}
