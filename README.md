# Circuit Breaker 

The Circuit Breaker pattern can prevent an application repeatedly trying to execute an operation that is likely to fail, allowing it to continue without waiting for the fault to be rectified or wasting CPU cycles while it determines that the fault is long lasting. The Circuit Breaker pattern also enables an application to detect whether the fault has been resolved. If the problem appears to have been rectified, the application can attempt to invoke the operation.

 Circuit break will act as a proxy which would determine if request would likely succeed or not. If not then it will return error. 

# Usage 
The struct `CircuitBreaker` is a state machine to prevent sending requests that are likely to fail.
The function `NewCircuitBreaker` creates a new `CircuitBreaker`.

```go
func NewCircuitBreaker(st Settings) *CircuitBreaker
```

``` Currently it only supports dedault settings. ```

- `circuitName` is the name of the `CircuitBreaker`.

- `currentState` and `currentTime` defines the last state update and it's time. 

- `tripCircuit` defines a method to trip the circuit based on the circuit counters. 
```
Ex. 
If fail/(success+fail)>0.90 then trip the circuit. 
```
- `untripCircuit` defines a method to untrip the circuit from half-open to close state. 

```
Ex. 
If success requests crosses 10 then untrip the circuit

```

- `openTime` defines the time after which circuit could transition from ```open``` to ```hal-open state ```

The struct `CircuitCounters` holds the numbers of requests and their successes/failures:

```go
type CircuitCounters struct {
	Failure   int64
	Success   int64
	Timeout   int64
	Rejection int64
}
```

`CircuitBreaker` clears the internal `Counts`
on the change of the state.
`Counts` ignores the results of the requests sent before clearing.

`CircuitBreaker` can wrap any function to send a request:

```go
func (cb *CircuitBreaker) Spark(req func() (interface{}, error)) (interface{}, error)
```

The method `Spark` runs the given request if `CircuitBreaker` accepts it.
`Spark` returns an error instantly if `CircuitBreaker` rejects the request.
Otherwise, `Spark` returns the result of the request.
If a panic occurs in the request, `Spark` handles it as an error
and causes the same panic again.

Example
-------

```go
var cb *breaker.CircuitBreaker

func Get(url string) ([]byte, error) {
	body, err := cb.Spark(func() (interface{}, error) {
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		return body, nil
	})
	if err != nil {
		return nil, err
	}

	return body.([]byte), nil
}
```

License
-------

The MIT License (MIT)


# Example 

# TODO 

- Add license 
- Add more functionality and test cases to library 
- Update README 
- Add examples 

