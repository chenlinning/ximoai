package admin

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

type BatchTestAccountRequest struct {
	AccountIDs []int64 `json:"account_ids"`
	ModelID    string  `json:"model_id"`
	Prompt     string  `json:"prompt"`
	Mode       string  `json:"mode"`
	TestType   string  `json:"test_type"`
	Seconds    int     `json:"seconds"`
	Size       string  `json:"size"`
}

type BatchTestAccountResult struct {
	AccountID int64  `json:"account_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

// BatchTest handles batch account connection tests.
// POST /api/v1/admin/accounts/batch-test
func (h *AccountHandler) BatchTest(c *gin.Context) {
	if h.accountTestService == nil {
		response.InternalError(c, "Account test service unavailable")
		return
	}

	var req BatchTestAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if len(req.AccountIDs) == 0 {
		response.BadRequest(c, "account_ids is required")
		return
	}

	ctx := c.Request.Context()
	results := make([]BatchTestAccountResult, len(req.AccountIDs))

	const maxConcurrency = 2
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)

	for index, id := range req.AccountIDs {
		i, accountID := index, id
		g.Go(func() error {
			results[i] = h.runSingleBatchAccountTest(gctx, accountID, req)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	successCount := 0
	errorItems := make([]gin.H, 0)
	for _, item := range results {
		if item.Success {
			successCount++
			continue
		}
		errorItems = append(errorItems, gin.H{
			"account_id": item.AccountID,
			"error":      item.Error,
		})
	}

	response.Success(c, gin.H{
		"total":   len(req.AccountIDs),
		"success": successCount,
		"failed":  len(req.AccountIDs) - successCount,
		"results": results,
		"errors":  errorItems,
	})
}

func (h *AccountHandler) runSingleBatchAccountTest(ctx context.Context, accountID int64, req BatchTestAccountRequest) BatchTestAccountResult {
	result := BatchTestAccountResult{AccountID: accountID}
	testResult, err := h.accountTestService.RunTestBackgroundWithOptions(ctx, accountID, req.ModelID, service.AccountConnectionTestOptions{
		Prompt:   req.Prompt,
		Mode:     req.Mode,
		TestType: req.TestType,
		Seconds:  req.Seconds,
		Size:     req.Size,
	})
	if testResult != nil {
		result.LatencyMs = testResult.LatencyMs
	}

	if err == nil && testResult != nil && testResult.Status == "success" {
		if err := h.markBatchTestSuccess(ctx, accountID); err != nil {
			result.Error = fmt.Sprintf("test passed but failed to update account status: %s", err.Error())
			return result
		}
		result.Success = true
		return result
	}

	if testResult != nil && testResult.ErrorMessage != "" {
		result.Error = testResult.ErrorMessage
	} else if err != nil {
		result.Error = err.Error()
	} else {
		result.Error = "test failed"
	}
	if err := h.markBatchTestFailure(ctx, accountID, result.Error); err != nil {
		result.Error = fmt.Sprintf("%s; failed to update account status: %s", result.Error, err.Error())
	}
	return result
}

func (h *AccountHandler) markBatchTestSuccess(ctx context.Context, accountID int64) error {
	if h.rateLimitService != nil {
		if _, err := h.rateLimitService.RecoverAccountAfterSuccessfulTest(ctx, accountID); err != nil {
			return err
		}
		return nil
	}

	if h.adminService != nil {
		if _, err := h.adminService.ClearAccountError(ctx, accountID); err != nil {
			return err
		}
	}
	return nil
}

func (h *AccountHandler) markBatchTestFailure(ctx context.Context, accountID int64, errorMsg string) error {
	if h.adminService == nil {
		return nil
	}
	if err := h.adminService.SetAccountError(ctx, accountID, errorMsg); err != nil {
		return err
	}
	return nil
}
