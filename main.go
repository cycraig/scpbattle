package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/jpillora/ipfilter"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"github.com/cycraig/scpbattle/db"
	"github.com/cycraig/scpbattle/handler"
	"github.com/cycraig/scpbattle/model"
	"github.com/cycraig/scpbattle/store"
)

// TemplateRegistry holds a map of named HTML templates.
type TemplateRegistry struct {
	templates map[string]*template.Template
}

// Render applies the specified named HTML template with the given data.
func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.templates[name]
	if !ok {
		err := errors.New("Template not found: " + name)
		fmt.Println("Template not found: " + name)
		return err
	}
	return tmpl.ExecuteTemplate(w, "base", data)
}

// Clacks "Do you not know that a man is not dead while his name is still spoken?"
// Adds the X-Clacks-Overhead header to HTTP responses.
func Clacks(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Add("X-Clacks-Overhead", "GNU Terry Pratchett")
		err := next(c)
		return err
	}
}

// CacheControlHeaders middleware adds the Cache-Control header when serving certain static files in Echo.
func CacheControlHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	shortCacheMaxAge := 86400   // 1 day
	longCacheMaxAge := 31536000 // 1 year
	shortCacheableExts := []string{".ico", ".jpg", ".jpeg", ".png", ".svg", ".css"}
	longCacheableExts := []string{".eot", ".ttf", ".woff", ".woff2"}
	return func(c echo.Context) error {
		uri := c.Request().RequestURI
		cacheMaxAge := 0
		// TODO: it might be faster to traverse a trie (containing reversed extensions)
		// or hashmaps of all the extensions of the same length.
		// With such few extensions it's fine for now, need to do benchmarks to see if it's worthwhile.
		for _, ext := range shortCacheableExts {
			if strings.HasSuffix(uri, ext) {
				cacheMaxAge = shortCacheMaxAge
				break
			}
		}
		if cacheMaxAge == 0 {
			for _, ext := range longCacheableExts {
				if strings.HasSuffix(uri, ext) {
					cacheMaxAge = longCacheMaxAge
					break
				}
			}
		}
		if cacheMaxAge != 0 {
			c.Response().Header().Add("Cache-Control", fmt.Sprintf("max-age=%d", cacheMaxAge))
		}
		err := next(c)
		return err
	}
}

// GzipSkipper for the Echo Gzip middleware skips compressing common image files.
func GzipSkipper(c echo.Context) bool {
	uri := c.Request().RequestURI
	excludeExts := []string{".ico", ".jpg", ".jpeg", ".png"}
	for _, ext := range excludeExts {
		if strings.HasSuffix(uri, ext) {
			return true
		}
	}
	return false
}

func filterIP(next echo.HandlerFunc) echo.HandlerFunc {
	// TODO: implement a radix tree / ART to perform CIDR lookups...
	filter := ipfilter.New(ipfilter.Options{
		BlockedCountries: []string{"CN"},
		BlockedIPs:       []string{"146.141.0.0/16"},
		BlockByDefault:   false,
	})
	return func(c echo.Context) error {
		ipAddr := c.RealIP()
		if !filter.Allowed(ipAddr) {
			return echo.NewHTTPError(http.StatusForbidden,
				fmt.Sprintf("Blocked IP address %s", ipAddr))
		}
		err := next(c)
		return err
	}
}

