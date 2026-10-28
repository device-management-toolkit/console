package httpapi

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed password-change.html
var passwordPage embed.FS

func servePasswordChangePage(c *gin.Context) {
	page, err := passwordPage.ReadFile("password-change.html")
	if err != nil {
		c.Status(http.StatusInternalServerError)

		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", page)
}
