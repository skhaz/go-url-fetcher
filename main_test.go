package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
)

func TestGetNoParameters(t *testing.T) {
	r := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(r)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	Fetch(c)

	assert.Equal(t, http.StatusBadRequest, r.Code)
	assert.Equal(t, "application/problem+json", r.Header().Get("Content-Type"))
	assert.Contains(t, r.Body.String(), "errors:params/invalid-query-parameters")
}

func TestGetCachedUrl(t *testing.T) {
	var ctx = context.TODO()
	redis, mock := redismock.NewClientMock()

	mock.MatchExpectationsInOrder(true)
	mock.ExpectGet("dfa8ce7471028ee0addb32f80fa8ecdcd7e112cf:data").SetVal("1.1.1.1")

	r := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(r)
	c.Request, _ = http.NewRequest("GET", "/?url=https://1.1.1.1/", nil)
	c.Set("Context", ctx)
	c.Set("Redis", redis)

	Fetch(c)

	assert.Equal(t, http.StatusOK, r.Code)
	assert.Equal(t, gin.MIMEPlain, r.Header().Get("Content-Type"))
	assert.Contains(t, r.Body.String(), "1.1.1.1")

	mock.ExpectationsWereMet()
}

func TestGetNotCachedUrl(t *testing.T) {
	var ctx = context.TODO()
	redis, mock := redismock.NewClientMock()

	r := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(r)
	c.Request, _ = http.NewRequest("GET", "/?url=https://1.1.1.1/", nil)
	c.Set("Context", ctx)
	c.Set("Redis", redis)

	Fetch(c)

	assert.Equal(t, http.StatusOK, r.Code)
	assert.Equal(t, gin.MIMEPlain, r.Header().Get("Content-Type"))
	assert.Contains(t, r.Body.String(), "<!DOCTYPE html>")

	mock.ExpectationsWereMet()
}
