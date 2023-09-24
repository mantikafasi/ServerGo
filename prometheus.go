package main

import (
	"server-go/modules"

	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	prometheus.MustRegister(ReviewCounter)
	prometheus.MustRegister(ReviewDBUserCounter)
	prometheus.MustRegister(TotalRequestCounter)
}

var TotalRequestCounter = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "total_request",
	Help: "Total request count",
})

var ReviewDBUserCounter = prometheus.NewCounterFunc(prometheus.CounterOpts{
	Name: "user_count",
	Help: "Count of user reviews users",
}, func() float64 {
	userCount, err := modules.GetURUserCount()

	if err != nil {
		return 0
	}

	return float64(userCount)
})

var ReviewCounter = prometheus.NewCounterFunc(prometheus.CounterOpts{
	Name: "review_count",
	Help: "Count of total user reviews",
}, func() float64 {
	count, err := modules.GetReviewCount()

	if err != nil {
		return 0
	}

	return float64(count)
})
