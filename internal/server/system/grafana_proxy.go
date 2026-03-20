// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package system

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	systembiz "github.com/ydcloud-dy/opshub/internal/biz/system"
	appLogger "github.com/ydcloud-dy/opshub/pkg/logger"
	"go.uber.org/zap"
)

// GrafanaProxyHandler Grafana 反向代理处理器
type GrafanaProxyHandler struct {
	configUseCase *systembiz.ConfigUseCase
}

// NewGrafanaProxyHandler 创建 Grafana 反向代理处理器
func NewGrafanaProxyHandler(configUseCase *systembiz.ConfigUseCase) *GrafanaProxyHandler {
	return &GrafanaProxyHandler{
		configUseCase: configUseCase,
	}
}

// RegisterProxyRoute 从配置读取 subpath，在根路由上注册代理路由
// 路由形如 /grafana_2syulinm/*proxyPath，与 Grafana sub_path 完全一致
// 这样 Grafana 页面中的静态资源相对路径可被浏览器正确解析
func (h *GrafanaProxyHandler) RegisterProxyRoute(router *gin.Engine) {
	ctx := context.Background()
	grafanaConfig, err := h.configUseCase.GetGrafanaConfig(ctx)
	if err != nil {
		appLogger.Warn("读取 Grafana 配置失败，使用默认 subpath", zap.Error(err))
		grafanaConfig = &systembiz.GrafanaIntegrationConfig{
			Enabled: true,
			Subpath: "/grafana_2syulinm/",
		}
	}

	if !grafanaConfig.Enabled {
		return
	}

	subpath := grafanaConfig.Subpath
	if subpath == "" {
		subpath = "/"
	}
	if subpath[0] != '/' {
		subpath = "/" + subpath
	}

	// 注册路由：/grafana_2syulinm/*proxyPath
	routePattern := strings.TrimRight(subpath, "/") + "/*proxyPath"
	appLogger.Info("注册 Grafana 代理路由", zap.String("pattern", routePattern))
	router.Any(routePattern, h.ProxyGrafana)
}

// ProxyGrafana 代理 Grafana 请求
// 请求路径与 Grafana sub_path 完全一致，直接转发到配置的 Grafana URL
func (h *GrafanaProxyHandler) ProxyGrafana(c *gin.Context) {
	// 动态读取 Grafana 配置
	grafanaConfig, err := h.configUseCase.GetGrafanaConfig(c.Request.Context())
	if err != nil || !grafanaConfig.Enabled {
		if err != nil {
			appLogger.Error("读取 Grafana 配置失败", zap.Error(err))
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"code":    503,
			"message": "Grafana 集成未启用或配置读取失败",
		})
		return
	}

	// 解析目标 URL（例如 http://grafana_mon:3000/grafana_2syulinm/）
	targetURL, err := url.Parse(grafanaConfig.URL)
	if err != nil {
		appLogger.Error("Grafana URL 解析失败", zap.String("url", grafanaConfig.URL), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Grafana URL 配置错误",
		})
		return
	}

	// 构建反向代理，target 为 Grafana host（scheme+host）
	proxyTarget := &url.URL{
		Scheme: targetURL.Scheme,
		Host:   targetURL.Host,
	}
	proxy := httputil.NewSingleHostReverseProxy(proxyTarget)

	// 路径重写：请求路径已经是 /grafana_2syulinm/...，直接转发原始路径
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = proxyTarget.Scheme
		req.URL.Host = proxyTarget.Host
		req.URL.Path = c.Request.URL.Path
		req.URL.RawPath = c.Request.URL.RawPath
		req.URL.RawQuery = c.Request.URL.RawQuery
		req.Host = proxyTarget.Host
	}

	// 移除阻止 iframe 嵌入的响应头
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("X-Frame-Options")
		resp.Header.Set("Content-Security-Policy", "frame-ancestors 'self' *")
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		appLogger.Error("Grafana 代理请求失败", zap.Error(err))
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"code":502,"message":"无法连接到 Grafana 服务"}`))
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}
