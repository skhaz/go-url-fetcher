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

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	_ "go.uber.org/automaxprocs"
	"schneider.vip/problem"
)

type Query struct {
	Url        string `form:"url" binding:"required"`
	Expiration int    `form:"interval,default=3600"`
}

func Fetch(c *gin.Context) {
	var err error

	q := Query{}
	if err = c.BindQuery(&q); err != nil {
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

	ctx := c.MustGet("Context").(context.Context)
	rdb := c.MustGet("Redis").(*redis.Client)
	body, _ := rdb.Get(ctx, dataKey).Bytes()

	if len(body) == 0 {
		var resp *http.Response

		resp, err = http.Get(q.Url)
		if err != nil {
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
			problem.New(
				problem.Title("Error While Reading The Body"),
				problem.Type("errors:request/invalid-url"),
				problem.Detail(err.Error()),
				problem.Status(http.StatusBadRequest),
			).WriteTo(c.Writer)

			return
		}

		rdb.Set(ctx, dataKey, body, time.Duration(q.Expiration)*time.Second)
	}

	c.Writer.Header().Set("Cache-Control", "public, max-age=86400, immutable")
	c.Data(http.StatusOK, gin.MIMEPlain, body)
	return
}

func main() {
	var ctx = context.Background()

	opts, _ := redis.ParseURL(os.Getenv("REDIS_URL"))
	rdb := redis.NewClient(opts)
	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(err)
	}

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Set("Context", ctx)
		c.Set("Redis", rdb)
		c.Next()
	})

	router.GET("/", Fetch)

	router.Run()
}