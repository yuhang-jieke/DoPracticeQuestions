package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

func RateLimit(perSecond int) gin.HandlerFunc {
	mu := &sync.Mutex{}
	visitors := make(map[uint]time.Time)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			cutoff := time.Now().Add(-time.Minute)
			for k, v := range visitors {
				if v.Before(cutoff) {
					delete(visitors, k)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		mu.Lock()
		last, ok := visitors[userID]
		visitors[userID] = time.Now()
		mu.Unlock()

		if ok && time.Since(last) < time.Second/time.Duration(perSecond) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "操作太频繁，请稍后再试"})
			c.Abort()
			return
		}
		c.Next()
	}
}
