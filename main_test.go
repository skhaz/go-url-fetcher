package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gopkg.in/h2non/gock.v1"
)

func TestGetNoParameters(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	r := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(r)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Set("Logger", logger)

	Fetch(c)

	assert.Equal(t, http.StatusBadRequest, r.Code)
	assert.Equal(t, "application/problem+json", r.Header().Get("Content-Type"))
	assert.Contains(t, r.Body.String(), "errors:params/invalid-query-parameters")
}

func TestGetCachedUrl(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	var ctx = context.TODO()
	redis, mock := redismock.NewClientMock()

	mock.MatchExpectationsInOrder(true)
	mock.ExpectGet("dfa8ce7471028ee0addb32f80fa8ecdcd7e112cf:data").SetVal("1.1.1.1")

	r := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(r)
	c.Request, _ = http.NewRequest("GET", "/?url=https://1.1.1.1/", nil)
	c.Set("Context", &ctx)
	c.Set("Logger", logger)
	c.Set("Redis", redis)

	Fetch(c)

	assert.Equal(t, http.StatusOK, r.Code)
	assert.Equal(t, gin.MIMEPlain, r.Header().Get("Content-Type"))
	assert.Contains(t, r.Body.String(), "1.1.1.1")

	mock.ExpectationsWereMet()
}

func TestGetNotCachedUrl(t *testing.T) {
	defer gock.Off()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	htmlBody := "<!DOCTYPE html>"

	gock.New("https://1.1.1.1/").
		Get("/").
		Reply(200).
		BodyString(htmlBody)

	var ctx = context.TODO()
	redis, mock := redismock.NewClientMock()

	mock.MatchExpectationsInOrder(true)
	mock.ExpectSet("dfa8ce7471028ee0addb32f80fa8ecdcd7e112cf:data", htmlBody, time.Duration(3600)*time.Second)

	r := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(r)
	c.Request, _ = http.NewRequest("GET", "/?url=https://1.1.1.1/", nil)
	c.Set("Context", &ctx)
	c.Set("Logger", logger)
	c.Set("Redis", redis)

	Fetch(c)

	assert.Equal(t, http.StatusOK, r.Code)
	assert.Equal(t, gin.MIMEPlain, r.Header().Get("Content-Type"))
	assert.Contains(t, r.Body.String(), htmlBody)
	assert.True(t, gock.IsDone())
	mock.ExpectationsWereMet()
}
