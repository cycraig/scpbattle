package main

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"github.com/cycraig/scpbattle/db"
	"github.com/cycraig/scpbattle/handler"
	"github.com/cycraig/scpbattle/model"
	"github.com/cycraig/scpbattle/store"
)

type TemplateRegistry struct {
	templates map[string]*template.Template
}

func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.templates[name]
	if !ok {
		err := errors.New("Template not found: " + name)
		fmt.Println("Template not found: " + name)
		return err
	}
	return tmpl.ExecuteTemplate(w, "base", data)
}

func Clacks(next echo.HandlerFunc) echo.HandlerFunc {
	// "Do you not know that a man is not dead while his name is still spoken?"
	return func(c echo.Context) error {
		c.Response().Header().Add("X-Clacks-Overhead", "GNU Terry Pratchett")
		err := next(c)
		return err
	}
}

func filterIP(next echo.HandlerFunc) echo.HandlerFunc {
	// TODO: expand on this
	return func(c echo.Context) error {
		// Block requests from localhost as a test.
		// ipAddr := c.RealIP()
		// fmt.Println("Connection from: " + ipAddr)
		// if ipAddr == "127.0.0.1" || ipAddr == "::1" {
		// 	//c.Response().WriteHeader(http.StatusUnauthorized)
		// 	return echo.NewHTTPError(http.StatusUnauthorized,
		// 		fmt.Sprintf("IP address %s not allowed", ipAddr))
		// }
		err := next(c)
		return err
	}
}

func main() {
	// Echo instance
	e := echo.New()

	e.Debug = true

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
	e.Renderer = &TemplateRegistry{
		templates: templates,
	}
	e.HTTPErrorHandler = handler.HTTPErrorHandler
	e.Logger.SetLevel(log.DEBUG)

	// Middleware
	e.Pre(middleware.Logger())
	e.Pre(middleware.Recover())
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Static("static"))
	e.Pre(filterIP)
	e.Use(middleware.BodyLimit("1M"))
	e.Use(Clacks)
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))

	// Initialise database
	d := db.NewDB("data.db")
	defer d.Close()
	scpCache := store.NewSCPCache(store.NewSCPStore(d))
	h := handler.NewHandler(scpCache)

	// Populate example data
	// TODO: replace this
	scpCache.Create(model.NewSCP("SCP-049", "The Plague Doctor", "scp_049.jpg", "http://www.scp-wiki.net/scp-049"))
	scpCache.Create(model.NewSCP("SCP-096", "The Shy Guy", "scp_096.jpg", "http://www.scp-wiki.net/scp-096"))
	scpCache.Create(model.NewSCP("SCP-106", "The Old Man", "scp_106.jpg", "http://www.scp-wiki.net/scp-106"))
	scpCache.Create(model.NewSCP("SCP-173", "The Sculpture", "scp_173.jpg", "http://www.scp-wiki.net/scp-173"))
	scpCache.Create(model.NewSCP("SCP-682", "The Hard-To-Destroy Reptile", "scp_682.jpg", "http://www.scp-wiki.net/scp-682"))
	scpCache.Create(model.NewSCP("SCP-939", "With Many Voices", "scp_939.jpg", "http://www.scp-wiki.net/scp-939"))
	scpCache.Create(model.NewSCP("●●|●●●●●|●●|●", "", "scp_2521.jpg", "http://www.scp-wiki.net/scp-2521"))

	// Routes
	e.GET("/", h.VotePageHandler)
	e.POST("/vote", h.VoteHandler)
	e.GET("/health", h.HealthCheckHandler)
	e.GET("/rankings", h.RankingsPageHandler)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		// local development
		port = "1323"
	}
	e.Logger.Fatal(e.Start(":" + port))
}
