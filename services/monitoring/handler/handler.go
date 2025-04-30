package handler

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/idlab-discover/kebeng/services/monitoring/internal/logic"
	"github.com/idlab-discover/kebeng/services/monitoring/internal/model"
	"github.com/idlab-discover/kebeng/services/monitoring/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type SnapOperation func() error

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
	r.POST("/snapcraft-upload", h.SnapcraftUpload)
	r.POST("/snapd-download", h.SnapdDownload)
}

func (h *Handler) RegisterName(c *gin.Context) {
	h.performOperation(c, func() SnapOperation {
		snapName := fmt.Sprintf("snap%s", uuid.New().String())
		return func() error {
			return h.Logic.RegisterName(snapName)
		}
	})
}

func (h *Handler) SnapcraftUpload(c *gin.Context) {
	var req model.SnapcraftUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	snaps := req.SnapNames

	h.performOperation(c, func() SnapOperation {
		return func() error {
			rc, snapName, err := util.RandomSnapReader(snaps, 30, h.Logic.Config.SnapDataPath)
			logrus.Debugf("Random snap name: %s", snapName)
			if err != nil {
				return fmt.Errorf("failed to create multi-source reader: %w", err)
			}
			defer rc.Close()

			err = h.Logic.RegisterName(snapName)
			if err != nil {
				return err
			}
			logrus.Debugf("Registered snap name: %s", snapName)

			pushRequest := model.SnapPushRequest{
				Name:   snapName,
				DryRun: true,
			}
			pushResp, err := h.Logic.SnapPush(pushRequest)
			if err != nil {
				return err
			}
			logrus.Debugf("Push response 1: %+v", pushResp)
			resp, err := h.Logic.UnscannedUpload(rc, snapName)
			if err != nil {
				return err
			}
			logrus.Debugf("Unscanned upload: %+v", resp)
			pushRequest = model.SnapPushRequest{
				Name:              snapName,
				UnscannedFileName: resp.UploadID,
				BinaryFileSize:    resp.Size,
			}
			pushResp, err = h.Logic.SnapPush(pushRequest)
			if err != nil {
				return err
			}
			logrus.Debugf("Push response 2: %+v", pushResp)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return fmt.Errorf("timed out waiting for upload %s to process", pushResp.UploadID)
				case <-ticker.C:
					status, err := h.Logic.GetUploadStatus(pushResp.UploadID)
					if err != nil {
						return fmt.Errorf("failed to get upload status: %w", err)
					}
					logrus.Debugf("Upload status: %+v", status)
					if status.Processed {
						return nil
					}
				}
			}
		}
	})
}

func (h *Handler) SnapdDownload(c *gin.Context) {
	var req model.SnapDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	h.performOperation(c, func() SnapOperation {
		return func() error {
			// 1) Get URL and download
			refreshResp, err := h.Logic.RefreshDownload(req.SnapName, req.Channel)
			if err != nil {
				return err
			}
			url := *refreshResp.Responses[0].Snap.Download.URL
			if err := h.Logic.SnapDownload(url); err != nil {
				return err
			}

			// 2) Revision assertion
			// NOTE: snapd calculates the sha themselves and checks with ours but we use our own for easy of testing
			hexSha := *refreshResp.Responses[0].Snap.Download.Sha3_384
			b, err := hex.DecodeString(hexSha)
			if err != nil {
				// handle
			}
			sha := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b)
			revBlob, err := h.Logic.GetSnapRevisionAssertion(sha, "0")
			if err != nil {
				return err
			}
			revFields := util.ParseAssertion(revBlob)
			snapID := revFields["snap-id"]
			nextKey := revFields["sign-key-sha3-384"]

			// 3) Declaration assertion
			declBlob, err := h.Logic.GetSnapDeclarationAssertion("16", snapID)
			if err != nil {
				return err
			}
			declFields := util.ParseAssertion(declBlob)
			// sometimes declaration uses the same key, but could differ:
			nextKey = declFields["sign-key-sha3-384"]

			// 4) Now climb the key/account chain
			seen := map[string]bool{}
			for {
				if seen[nextKey] {
					return fmt.Errorf("cycle detected in key chain at %s", nextKey)
				}
				seen[nextKey] = true

				// fetch account-key assertion
				keyBlob, err := h.Logic.GetAccountKeyAssertion(nextKey, "0")
				if err != nil {
					return err
				}
				keyFields := util.ParseAssertion(keyBlob)
				accountID := keyFields["account-id"]

				// fetch account assertion
				acctBlob, err := h.Logic.GetAccountAssertion(accountID, "0")
				if err != nil {
					return err
				}
				logrus.Debugf("Account assertion blob: %s", acctBlob)
				acctFields := util.ParseAssertion(acctBlob)
				displayName := acctFields["display-name"]
				logrus.Debugf("Account displayName of account assertion: %s", displayName)
				if displayName == "kebeng" && acctFields["validation"] == "certified" {
					// we trust this account—done!
					return nil
				}
				// otherwise, climb again using the sign-key from this account
				nextKey = acctFields["sign-key-sha3-384"]
			}
		}
	})
}

// ================ Helper Functions ================

func (h *Handler) performOperation(c *gin.Context, operationFactory func() SnapOperation) {
	total, waitMs, concurrentParse := setupQueryParams(c)

	var wg sync.WaitGroup
	wg.Add(total)

	var (
		successCount  int64
		failureCount  int64
		maxInFlight   int64
		latestErr     error
		latestErrLock sync.Mutex
	)

	sem := make(chan struct{}, concurrentParse)

	for range total {
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
				logrus.Errorf("operation failed: %v", err)
				atomic.AddInt64(&failureCount, 1)

				latestErrLock.Lock()
				latestErr = err
				latestErrLock.Unlock()
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(op)

		if waitMs > 0 {
			time.Sleep(time.Duration(waitMs) * time.Millisecond)
		}
	}

	wg.Wait()

	// read out latestErr under lock
	latestErrLock.Lock()
	errMsg := ""
	if latestErr != nil {
		errMsg = latestErr.Error()
	}
	latestErrLock.Unlock()

	c.JSON(200, gin.H{
		"requested":      total,
		"succeeded":      atomic.LoadInt64(&successCount),
		"failed":         atomic.LoadInt64(&failureCount),
		"max_concurrent": atomic.LoadInt64(&maxInFlight),
		"latest_error":   errMsg,
	})
}

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
