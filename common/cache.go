package common

import (
	"time"

	"github.com/patrickmn/go-cache"
)

var Cache *cache.Cache
var ReviewsCache *cache.Cache

func InitCache() {
	
	ReviewsCache = cache.New(5*time.Minute, 10*time.Minute)
	Cache = cache.New(5*time.Minute, 10*time.Minute)
}
