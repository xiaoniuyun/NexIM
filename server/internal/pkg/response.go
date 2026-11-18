package pkg

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code       int         `json:"code"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
	TraceID    string      `json:"traceId"`
	ServerTime int64       `json:"serverTime"`
}

func traceID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func NewTrace() string { return traceID() }

func JSON(c *gin.Context, httpStatus int, code int, msg string, data interface{}) {
	c.Set("traceId", traceID())
	c.JSON(httpStatus, Response{
		Code: code, Message: msg, Data: data,
		TraceID: c.GetString("traceId"), ServerTime: time.Now().UnixMilli(),
	})
}

func OK(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, 0, "ok", data)
}

func Fail(c *gin.Context, httpStatus int, code int, msg string) {
	JSON(c, httpStatus, code, msg, nil)
}