func main() {
	// Echo instance
	e := echo.New()

	e.Debug = true
	e.IPExtractor = echo.ExtractIPFromXFFHeader() // Heroku uses x-forwarded-for

	// Using a map of template files instead of parseGlob because template
	// definitions overwrite each other, e.g. body will be overwritten by
	// the html template parsed last.
	templates := make(map[string]*template.Template)
	// renderer := &TemplateRenderer{
	// 	templates: template.Must(template.ParseGlob(path.Join("view", "*.html"))),
	// }
	templates["vote.html"] = template.Must(template.ParseFiles(path.Join("view", "vote.html"), path.Join("view", "base.html")))
	templates["rankings.html"] = template.Must(template.ParseFiles(path.Join("view", "rankings.html"), path.Join("view", "base.html")))
	templates["error.html"] = template.Must(template.ParseFiles(path.Join("view", "error.html"), path.Join("view", "base.html")))
	templates["about.html"] = template.Must(template.ParseFiles(path.Join("view", "about.html"), path.Join("view", "base.html")))
	e.Renderer = &TemplateRegistry{
		templates: templates,
	}
	e.HTTPErrorHandler = handler.HTTPErrorHandler
	e.Logger.SetLevel(log.DEBUG)

	// Middleware
	e.Pre(middleware.Logger())
	e.Pre(middleware.Recover())
	e.Pre(middleware.RemoveTrailingSlash())
	e.Pre(filterIP)
	e.Use(middleware.BodyLimit("1M"))
	e.Use(Clacks)
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Skipper: GzipSkipper,
		Level:   5,
	}))
	e.Use(CacheControlHeaders)
	e.Use(middleware.Static("static"))

	// Initialise database
	dbURL := os.Getenv("DATABASE_URL")
	var d *gorm.DB
	if dbURL == "" {
		d = db.NewDB("sqlite3", "data.db", true)
	} else {
		d = db.NewDB("postgres", dbURL, false)
	}
	defer d.Close()
	scpCache := store.NewSCPCache(store.NewSCPStore(d))
	h := handler.NewHandler(scpCache, "images/")

	// Populate example data
	// TODO: replace this
	scpCache.Create(model.NewSCP("SCP-049", "The Plague Doctor", "scp_049.jpg", "http://www.scp-wiki.net/scp-049"))
	scpCache.Create(model.NewSCP("SCP-055", "Not a Sphere?", "scp_055.jpg", "http://www.scp-wiki.net/scp-055"))
	scpCache.Create(model.NewSCP("SCP-087", "The Stairwell", "scp_087.jpg", "http://www.scp-wiki.net/scp-087"))
	scpCache.Create(model.NewSCP("SCP-093", "Red Sea Object", "scp_093.jpg", "http://www.scp-wiki.net/scp-093"))
	scpCache.Create(model.NewSCP("SCP-096", "The Shy Guy", "scp_096.jpg", "http://www.scp-wiki.net/scp-096"))
	scpCache.Create(model.NewSCP("SCP-106", "The Old Man", "scp_106.jpg", "http://www.scp-wiki.net/scp-106"))
	scpCache.Create(model.NewSCP("SCP-173", "The Sculpture", "scp_173.jpg", "http://www.scp-wiki.net/scp-173"))
	scpCache.Create(model.NewSCP("SCP-610", "The Flesh that Hates", "scp_610.jpg", "http://www.scp-wiki.net/scp-610"))
	scpCache.Create(model.NewSCP("SCP-682", "The Hard-To-Destroy Reptile", "scp_682.jpg", "http://www.scp-wiki.net/scp-682"))
	scpCache.Create(model.NewSCP("SCP-939", "With Many Voices", "scp_939.jpg", "http://www.scp-wiki.net/scp-939"))
	scpCache.Create(model.NewSCP("●●|●●●●●|●●|●", "", "scp_2521.jpg", "http://www.scp-wiki.net/scp-2521"))
	scpCache.Create(model.NewSCP("SCP-3000", "Anantashesha", "scp_3000.jpg", "http://www.scp-wiki.net/scp-3000"))
	scpCache.Create(model.NewSCP("SCP-3001", "Red Reality", "scp_3001.jpg", "http://www.scp-wiki.net/scp-3001"))
	scpCache.Create(model.NewSCP("SCP-____-J", "Procrastinati", "scp_j.jpg", "http://www.scp-wiki.net/scp-j"))

	// Routes
	e.GET("/", h.VotePageHandler)
	e.POST("/vote", h.VoteHandler)
	e.GET("/health", h.HealthCheckHandler)
	e.GET("/rankings", h.RankingsPageHandler)
	e.GET("/about", h.AboutPageHandler)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		// local development
		port = "1323"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
