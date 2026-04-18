package controllers

import (
	"encoding/json"
	"net/http"
	"shorty-url/models"
	"shorty-url/utils"
	"strings"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/core/logs"
)

type URLController struct {
	beego.Controller
}

type ShortenRequest struct {
	URL string `json:"url" valid:"required,url"`
}

type ShortenResponse struct {
	ShortCode   string `json:"short_code"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// @Title Shorten URL
// @Description Create a shortened URL
// @Param	body	body	ShortenRequest	true	"URL to shorten"
// @Success 201 {object} ShortenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @router / [post]
func (u *URLController) Post() {
	var req ShortenRequest
	if err := json.Unmarshal(u.Ctx.Input.RequestBody, &req); err != nil {
		u.sendError(http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	if req.URL == "" {
		u.sendError(http.StatusBadRequest, "URL is required", "URL field cannot be empty")
		return
	}

	validatedURL, err := utils.ValidateURL(req.URL)
	if err != nil {
		u.sendError(http.StatusBadRequest, "Invalid URL", err.Error())
		return
	}

	shortener := utils.NewShortener()
	userAgent := u.Ctx.Input.Header("User-Agent")
	clientIP := u.getClientIP()

	var shortCode string
	var url *models.URL
	maxAttempts := 10

	for attempts := 0; attempts < maxAttempts; attempts++ {
		shortCode, err = shortener.GenerateShortCode()
		if err != nil {
			u.sendError(http.StatusInternalServerError, "Code generation failed", err.Error())
			return
		}

		url, err = models.CreateURL(validatedURL, shortCode, userAgent, clientIP)
		if err == nil {
			break
		}

		if !strings.Contains(err.Error(), "already exists") {
			u.sendError(http.StatusInternalServerError, "Database error", err.Error())
			return
		}

		shortener.SetLength(shortener.Length + 1)
	}

	if url == nil {
		u.sendError(http.StatusInternalServerError, "Failed to generate unique code", "Please try again")
		return
	}

	baseURL := u.getBaseURL()
	response := ShortenResponse{
		ShortCode:   shortCode,
		ShortURL:    baseURL + "/" + shortCode,
		OriginalURL: validatedURL,
	}

	u.Ctx.Output.SetStatus(http.StatusCreated)
	u.Data["json"] = response
	u.ServeJSON()
}

// @Title Redirect
// @Description Redirect to original URL
// @Param	shortCode	path	string	true	"Short code"
// @Success 302 {string} string "redirect"
// @Failure 404 {object} ErrorResponse
// @Failure 410 {object} ErrorResponse
// @router /:shortCode [get]
func (u *URLController) Get() {
	shortCode := u.Ctx.Input.Param(":shortCode")
	if shortCode == "" {
		u.sendError(http.StatusBadRequest, "Short code required", "Short code parameter is missing")
		return
	}

	url, err := models.GetURLByShortCode(shortCode)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			u.sendError(http.StatusNotFound, "URL not found", "The requested short URL does not exist")
		} else if strings.Contains(err.Error(), "expired") {
			u.sendError(http.StatusGone, "URL expired", "This short URL has expired")
		} else if strings.Contains(err.Error(), "deactivated") {
			u.sendError(http.StatusGone, "URL deactivated", "This short URL has been deactivated")
		} else {
			u.sendError(http.StatusInternalServerError, "Database error", err.Error())
		}
		return
	}

	if err := models.IncrementClicks(shortCode); err != nil {
		logs.Warning("Failed to increment clicks for", shortCode, ":", err)
	}

	userAgent := u.Ctx.Input.Header("User-Agent")
	clientIP := u.getClientIP()
	referer := u.Ctx.Input.Header("Referer")
	models.LogClick(shortCode, userAgent, clientIP, referer)

	u.Ctx.Redirect(http.StatusFound, url.OriginalURL)
}

// @Title Get URL Stats
// @Description Get statistics for a short URL
// @Param	shortCode	path	string	true	"Short code"
// @Success 200 {object} models.URL
// @Failure 404 {object} ErrorResponse
// @router /:shortCode/stats [get]
func (u *URLController) GetStats() {
	shortCode := u.Ctx.Input.Param(":shortCode")
	if shortCode == "" {
		u.sendError(http.StatusBadRequest, "Short code required", "Short code parameter is missing")
		return
	}

	url, err := models.GetURLByShortCode(shortCode)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			u.sendError(http.StatusNotFound, "URL not found", "The requested short URL does not exist")
		} else {
			u.sendError(http.StatusInternalServerError, "Database error", err.Error())
		}
		return
	}

	u.Data["json"] = url
	u.ServeJSON()
}

// @Title Get URL Analytics
// @Description Get detailed analytics for a short URL
// @Param	shortCode	path	string	true	"Short code"
// @Success 200 {object} models.Analytics
// @Failure 404 {object} ErrorResponse
// @router /:shortCode/analytics [get]
func (u *URLController) GetAnalytics() {
	shortCode := u.Ctx.Input.Param(":shortCode")
	if shortCode == "" {
		u.sendError(http.StatusBadRequest, "Short code required", "Short code parameter is missing")
		return
	}

	_, err := models.GetURLByShortCode(shortCode)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			u.sendError(http.StatusNotFound, "URL not found", "The requested short URL does not exist")
		} else {
			u.sendError(http.StatusInternalServerError, "Database error", err.Error())
		}
		return
	}

	analytics := models.GetAnalytics(shortCode)
	u.Data["json"] = analytics
	u.ServeJSON()
}

// @Title List URLs
// @Description Get all shortened URLs
// @Success 200 {object} map[string]models.URL
// @router /list [get]
func (u *URLController) GetAll() {
	urls := models.GetAllURLs()
	u.Data["json"] = urls
	u.ServeJSON()
}

// @Title Delete URL
// @Description Delete a shortened URL
// @Param	shortCode	path	string	true	"Short code"
// @Success 200 {string} string "deleted"
// @Failure 404 {object} ErrorResponse
// @router /:shortCode [delete]
func (u *URLController) Delete() {
	shortCode := u.Ctx.Input.Param(":shortCode")
	if shortCode == "" {
		u.sendError(http.StatusBadRequest, "Short code required", "Short code parameter is missing")
		return
	}

	err := models.DeleteURL(shortCode)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			u.sendError(http.StatusNotFound, "URL not found", "The requested short URL does not exist")
		} else {
			u.sendError(http.StatusInternalServerError, "Database error", err.Error())
		}
		return
	}

	u.Data["json"] = map[string]string{"message": "URL deleted successfully"}
	u.ServeJSON()
}

func (u *URLController) sendError(status int, error, message string) {
	u.Ctx.Output.SetStatus(status)
	u.Data["json"] = ErrorResponse{
		Error:   error,
		Code:    status,
		Message: message,
	}
	u.ServeJSON()
}

func (u *URLController) getClientIP() string {
	forwarded := u.Ctx.Input.Header("X-Forwarded-For")
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}

	realIP := u.Ctx.Input.Header("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	return u.Ctx.Input.IP()
}

func (u *URLController) getBaseURL() string {
	scheme := "http"
	if u.Ctx.Input.IsSecure() {
		scheme = "https"
	}
	return scheme + "://" + u.Ctx.Input.Host()
}

