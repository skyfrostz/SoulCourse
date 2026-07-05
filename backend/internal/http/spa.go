package httpserver

import (
	"bytes"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/http/webdist"
	"subject-choice-forum/backend/internal/logx"

	"github.com/gin-gonic/gin"
)

type spaAssets struct {
	filesystem fs.FS
	source     string
}

func registerSPA(router *gin.Engine, logger *logx.Logger, distDir string, basePath string) {
	assets, ok := resolveSPAAssets(distDir)
	if !ok {
		logger.Warn("静态资源", "未找到可托管的前端构建产物", logx.F("磁盘目录", distDir))
		return
	}

	logger.Info("静态资源", "已启用前端托管", logx.F("来源", assets.source))

	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}

		requestPath := cleanRequestPath(c.Request.URL.Path)
		appPath, ok := stripBasePath(requestPath, basePath)
		if !ok || skipSPAPath(appPath) {
			c.Status(http.StatusNotFound)
			return
		}

		if appPath == "/" {
			serveFSFile(c, assets.filesystem, "index.html")
			return
		}

		candidate := strings.TrimPrefix(appPath, "/")
		if existsInFS(assets.filesystem, candidate) {
			serveFSFile(c, assets.filesystem, candidate)
			return
		}

		// Static asset requests should fail fast when the file is missing.
		// Returning index.html here causes browsers to report confusing MIME errors
		// for JS/CSS requests and hides deployment sync problems.
		if filepath.Ext(candidate) != "" {
			c.Status(http.StatusNotFound)
			return
		}

		serveFSFile(c, assets.filesystem, "index.html")
	})
}

func resolveSPAAssets(distDir string) (spaAssets, bool) {
	if distDir == "" {
		if embedded, ok := embeddedSPAAssets(); ok {
			return embedded, true
		}
		return spaAssets{}, false
	}
	if !existsInFS(os.DirFS(distDir), "index.html") {
		if embedded, ok := embeddedSPAAssets(); ok {
			return embedded, true
		}
		return spaAssets{}, false
	}
	return spaAssets{
		filesystem: os.DirFS(distDir),
		source:     "磁盘目录：" + distDir,
	}, true
}

func embeddedSPAAssets() (spaAssets, bool) {
	if !existsInFS(webdist.Files, "dist/index.html") {
		return spaAssets{}, false
	}
	sub, err := fs.Sub(webdist.Files, "dist")
	if err != nil {
		return spaAssets{}, false
	}
	return spaAssets{
		filesystem: sub,
		source:     "二进制内嵌资源",
	}, true
}

func existsInFS(filesystem fs.FS, filePath string) bool {
	info, err := fs.Stat(filesystem, filePath)
	return err == nil && !info.IsDir()
}

func cleanRequestPath(requestPath string) string {
	cleaned := path.Clean("/" + requestPath)
	if cleaned == "." {
		return "/"
	}
	return cleaned
}

func stripBasePath(requestPath string, basePath string) (string, bool) {
	if basePath == "" {
		return requestPath, true
	}
	if requestPath == basePath {
		return "/", true
	}
	if !strings.HasPrefix(requestPath, basePath+"/") {
		return "", false
	}
	trimmed := strings.TrimPrefix(requestPath, basePath)
	if trimmed == "" {
		return "/", true
	}
	return trimmed, true
}

func skipSPAPath(requestPath string) bool {
	return requestPath == "/healthz" ||
		requestPath == "/readyz" ||
		strings.HasPrefix(requestPath, "/api/") ||
		strings.HasPrefix(requestPath, "/uploads/")
}

func serveFSFile(c *gin.Context, filesystem fs.FS, filePath string) {
	content, err := fs.ReadFile(filesystem, filePath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	if contentType := mime.TypeByExtension(path.Ext(filePath)); contentType != "" {
		c.Header("Content-Type", contentType)
	}
	http.ServeContent(c.Writer, c.Request, filePath, time.Time{}, bytes.NewReader(content))
}
