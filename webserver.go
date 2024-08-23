package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type ErrorHandler func(w http.ResponseWriter, r *http.Request) error

func errH(f ErrorHandler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err != nil {
			logrus.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "error: %s", err)
			return
		}
	})
}

func startWs(config *Config, ctx context.Context) {
	handler := http.NewServeMux()

	handler.HandleFunc("/api/hosts/{host}", errH(func(w http.ResponseWriter, r *http.Request) error {
		if r.Method != http.MethodDelete {
			return fmt.Errorf("only DELETE is supported")
		}
		host := r.PathValue("host")
		mutex.Lock()
		defer mutex.Unlock()

		files, err := fs.Glob(os.DirFS(config.FileSdPath), "*.json")
		if err != nil {
			return err
		}

		for _, file := range files {
			path := filepath.Join(config.FileSdPath, file)
			groups, err := getOldGroups(path)
			if err != nil {
				return err
			}

			newGroups := slices.DeleteFunc(groups, func(dp Group) bool {
				return dp.Labels["host"] == host
			})

			if len(groups) == len(newGroups) {
				continue
			}

			file, err := os.Create(path)
			if err != nil {
				return err
			}
			defer file.Close()

			encoder := json.NewEncoder(file)
			encoder.SetIndent("", "\t")
			err = encoder.Encode(newGroups)
			if err != nil {
				return err
			}
		}

		return nil

	}))
	handler.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 20 * time.Second,
		Addr:              ":" + config.Port,
		Handler:           handler,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.Fatalf("error starting webserver %s", err)
		}
	}()

	logrus.Debug("webserver started")

	<-ctx.Done()

	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != "" {
		logrus.Debug("sleeping 5 sec before shutdown") // to give k8s ingresses time to sync
		time.Sleep(5 * time.Second)
	}
	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutDown); !errors.Is(err, http.ErrServerClosed) && err != nil {
		logrus.Error(err)
	}
}
