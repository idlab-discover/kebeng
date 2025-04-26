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

type SnapOperation func() error

func (h *Handler) RegisterName(c *gin.Context) {
	h.performOperation(c, func() SnapOperation {
		snapName := fmt.Sprintf("snap_%s", uuid.New().String())
		return func() error {
			return h.Logic.RegisterName(snapName)
		}
	})
}

func (h *Handler) performOperation(c *gin.Context, operationFactory func() SnapOperation) {
	total, waitMs, concurrentParse := setupQueryParams(c)

	var wg sync.WaitGroup
	wg.Add(total)

	var successCount, failureCount int64
	var maxInFlight int64

	sem := make(chan struct{}, concurrentParse)

	for i := 0; i < total; i++ {
		op := operationFactory()

		go func(op SnapOperation) {
			defer wg.Done()
			sem <- struct{}{}
			inFlight := int64(len(sem))
			for {
				old := atomic.LoadInt64(&maxInFlight)
				if inFlight <= old || atomic.CompareAndSwapInt64(&maxInFlight, old, inFlight) {
					break
				}
			}
			err := op()
			<-sem
			if err != nil {
				atomic.AddInt64(&failureCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(op)

		if waitMs > 0 {
			time.Sleep(time.Duration(waitMs) * time.Millisecond)
		}
	}

	wg.Wait()

	c.JSON(200, gin.H{
		"requested":      total,
		"succeeded":      atomic.LoadInt64(&successCount),
		"failed":         atomic.LoadInt64(&failureCount),
		"max_concurrent": atomic.LoadInt64(&maxInFlight),
	})
}

// ================ Helper Functions ================

func setupQueryParams(c *gin.Context) (int, int, int) {
	amount := c.Query("amount")
	if amount == "" {
		amount = "1"
	}
	total, err := strconv.Atoi(amount)
	if err != nil {
		logrus.Errorf("Error converting amount to int: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount"})
		return 0, 0, 0
	}

	waitTime := c.Query("time")
	if waitTime == "" {
		waitTime = "0"
	}
	waitMs, err := strconv.Atoi(waitTime)
	if err != nil {
		logrus.Errorf("Error converting wait time to int: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid wait time"})
		return 0, 0, 0
	}

	concurrent := c.Query("concurrent")
	if concurrent == "" {
		concurrent = "1"
	}
	concurrentParse, err := strconv.Atoi(concurrent)
	if err != nil {
		logrus.Errorf("Error converting concurrent to int: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid concurrent"})
		return 0, 0, 0
	}

	return total, waitMs, concurrentParse
}
