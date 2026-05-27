package http

import (
	"encoding/json"
	"net/http"

	"proxy/internal/domain"
	"proxy/internal/usecases"
)

type IPFilterHandler struct {
	service usecases.IPFilterService
}

func NewIPFilterHandler(service usecases.IPFilterService) *IPFilterHandler {
	return &IPFilterHandler{service: service}
}

type AddIPRequest struct {
	Pattern string `json:"pattern" example:"192.168.1.0/24"`
	Comment string `json:"comment" example:"office subnet"`
}

// CheckIP godoc
// @Summary Check IP access
// @Tags ip-filter
// @Description Validate IP against allow/deny/grey lists
// @Produce json
// @Param ip query string true "IP address"
// @Success 200 {object} domain.IPCheckResult
// @Success 202 {object} domain.IPCheckResult
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /ip_access/check [get]
func (h *IPFilterHandler) CheckIP(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		writeError(w, http.StatusBadRequest, "ip is required")
		return
	}

	result, err := h.service.CheckIP(r.Context(), ip)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	status := http.StatusOK
	if !result.IsAllowed {
		status = http.StatusForbidden
	}

	if result.ListType == domain.GreyList {
		status = http.StatusAccepted
	}

	writeJSON(w, status, result)
}

// GetPolicy godoc
// @Summary Get IP policy
// @Tags ip-filter
// @Produce json
// @Success 200 {object} domain.IPAccessPolicy
// @Failure 500 {object} ErrorResponse
// @Router /ip_access/policy [get]
func (h *IPFilterHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	policy, err := h.service.GetPolicy(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

// ReloadPolicy godoc
// @Summary Reload policy
// @Tags ip-filter
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /ip_access/policy/reload [post]
func (h *IPFilterHandler) ReloadPolicy(w http.ResponseWriter, r *http.Request) {
	if err := h.service.ReloadPolicy(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "policy reloaded"})
}

// GetAllowList godoc
// @Summary Get allow list
// @Tags ip-filter
// @Produce json
// @Success 200 {array} domain.IPEntry
// @Failure 500 {object} ErrorResponse
// @Router /ip_access/allowlists [get]
func (h *IPFilterHandler) GetAllowList(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.GetAllowList(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// GetDenyList godoc
// @Summary Get deny list
// @Tags ip-filter
// @Produce json
// @Success 200 {array} domain.IPEntry
// @Failure 500 {object} ErrorResponse
// @Router /ip_access/denylists [get]
func (h *IPFilterHandler) GetDenyList(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.GetDenyList(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// GetGreyList godoc
// @Summary Get grey list
// @Tags ip-filter
// @Produce json
// @Success 200 {array} domain.IPEntry
// @Failure 500 {object} ErrorResponse
// @Router /ip_access/greylists [get]
func (h *IPFilterHandler) GetGreyList(w http.ResponseWriter, r *http.Request) {
	list, err := h.service.GetGreyList(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// AddToAllowList godoc
// @Summary Add to allow list
// @Tags ip-filter
// @Accept json
// @Produce json
// @Param request body AddIPRequest true "request"
// @Success 201 {object} domain.IPEntry
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /ip_access/allowlists [post]
func (h *IPFilterHandler) AddToAllowList(w http.ResponseWriter, r *http.Request) {
	h.addToList(w, r, domain.AllowList)
}

// AddToDenyList godoc
// @Summary Add to deny list
// @Tags ip-filter
// @Accept json
// @Produce json
// @Param request body AddIPRequest true "request"
// @Success 201 {object} domain.IPEntry
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /ip_access/denylists [post]
func (h *IPFilterHandler) AddToDenyList(w http.ResponseWriter, r *http.Request) {
	h.addToList(w, r, domain.DenyList)
}

// AddToGreyList godoc
// @Summary Add to grey list
// @Tags ip-filter
// @Accept json
// @Produce json
// @Param request body AddIPRequest true "request"
// @Success 201 {object} domain.IPEntry
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /ip_access/greylists [post]
func (h *IPFilterHandler) AddToGreyList(w http.ResponseWriter, r *http.Request) {
	h.addToList(w, r, domain.GreyList)
}

func (h *IPFilterHandler) addToList(w http.ResponseWriter, r *http.Request, list domain.IPListType) {
	var req AddIPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	var (
		entry *domain.IPEntry
		err   error
	)

	switch list {
	case domain.AllowList:
		entry, err = h.service.AddToAllowList(r.Context(), req.Pattern, req.Comment)
	case domain.DenyList:
		entry, err = h.service.AddToDenyList(r.Context(), req.Pattern, req.Comment)
	case domain.GreyList:
		entry, err = h.service.AddToGreyList(r.Context(), req.Pattern, req.Comment)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, entry)
}

// RemoveEntry godoc
// @Summary Remove entry
// @Tags ip-filter
// @Produce json
// @Param id path string true "entry id"
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /ip_access/{id} [delete]
func (h *IPFilterHandler) RemoveEntry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.service.RemoveEntry(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "removed"})
}

// GetStats godoc
// @Summary Get stats
// @Tags ip-filter
// @Produce json
// @Success 200 {object} domain.IPFilterStats
// @Failure 500 {object} ErrorResponse
// @Router /ip_access/stats [get]
func (h *IPFilterHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}
