package main

import (
	"back-end/internal/handlers"
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"io"
	"net/http"
	"os"
	"time"
)

var rdb *redis.Client
var ctx = context.Background()

// @title My API
// @version 1.0
// @description This is a sample API for managing posts and users
// @host localhost:8080
// @BasePath /

func main() {
	logFile, err := os.OpenFile("gin.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Failed to open log file:", err)
		return
	}
	defer func(logFile *os.File) {
		err := logFile.Close()
		if err != nil {

		}
	}(logFile)

	gin.DefaultWriter = io.MultiWriter(logFile)

	r := gin.Default()

	// Настройка CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"}, // TODO перенести в .env?
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour, // Время кэширования CORS
	}))

	// Инициализация Redis
	rdb = redis.NewClient(&redis.Options{
		Addr:     getRedisAddr(), // адрес Redis
		Password: "secret",       // ваш пароль
		DB:       0,              // использовать стандартную БД
	})

	// GET Роуты

	// @Summary Get all posts
	// @Description Get a list of all posts
	// @Tags posts
	// @Produce json
	// @Success 200 {array} handlers.Post  "Success"
	// @Router /posts [get]
	r.GET("/posts", func(c *gin.Context) {
		handlers.GetPosts(c, rdb, ctx)
	})

	// @Summary Get post by ID
	// @Description Get details of a specific post
	// @Tags posts
	// @Produce json
	// @Param id path int true "Post ID"
	// @Success 200 {object} handlers.Post  "Success"
	// @Router /posts/{id} [get]
	r.GET("/posts/:id", func(c *gin.Context) {
		handlers.GetPostDetail(c, rdb, ctx)
	})

	// @Summary Get all authors
	// @Description Get a list of all authors
	// @Tags authors
	// @Produce json
	// @Success 200 {array} handlers.User  "Success"
	// @Router /authors [get]
	r.GET("/authors", func(c *gin.Context) {
		handlers.GetUsers(c, rdb, ctx)
	})

	// @Summary Get author by ID
	// @Description Get details of a specific author
	// @Tags authors
	// @Produce json
	// @Param id path int true "Author ID"
	// @Success 200 {object} handlers.User  "Success"
	// @Router /authors/{id} [get]
	r.GET("/authors/:id", func(c *gin.Context) {
		handlers.GetUser(c, rdb, ctx)
	})

	// @Summary Get posts by author ID
	// @Description Get all posts from a specific author
	// @Tags authors
	// @Produce json
	// @Param id path int true "Author ID"
	// @Success 200 {array} handlers.Post  "Success"
	// @Router /userPosts/{id} [get]
	r.GET("/userPosts/:id", func(c *gin.Context) {
		handlers.GetUserPosts(c, rdb, ctx)
	})

	// @Summary Redirect to a fun GIF
	// @Description Redirects the root URL to a fun GIF
	// @Tags misc
	// @Produce json
	// @Success 301 {string} string "Redirected"
	// @Router / [get]
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "https://c.tenor.com/CgGUXc-LDc4AAAAC/tenor.gif")
	})

	// @Summary Clear cache
	// @Description Clears the Redis cache
	// @Tags cache
	// @Produce json
	// @Success 200 {string} string "Cache cleared"
	// @Router /clearCache [get]
	r.GET("clearCache", func(c *gin.Context) {
		handlers.ClearCache(c, rdb, ctx)
	})

	// POST Роуты

	// @Summary Get recent posts
	// @Description Get a list of recent posts
	// @Tags posts
	// @Accept json
	// @Produce json
	// @Param payload body handlers.RecentPostsRequest true "Request body"
	// @Success 200 {array} handlers.Post  "Success"
	// @Router /recentPosts [post]
	r.POST("/recentPosts", func(c *gin.Context) {
		handlers.GetRecentPosts(c, rdb, ctx)
	})

	// Заготовка под swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Запуск сервера
	err = r.Run(":8080")
	if err != nil {
		fmt.Println(err)
		return
	}
}

func getRedisAddr() string {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "redis:6379" //дефолтный адрес для редиса TODO перенести в .env
	}
	return addr
}
