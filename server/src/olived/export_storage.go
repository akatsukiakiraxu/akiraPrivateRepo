package main

import (
	"encoding/json"
	"log"
	"net/http"
	"olived/core"
	"sync"
	"time"

	"github.com/labstack/echo"
)

type exportToStorageStatus struct {
	IsRunning        bool    `json:"is_running"`
	Progress         float32 `json:"progress"`
	StorageInstalled bool    `json:"storage_installed"`
	LastError        *string `json:"last_error"`
}
type exportToStorageRequest struct {
	TargetFiles []string `json:"target_files"`
}

var status exportToStorageStatus
var statusLock sync.RWMutex
var eventServer *core.WSServer
var executeCh chan *exportToStorageRequest
var cancelCh chan struct{}
var installCh chan bool
var errorCh chan string

func RunExportStorageService(e *echo.Echo) error {
	eventServer = core.NewWSServer("/export/storage/event", func(_ *core.WSClient) {
		log.Println("/export/storage/event client connected.")
	})

	executeCh = make(chan *exportToStorageRequest)
	cancelCh = make(chan struct{})
	installCh = make(chan bool)
	errorCh = make(chan string)

	e.POST("/export/storage/execute", func(c echo.Context) error {
		request := new(exportToStorageRequest)
		if err := c.Bind(request); err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}
		statusLock.RLock()
		defer statusLock.RUnlock()
		if status.IsRunning {
			return c.String(http.StatusBadRequest, "Already running.")
		}
		if !status.StorageInstalled {
			return c.String(http.StatusBadRequest, "Storage not ready.")
		}
		if len(request.TargetFiles) == 0 {
			return c.String(http.StatusBadRequest, "No target files specified.")
		}
		executeCh <- request
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/export/storage/cancel", func(c echo.Context) error {
		statusLock.RLock()
		defer statusLock.Unlock()
		if !status.IsRunning {
			return c.String(http.StatusBadRequest, "Not running")
		}
		cancelCh <- struct{}{}
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/export/storage/status", func(c echo.Context) error {
		statusLock.RLock()
		defer statusLock.RUnlock()
		return c.JSON(http.StatusOK, status)
	})
	e.GET("/export/storage/test_install", func(c echo.Context) error {
		installCh <- true
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/export/storage/test_uninstall", func(c echo.Context) error {
		installCh <- false
		return c.String(http.StatusOK, "OK")
	})
	e.GET("/export/storage/test_error", func(c echo.Context) error {
		errorCh <- "error occured!"
		return c.String(http.StatusOK, "OK")
	})

	go func() {
		var files int
		var progress int
		ticker := time.NewTicker(time.Millisecond * 1000)
		for {
			select {
			case message := <-errorCh:
				statusLock.Lock()
				status.LastError = &message
				statusLock.Unlock()
				if event, err := json.Marshal(status); err == nil {
					eventServer.Write(string(event))
				}
			case install := <-installCh:
				statusLock.Lock()
				status.StorageInstalled = install
				statusLock.Unlock()
				if event, err := json.Marshal(status); err == nil {
					eventServer.Write(string(event))
				}
			case request := <-executeCh:
				if !status.IsRunning {
					statusLock.Lock()
					status.IsRunning = true
					status.Progress = 0
					status.LastError = nil
					statusLock.Unlock()
					files = len(request.TargetFiles)
					progress = 0
				}
			case <-cancelCh:
				if status.IsRunning {
					statusLock.Lock()
					status.IsRunning = false
					status.Progress = 0
					statusLock.Unlock()
				}
			case <-ticker.C:
				if status.IsRunning {
					progress++
					statusLock.Lock()
					status.Progress = float32(progress) / float32(files)
					statusLock.Unlock()
					if event, err := json.Marshal(status); err == nil {
						eventServer.Write(string(event))
					}
					if progress >= files {
						statusLock.Lock()
						status.IsRunning = false
						status.Progress = 0
						statusLock.Unlock()
					}
				}
			}
		}
	}()

	eventServer.Listen()

	return nil
}
