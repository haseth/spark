package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	circuitbreaker "github.com/haseth/spark"
)

// Get has a Circuit Breaker middleware
func Get(url string) ([]byte, error) {
	cb := circuitbreaker.NewCircuitBreaker()
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
func main() {

	b, err := Get("http://172.16.0.100:8080")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(b))

}
