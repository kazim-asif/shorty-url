package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "shorty-url/routers"
	"shorty-url/controllers"
	"shorty-url/models"
	"shorty-url/utils"

	beego "github.com/beego/beego/v2/server/web"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	_, file, _, _ := runtime.Caller(0)
	apppath, _ := filepath.Abs(filepath.Dir(filepath.Join(file, ".."+string(filepath.Separator))))
	beego.TestBeegoInit(apppath)
}

func TestURLShortening(t *testing.T) {
	Convey("URL Shortening Tests", t, func() {
		// Clean up any existing data
		models.URLs = make(map[string]*models.URL)
		Convey("Should create short URL successfully", func() {
			reqBody := controllers.ShortenRequest{
				URL: "https://example.com",
			}
			body, _ := json.Marshal(reqBody)

			r, _ := http.NewRequest("POST", "/api/v1/urls/", bytes.NewBuffer(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			beego.BeeApp.Handlers.ServeHTTP(w, r)

			So(w.Code, ShouldEqual, 201)

			var response controllers.ShortenResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response.ShortCode, ShouldNotBeEmpty)
			So(response.OriginalURL, ShouldEqual, "https://example.com")
			So(strings.Contains(response.ShortURL, response.ShortCode), ShouldBeTrue)
		})

		Convey("Should reject invalid URLs", func() {
			reqBody := controllers.ShortenRequest{
				URL: "invalid-url",
			}
			body, _ := json.Marshal(reqBody)

			r, _ := http.NewRequest("POST", "/api/v1/urls/", bytes.NewBuffer(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			beego.BeeApp.Handlers.ServeHTTP(w, r)

			So(w.Code, ShouldEqual, 400)

			var response controllers.ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			So(err, ShouldBeNil)
			So(response.Error, ShouldContainSubstring, "Invalid URL")
		})

		Convey("Should reject empty URL", func() {
			reqBody := controllers.ShortenRequest{
				URL: "",
			}
			body, _ := json.Marshal(reqBody)

			r, _ := http.NewRequest("POST", "/api/v1/urls/", bytes.NewBuffer(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			beego.BeeApp.Handlers.ServeHTTP(w, r)

			So(w.Code, ShouldEqual, 400)
		})
	})
}

func TestURLRedirection(t *testing.T) {
	Convey("URL Redirection Tests", t, func() {
		// Clean up any existing data
		models.URLs = make(map[string]*models.URL)

		url, err := models.CreateURL("https://example.com", "test123", "TestAgent", "127.0.0.1")
		So(err, ShouldBeNil)
		So(url, ShouldNotBeNil)

		Convey("Should redirect to original URL", func() {
			r, _ := http.NewRequest("GET", "/test123", nil)
			w := httptest.NewRecorder()
			beego.BeeApp.Handlers.ServeHTTP(w, r)

			So(w.Code, ShouldEqual, 302)
			So(w.Header().Get("Location"), ShouldEqual, "https://example.com")
		})

		Convey("Should return 404 for non-existent short code", func() {
			r, _ := http.NewRequest("GET", "/nonexistent", nil)
			w := httptest.NewRecorder()
			beego.BeeApp.Handlers.ServeHTTP(w, r)

			So(w.Code, ShouldEqual, 404)
		})

		Convey("Should increment click count", func() {
			initialClicks := url.Clicks

			r, _ := http.NewRequest("GET", "/test123", nil)
			w := httptest.NewRecorder()
			beego.BeeApp.Handlers.ServeHTTP(w, r)

			updatedURL, err := models.GetURLByShortCode("test123")
			So(err, ShouldBeNil)
			So(updatedURL.Clicks, ShouldEqual, initialClicks+1)
		})
	})
}

func TestURLStats(t *testing.T) {
	Convey("URL Statistics Tests", t, func() {
		// Clean up any existing data
		models.URLs = make(map[string]*models.URL)
		models.CreateURL("https://example.com", "stats123", "TestAgent", "127.0.0.1")

		Convey("Should return URL stats", func() {
			r, _ := http.NewRequest("GET", "/stats123/stats", nil)
			w := httptest.NewRecorder()
			beego.BeeApp.Handlers.ServeHTTP(w, r)

			So(w.Code, ShouldEqual, 200)

			var url models.URL
			err := json.Unmarshal(w.Body.Bytes(), &url)
			So(err, ShouldBeNil)
			So(url.ShortCode, ShouldEqual, "stats123")
			So(url.OriginalURL, ShouldEqual, "https://example.com")
		})

		Convey("Should return analytics", func() {
			r, _ := http.NewRequest("GET", "/stats123/analytics", nil)
			w := httptest.NewRecorder()
			beego.BeeApp.Handlers.ServeHTTP(w, r)

			So(w.Code, ShouldEqual, 200)

			var analytics models.Analytics
			err := json.Unmarshal(w.Body.Bytes(), &analytics)
			So(err, ShouldBeNil)
			So(analytics.TotalClicks, ShouldBeGreaterThanOrEqualTo, 0)
		})
	})
}

func TestURLValidation(t *testing.T) {
	Convey("URL Validation Tests", t, func() {
		Convey("Should validate correct URLs", func() {
			validURLs := []string{
				"https://example.com",
				"http://example.com",
				"https://subdomain.example.com",
				"https://example.com/path?query=value",
			}

			for _, url := range validURLs {
				_, err := utils.ValidateURL(url)
				So(err, ShouldBeNil)
			}
		})

		Convey("Should reject invalid URLs", func() {
			invalidURLs := []string{
				"invalid-url",
				"ftp://example.com",
				"example.com",
				"",
				"javascript:alert(1)",
			}

			for _, url := range invalidURLs {
				_, err := utils.ValidateURL(url)
				So(err, ShouldNotBeNil)
			}
		})
	})
}

func TestShortCodeGeneration(t *testing.T) {
	Convey("Short Code Generation Tests", t, func() {
		shortener := utils.NewShortener()

		Convey("Should generate codes of correct length", func() {
			code, err := shortener.GenerateShortCode()
			So(err, ShouldBeNil)
			So(len(code), ShouldEqual, utils.DefaultLength)
		})

		Convey("Should generate different codes", func() {
			codes := make(map[string]bool)
			for i := 0; i < 100; i++ {
				code, err := shortener.GenerateShortCode()
				So(err, ShouldBeNil)
				So(codes[code], ShouldBeFalse)
				codes[code] = true
			}
		})

		Convey("Should respect length limits", func() {
			shortener.SetLength(2)
			So(shortener.Length, ShouldEqual, utils.MinLength)

			shortener.SetLength(15)
			So(shortener.Length, ShouldEqual, utils.MaxLength)

			shortener.SetLength(7)
			So(shortener.Length, ShouldEqual, 7)
		})
	})
}

func TestAnalytics(t *testing.T) {
	Convey("Analytics Tests", t, func() {
		// Clean up any existing data
		models.URLs = make(map[string]*models.URL)
		models.ClearAnalyticsData()

		shortCode := "analytics123"
		models.CreateURL("https://example.com", shortCode, "TestAgent", "127.0.0.1")

		Convey("Should track click data", func() {
			models.LogClick(shortCode, "Chrome", "192.168.1.1", "https://google.com")
			models.LogClick(shortCode, "Firefox", "192.168.1.2", "https://bing.com")

			analytics := models.GetAnalytics(shortCode)
			So(analytics.TotalClicks, ShouldEqual, 2)
			So(analytics.UniqueClicks, ShouldEqual, 2)
			So(len(analytics.TopReferrers), ShouldBeGreaterThan, 0)
			So(len(analytics.TopUserAgents), ShouldBeGreaterThan, 0)
		})

		Convey("Should calculate unique clicks correctly", func() {
			// Use a different short code to avoid interference
			shortCode2 := "unique123"
			models.CreateURL("https://example.com", shortCode2, "TestAgent", "127.0.0.1")

			models.LogClick(shortCode2, "Chrome", "192.168.1.3", "")
			models.LogClick(shortCode2, "Chrome", "192.168.1.3", "")

			analytics := models.GetAnalytics(shortCode2)
			So(analytics.UniqueClicks, ShouldEqual, 1)
		})
	})
}

