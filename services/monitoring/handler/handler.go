package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"monitoring/internal/logic"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	Logic *logic.Logic
}

func NewHandler(logic *logic.Logic) *Handler {
	return &Handler{
		Logic: logic,
	}
}

func (h *Handler) SetupEndpoints(r *gin.Engine) {
	r.GET("/register-name", h.RegisterName)
}

func (h *Handler) RegisterName(c *gin.Context) {
	amount := c.Query("amount")
	if amount == "" {
		amount = "1"
	}
	total, err := strconv.Atoi(amount)
	if err != nil {
		logrus.Errorf("Error converting amount to int: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount"})
		return
	}

	waitTime := c.Query("time")
	if waitTime == "" {
		// no sleep
		waitTime = "0"
	}
	waitMs, err := strconv.Atoi(waitTime)
	if err != nil {
		logrus.Errorf("Error converting wait time to int: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wait time"})
		return
	}

	concurrent := c.Query("concurrent")
	if concurrent == "" {
		concurrent = "1"
	}
	concurrentParse, err := strconv.Atoi(concurrent)
	if err != nil {
		logrus.Errorf("Error converting concurrent to int: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid concurrent"})
		return
	}

	var wg sync.WaitGroup
	wg.Add(total)

	var successCount, failureCount int64

	var maxInFlight int64
	// limit concurent requests
	sem := make(chan struct{}, concurrentParse)

	for range total {
		snapName := fmt.Sprintf("snap_%s", uuid.New().String())

		go func(name string) {
			defer wg.Done()
			sem <- struct{}{} // acquire a token
			inFlight := int64(len(sem))
			for {
				old := atomic.LoadInt64(&maxInFlight)
				if inFlight <= old || atomic.CompareAndSwapInt64(&maxInFlight, old, inFlight) {
					break
				}
			}
			err := h.Logic.RegisterName(name)
			<-sem // release token
			if err != nil {
				atomic.AddInt64(&failureCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(snapName)
		if waitMs > 0 {
			time.Sleep(time.Duration(waitMs) * time.Millisecond)
		}
	}

	wg.Wait() // wait for all goroutines to finish

	c.JSON(200, gin.H{
		"requested":      total,
		"succeeded":      atomic.LoadInt64(&successCount),
		"failed":         atomic.LoadInt64(&failureCount),
		"max_concurrent": atomic.LoadInt64(&maxInFlight),
	})
}
