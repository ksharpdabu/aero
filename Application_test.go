package aero_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/aerogo/aero"
	"github.com/aerogo/http/client"
	"github.com/stretchr/testify/assert"
)

const helloWorld = "Hello World"

func TestApplicationGet(t *testing.T) {
	app := aero.New()

	// Register route
	app.Get("/", func(ctx *aero.Context) string {
		return ctx.Text(helloWorld)
	})

	// Get response
	response := getResponse(app, "/")

	// Verify response
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, helloWorld, response.Body.String())
}

func TestApplicationPost(t *testing.T) {
	app := aero.New()

	// Register route
	app.Post("/", func(ctx *aero.Context) string {
		return ctx.Text(helloWorld)
	})

	// Get response
	request, _ := http.NewRequest("POST", "/", nil)
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)

	// Verify response
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, helloWorld, response.Body.String())
}

func TestApplicationRewrite(t *testing.T) {
	app := aero.New()

	// Register route
	app.Get("/hello", func(ctx *aero.Context) string {
		return ctx.Text(helloWorld)
	})

	// Rewrite route
	app.Rewrite(func(ctx *aero.RewriteContext) {
		if ctx.URI() == "/" {
			ctx.SetURI("/hello")
			return
		}
	})

	// Get response
	response := getResponse(app, "/")

	// Verify response
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, helloWorld, response.Body.String())
}

func TestApplicationLoadConfig(t *testing.T) {
	app := aero.New()
	workingDirectory, _ := os.Getwd()

	err := os.Chdir("testdata")
	assert.NoError(t, err)

	app.Load()

	err = os.Chdir(workingDirectory)
	assert.NoError(t, err)

	assert.Equal(t, "Test title", app.Config.Title)
}

func TestApplicationRun(t *testing.T) {
	app := aero.New()

	// When frontpage is requested, kill the server
	app.Get("/", func(ctx *aero.Context) string {
		return ctx.HTML(helloWorld)
	})

	// When the server is started, we request the frontpage
	app.OnStart(func() {
		_, err := client.Get(fmt.Sprintf("http://localhost:%d/", app.Config.Ports.HTTP)).End()
		assert.NoError(t, err)

		err = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		assert.NoError(t, err)
	})

	// When the server ends, check elapsed time
	app.OnEnd(func() {
		elapsed := time.Since(app.StartTime())
		assert.True(t, elapsed < 2*time.Second)
	})

	// Run
	app.Run()
}

func TestApplicationRunHTTPS(t *testing.T) {
	app := aero.New()
	app.Security.Load("testdata/fullchain.pem", "testdata/privkey.pem")

	// Register route
	app.Get("/", func(ctx *aero.Context) string {
		return ctx.HTML(helloWorld)
	})

	// When the server is started, we request the frontpage
	app.OnStart(func() {
		_, err := client.Get(fmt.Sprintf("http://localhost:%d/", app.Config.Ports.HTTP)).End()
		assert.NoError(t, err)

		_, err = client.Get(fmt.Sprintf("https://localhost:%d/", app.Config.Ports.HTTPS)).End()
		assert.NoError(t, err)

		go func() {
			err = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			assert.NoError(t, err)
		}()
	})

	// Run
	app.Run()
}

func TestApplicationUnavailablePort(t *testing.T) {
	defer func() {
		_ = recover()
		// assert.NotNil(t, r)
		// assert.Contains(t, r.(error).Error(), "bind: permission denied")
	}()

	app := aero.New()
	app.Config.Ports.HTTP = 80
	app.Config.Ports.HTTPS = 443
	app.ListenAndServe()
}

// getResponse sends a request to the server and returns the response.
func getResponse(app *aero.Application, route string) *httptest.ResponseRecorder {
	// Create request
	request, _ := http.NewRequest("GET", route, nil)
	request.Header.Set("Accept-Encoding", "gzip")

	// Get response
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, request)

	return response
}
