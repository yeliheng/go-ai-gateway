package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/yeliheng/go-ai-gateway/common/config"
	"github.com/yeliheng/go-ai-gateway/common/logger"

	"go.uber.org/zap"
)

func main() {

	logger.InitLogger()
	defer logger.Log.Sync()

	if err := config.LoadConfig(); err != nil {
		logger.Log.Fatal("Failed to load config", zap.Error(err))
	}

	port := config.GlobalConfig.Services.Gateway.Port
	if port == "" {
		logger.Log.Fatal("Gateway port is required but missing")
	}

	bizAddr := config.GlobalConfig.Services.Biz.Addr
	if bizAddr == "" {
		logger.Log.Fatal("Biz Service Address is required but missing")
	}

	targetUrl, err := url.Parse(bizAddr)
	if err != nil {
		logger.Log.Fatal("Invalid Biz Address", zap.Error(err))
	}

	proxy := httputil.NewSingleHostReverseProxy(targetUrl)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetUrl.Host
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Log.Info("Proxying request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
		)
		proxy.ServeHTTP(w, r)
	})

	logger.Log.Info("Gateway listening", zap.String("port", port))
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Log.Fatal("Gateway failed", zap.Error(err))
	}
}
