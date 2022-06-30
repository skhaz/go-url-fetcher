package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"
	"schneider.vip/problem"
)

type Query struct {
	Url        string `form:"url" binding:"required"`
	Expiration int    `form:"interval,default=3600"`
}

func Fetch(c *gin.Context) {
	var err error
	logger := c.MustGet("Logger").(*zap.Logger)
	q := Query{}
	if err = c.BindQuery(&q); err != nil {
		logger.Warn("invalid query parameters",
			zap.String("url", c.FullPath()),
			zap.Error(err),
		)

		problem.New(
			problem.Title("Invalid Query Parameters"),
			problem.Type("errors:params/invalid-query-parameters"),
			problem.Detail(err.Error()),
			problem.Status(http.StatusBadRequest),
		).WriteTo(c.Writer)

		return
	}

	h := sha1.New()
	h.Write([]byte(q.Url))
	keyPrefix := hex.EncodeToString(h.Sum(nil))
	dataKey := fmt.Sprintf("%v:data", keyPrefix)

	ctx := c.MustGet("Context").(*context.Context)
	rdb := c.MustGet("Redis").(*redis.Client)
	body, _ := rdb.Get(*ctx, dataKey).Bytes()

	if len(body) == 0 {
		var resp *http.Response

		resp, err = http.Get(q.Url)
		if err != nil {
			logger.Warn("invalid url",
				zap.String("url", q.Url),
				zap.Error(err),
			)

			problem.New(
				problem.Title("Invalid URL"),
				problem.Type("errors:request/invalid-url"),
				problem.Detail(err.Error()),
				problem.Status(http.StatusBadRequest),
			).WriteTo(c.Writer)

			return
		}

		defer resp.Body.Close()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			logger.Warn("error while reading the body",
				zap.String("url", q.Url),
				zap.Error(err),
			)

			problem.New(
				problem.Title("Error While Reading The Body"),
				problem.Type("errors:request/invalid-url"),
				problem.Detail(err.Error()),
				problem.Status(http.StatusBadRequest),
			).WriteTo(c.Writer)

			return
		}

		rdb.Set(*ctx, dataKey, body, time.Duration(q.Expiration)*time.Second)
	}

	c.Writer.Header().Set("Cache-Control", "public, max-age=86400, immutable")
	c.Data(http.StatusOK, gin.MIMEPlain, body)
	return
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	var ctx = context.Background()

	opts, _ := redis.ParseURL(os.Getenv("REDIS_URL"))
	rdb := redis.NewClient(opts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	router := gin.Default()

	router.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger, true))
	router.Use(func(c *gin.Context) {
		c.Set("Context", &ctx)
		c.Set("Logger", logger)
		c.Set("Redis", rdb)
		c.Next()
	})

	router.GET("/", Fetch)

	router.Run()
}
