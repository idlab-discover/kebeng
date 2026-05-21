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

	"github.com/idlab-discover/kebeng/common/monitoring"
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
	r.POST("/cohort-keys", h.CohortKeys)
	r.POST("/find-snaps", h.FindSnaps)
	r.POST("/delta-upload", h.DeltaUpload)
	r.POST("/delta-download", h.DeltaDownload)
}

func (h *Handler) RegisterName(c *gin.Context) {
	total, _, concurrentParse := setupQueryParams(c)
	stamp := time.Now().Format("2006-01-02_15-04-05")
	err := monitoring.InitFileRecorder(fmt.Sprintf("/var/log/request_duration_%s_%s_%d_%d.csv", "register-name", stamp, concurrentParse, total), 5*time.Second, 10000)
	if err != nil {
		logrus.Errorf("failed to initialize file recorder: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize monitoring"})
		return
	}
	defer monitoring.ShutdownFileRecorder()

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

	total, _, concurrentParse := setupQueryParams(c)
	stamp := time.Now().Format("2006-01-02_15-04-05")
	err := monitoring.InitFileRecorder(fmt.Sprintf("/var/log/request_duration_%s_%s_%d_%d_%s.csv", "upload", stamp, concurrentParse, total, snaps[0]), 5*time.Second, 10000)
	if err != nil {
		logrus.Errorf("failed to initialize file recorder: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize monitoring"})
		return
	}

	defer monitoring.ShutdownFileRecorder()

	h.performOperation(c, func() SnapOperation {
		return func() error {
			rc, snapName, err := util.RandomSnapReader(snaps, 30, h.Logic.Config.SnapDataPath)
			logrus.Debugf("random snap name: %s", snapName)
			if err != nil {
				return fmt.Errorf("failed to create multi-source reader: %w", err)
			}
			defer rc.Close()

			stop := monitoring.StartMonitoringTimer("register_name")
			err = h.Logic.RegisterName(snapName)
			stop()
			if err != nil {
				return err
			}
			logrus.Debugf("registered snap name: %s", snapName)

			pushRequest := model.SnapPushRequest{
				Name:   snapName,
				DryRun: true,
			}
			stop = monitoring.StartMonitoringTimer("snap_push_dry_run")
			pushResp, err := h.Logic.SnapPush(pushRequest)
			stop()
			if err != nil {
				return err
			}
			logrus.Debugf("push response 1: %+v", pushResp)

			stop = monitoring.StartMonitoringTimer("unscanned_upload")
			resp, err := h.Logic.UnscannedUpload(rc, snapName)
			stop()
			if err != nil {
				return err
			}
			logrus.Debugf("unscanned upload: %+v", resp)
			pushRequest = model.SnapPushRequest{
				Name:              snapName,
				UnscannedFileName: resp.UploadID,
				BinaryFileSize:    resp.Size,
				Series:            "20",
			}

			stop = monitoring.StartMonitoringTimer("snap_push")
			pushResp, err = h.Logic.SnapPush(pushRequest)
			stop()
			if err != nil {
				return err
			}
			logrus.Debugf("push response 2: %+v", pushResp)
			stopProcessing := monitoring.StartMonitoringTimer("snap_processing")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					stopProcessing()
					return fmt.Errorf("timed out waiting for upload %s to process", pushResp.UploadID)
				case <-ticker.C:
					status, err := h.Logic.GetUploadStatus(pushResp.UploadID)
					if err != nil {
						stopProcessing()
						return fmt.Errorf("failed to get upload status: %w", err)
					}
					logrus.Debugf("upload status: %+v", status)
					if status.Processed {
						stopProcessing()
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

	total, _, concurrentParse := setupQueryParams(c)
	stamp := time.Now().Format("2006-01-02_15-04-05")
	err := monitoring.InitFileRecorder(fmt.Sprintf("/var/log/request_duration_%s_%s_%d_%d_%s.csv", "download", stamp, concurrentParse, total, req.SnapName), 5*time.Second, 10000)
	if err != nil {
		logrus.Errorf("failed to initialize file recorder: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize monitoring"})
		return
	}

	defer monitoring.ShutdownFileRecorder()

	h.performOperation(c, func() SnapOperation {
		return func() error {
			// 1) Get URL and download
			stop := monitoring.StartMonitoringTimer("snap_refresh")
			refreshResp, err := h.Logic.RefreshDownload(req.SnapName, req.Channel)
			stop()
			if err != nil {
				return err
			}
			url := *refreshResp.Responses[0].Snap.Download.URL
			stop = monitoring.StartMonitoringTimer("snap_download")
			if err := h.Logic.SnapDownload(url); err != nil {
				return err
			}
			stop()

			// 2) Revision assertion
			// NOTE: snapd calculates the sha themselves and checks with ours but we use our own for easy of testing
			hexSha := *refreshResp.Responses[0].Snap.Download.Sha3_384
			b, err := hex.DecodeString(hexSha)
			if err != nil {
				return fmt.Errorf("failed to decode hex string: %w", err)
			}
			sha := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b)
			stop = monitoring.StartMonitoringTimer("snap_revision_assertion")
			revBlob, err := h.Logic.GetSnapRevisionAssertion(sha, "0")
			stop()
			if err != nil {
				return err
			}
			revFields := util.ParseAssertion(revBlob)
			snapID := revFields["snap-id"]

			// 3) Declaration assertion
			stop = monitoring.StartMonitoringTimer("snap_declaration_assertion")
			declBlob, err := h.Logic.GetSnapDeclarationAssertion("16", snapID)
			stop()
			if err != nil {
				return err
			}
			declFields := util.ParseAssertion(declBlob)
			// sometimes declaration uses the same key, but could differ:
			nextKey := declFields["sign-key-sha3-384"]

			// 4) Now climb the key/account chain
			seen := map[string]bool{}
			for {
				if seen[nextKey] {
					return fmt.Errorf("cycle detected in key chain at %s", nextKey)
				}
				seen[nextKey] = true

				// fetch account-key assertion
				stop = monitoring.StartMonitoringTimer("account_key_assertion")
				keyBlob, err := h.Logic.GetAccountKeyAssertion(nextKey, "0")
				stop()
				if err != nil {
					return err
				}
				keyFields := util.ParseAssertion(keyBlob)
				accountID := keyFields["account-id"]

				// fetch account assertion
				stop = monitoring.StartMonitoringTimer("account_assertion")
				acctBlob, err := h.Logic.GetAccountAssertion(accountID, "0")
				stop()
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

func (h *Handler) CohortKeys(c *gin.Context) {
	var req model.CohortKeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch_size is required."})
		return
	}

	total, _, concurrentParse := setupQueryParams(c)
	stamp := time.Now().Format("2006-01-02_15-04-05")
	err := monitoring.InitFileRecorder(
		fmt.Sprintf(
			"/var/log/request_duration_%s_%s_%d_%d_%d.csv",
			"cohort-keys", stamp, concurrentParse, total, req.BatchSize,
		),
		5*time.Second,
		10000,
	)
	if err != nil {
		logrus.Errorf("failed to initialize file recorder: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize monitoring"})
		return
	}
	defer monitoring.ShutdownFileRecorder()

	// Snap provisioning: cohorts can only be created for registered snaps
	// Create batch_size snap registrations to use for the benchmark
	// NOTE: This isn't actually what is benchmarked, so this isn't timed
	snapNames := make([]string, 0, req.BatchSize)
	for i := range req.BatchSize {

		name := fmt.Sprintf("cohort-bench-%s", uuid.New().String())

		if err := h.Logic.RegisterName(name); err != nil {
			logrus.Errorf("failed to provision snap %d/%d (%s): %v", i+1, req.BatchSize, name, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("provisioning failed at snap %d: %v", i+1, err),
			})
			return
		}

		snapNames = append(snapNames, name)

	}

	logrus.Infof("Provisioned %d snap names for cohort benchmark", len(snapNames))

	h.performOperation(c, func() SnapOperation {
		return func() error {
			stop := monitoring.StartMonitoringTimer("create_cohorts")
			_, err := h.Logic.CreateCohorts(snapNames)
			stop()
			return err
		}
	})
}

func (h *Handler) FindSnaps(c *gin.Context) {
	var req model.FindSnapsRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	total, _, concurrentParse := setupQueryParams(c)
	stamp := time.Now().Format("2006-01-02_15-04-05")

	queryLabel := req.Query
	if queryLabel == "" {
		queryLabel = "emptyquery"
	}

	err := monitoring.InitFileRecorder(
		fmt.Sprintf(
			"/var/log/request_duration_%s_%s_%d_%d_%s.csv",
			"find-snaps", stamp, concurrentParse, total, queryLabel,
		),
		5*time.Second,
		10000,
	)

	if err != nil {
		logrus.Errorf("failed to initialize file recorder: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize monitoring"})
		return
	}
	defer monitoring.ShutdownFileRecorder()

	h.performOperation(c, func() SnapOperation {
		return func() error {
			stop := monitoring.StartMonitoringTimer("find_snaps")
			_, err := h.Logic.FindSnaps(req)
			stop()
			return err
		}
	})
}

func (h *Handler) DeltaUpload(c *gin.Context) {
	var req model.DeltaUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 60
	}

	total, _, concurrentParse := setupQueryParams(c)
	stamp := time.Now().Format("2006-01-02_15-04-05")
	err := monitoring.InitFileRecorder(
		fmt.Sprintf("/var/log/request_duration_%s_%s_%d_%d_%s.csv",
			"delta-upload", stamp, concurrentParse, total, req.SnapName),
		5*time.Second,
		10000,
	)
	if err != nil {
		logrus.Errorf("failed to initialize file recorder: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize monitoring"})
		return
	}
	defer monitoring.ShutdownFileRecorder()

	snapNames := make([]string, 0, total)
	for i := range total {
		name := fmt.Sprintf("%s-%s", req.SnapName, uuid.New().String())

		if err := h.Logic.RegisterName(name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("provisioning: register failed at %d: %v", i, err),
			})
			return
		}
		rc, _, err := util.DeltaFileReader(req.BaseSnapFilePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("provisioning: open snap failed at %d: %v", i, err),
			})
			return
		}

		uploadResp, err := h.Logic.UnscannedUpload(rc, name)
		rc.Close()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("provisioning: upload failed at %d: %v", i, err),
			})
			return
		}

		pushResp, err := h.Logic.SnapPush(model.SnapPushRequest{
			Name:              name,
			UnscannedFileName: uploadResp.UploadID,
			BinaryFileSize:    uploadResp.Size,
			Series:            "20",
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("provisioning: snap push failed at %d: %v", i, err),
			})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		ticker := time.NewTicker(1 * time.Second)
		processed := false
		for !processed {
			select {
			case <-ctx.Done():
				ticker.Stop()
				cancel()
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("provisioning: timed out waiting for snap %s", name),
				})
				return
			case <-ticker.C:
				status, err := h.Logic.GetUploadStatus(pushResp.UploadID)
				if err == nil && status.Processed {
					processed = true
				}
			}
		}
		ticker.Stop()
		cancel()

		snapNames = append(snapNames, name)
		logrus.Infof("provisioned base revision for %s (%d/%d)", name, i+1, total)
	}

	var nameIdx int64
	h.performOperation(c, func() SnapOperation {
		idx := atomic.AddInt64(&nameIdx, 1) - 1
		name := snapNames[idx%int64(len(snapNames))]

		return func() error {
			sha, err := util.ComputeSHA3_384(req.DeltaFilePath)
			if err != nil {
				return err
			}

			rc, fileName, err := util.DeltaFileReader(req.DeltaFilePath)
			if err != nil {
				return err
			}
			defer rc.Close()

			stop := monitoring.StartMonitoringTimer("delta_unscanned_upload")
			uploadResp, err := h.Logic.UnscannedUpload(rc, fileName)
			stop()
			if err != nil {
				return fmt.Errorf("unscanned upload of delta: %w", err)
			}

			stop = monitoring.StartMonitoringTimer("delta_push")
			_, err = h.Logic.DeltaPush(req, name, uploadResp.UploadID, sha)
			stop()
			return err
		}
	})
}

func (h *Handler) DeltaDownload(c *gin.Context) {
	var req model.DeltaDownloadRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	total, _, concurrentParse := setupQueryParams(c)
	stamp := time.Now().Format("2006-01-02_15-04-05")

	err := monitoring.InitFileRecorder(
		fmt.Sprintf("/var/log/request_duration_%s_%s_%d_%d_%s.csv",
			"delta-download", stamp, concurrentParse, total, req.SnapName),
		5*time.Second,
		10000,
	)

	if err != nil {
		logrus.Errorf("failed to initialize file recorder: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize monitoring"})
		return
	}
	defer monitoring.ShutdownFileRecorder()

	h.performOperation(c, func() SnapOperation {
		return func() error {
			stop := monitoring.StartMonitoringTimer("delta_download")
			err := h.Logic.DeltaDownload(req.SnapName, req.DeltaFormat, req.DeltaName)
			stop()
			return err
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

	logrus.Debug("Waiting for all operations to finish...")
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
