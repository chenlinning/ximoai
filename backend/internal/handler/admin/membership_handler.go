package admin

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type MembershipHandler struct {
	membershipService *service.MembershipService
}

func NewMembershipHandler(membershipService *service.MembershipService) *MembershipHandler {
	return &MembershipHandler{membershipService: membershipService}
}

type membershipLevelRequest struct {
	Name         *string  `json:"name"`
	Code         *string  `json:"code"`
	DiscountRate *float64 `json:"discount_rate"`
	Enabled      *bool    `json:"enabled"`
	IsDefault    *bool    `json:"is_default"`
	SortOrder    *int     `json:"sort_order"`
	Description  *string  `json:"description"`
	GroupIDs     []int64  `json:"group_ids"`
}

type assignMembershipRequest struct {
	MembershipLevelID int64   `json:"membership_level_id" binding:"required"`
	ExpiresAt         *string `json:"expires_at"`
	Source            string  `json:"source"`
}

func (h *MembershipHandler) List(c *gin.Context) {
	includeDisabled := c.Query("include_disabled") == "1" || c.Query("include_disabled") == "true"
	levels, err := h.membershipService.ListLevels(c.Request.Context(), includeDisabled)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, levels)
}

func (h *MembershipHandler) Get(c *gin.Context) {
	id, ok := parseMembershipID(c)
	if !ok {
		return
	}
	level, err := h.membershipService.GetLevel(c.Request.Context(), id)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, level)
}

func (h *MembershipHandler) Create(c *gin.Context) {
	var req membershipLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	input := membershipLevelInputFromRequest(req, nil)
	level, err := h.membershipService.CreateLevel(c.Request.Context(), input)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Created(c, level)
}

func (h *MembershipHandler) Update(c *gin.Context) {
	id, ok := parseMembershipID(c)
	if !ok {
		return
	}
	existing, err := h.membershipService.GetLevel(c.Request.Context(), id)
	if response.ErrorFrom(c, err) {
		return
	}
	var req membershipLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	input := membershipLevelInputFromRequest(req, existing)
	level, err := h.membershipService.UpdateLevel(c.Request.Context(), id, input)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, level)
}

func (h *MembershipHandler) Delete(c *gin.Context) {
	id, ok := parseMembershipID(c)
	if !ok {
		return
	}
	err := h.membershipService.DisableLevel(c.Request.Context(), id)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, gin.H{"message": "membership level disabled"})
}

func (h *MembershipHandler) Sync(c *gin.Context) {
	id, ok := parseMembershipID(c)
	if !ok {
		return
	}
	err := h.membershipService.SyncMembershipLevel(c.Request.Context(), id)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, gin.H{"message": "membership level synced"})
}

func (h *MembershipHandler) AssignUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		response.BadRequest(c, "invalid user id")
		return
	}
	var req assignMembershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			response.BadRequest(c, "invalid expires_at")
			return
		}
		expiresAt = &parsed
	}
	summary, err := h.membershipService.AssignMembership(c.Request.Context(), service.AssignMembershipInput{
		UserID:    userID,
		LevelID:   req.MembershipLevelID,
		ExpiresAt: expiresAt,
		Source:    req.Source,
	})
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, summary)
}

func parseMembershipID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid membership id")
		return 0, false
	}
	return id, true
}

func membershipLevelInputFromRequest(req membershipLevelRequest, existing *service.MembershipLevel) service.MembershipLevelInput {
	input := service.MembershipLevelInput{
		Enabled: true,
	}
	if existing != nil {
		input = service.MembershipLevelInput{
			Name:         existing.Name,
			Code:         existing.Code,
			DiscountRate: existing.DiscountRate,
			Enabled:      existing.Enabled,
			IsDefault:    existing.IsDefault,
			SortOrder:    existing.SortOrder,
			Description:  existing.Description,
			GroupIDs:     make([]int64, 0, len(existing.Groups)),
		}
		for _, group := range existing.Groups {
			input.GroupIDs = append(input.GroupIDs, group.ID)
		}
	}
	if req.Name != nil {
		input.Name = *req.Name
	}
	if req.Code != nil {
		input.Code = *req.Code
	}
	if req.DiscountRate != nil {
		input.DiscountRate = *req.DiscountRate
	}
	if req.Enabled != nil {
		input.Enabled = *req.Enabled
	}
	if req.IsDefault != nil {
		input.IsDefault = *req.IsDefault
	}
	if req.SortOrder != nil {
		input.SortOrder = *req.SortOrder
	}
	if req.Description != nil {
		input.Description = *req.Description
	}
	if req.GroupIDs != nil {
		input.GroupIDs = req.GroupIDs
	}
	return input
}
