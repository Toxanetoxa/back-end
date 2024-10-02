package main

import (
	"back-end/internal/handlers"
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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
	// @Success 200 {array} Post  "Success"
	// @Router /posts [get]
	r.GET("/posts", func(c *gin.Context) {
		handlers.GetPosts(c, rdb, ctx)
	})

	// @Summary Get post by ID
	// @Description Get details of a specific post
	// @Tags posts
	// @Produce json
	// @Param id path int true "Post ID"
	// @Success 200 {object} Post  "Success"
	// @Router /posts/{id} [get]
	r.GET("/posts/:id", func(c *gin.Context) {
		handlers.GetPostDetail(c, rdb, ctx)
	})

	// @Summary Get all users
	// @Description Get a list of all users
	// @Tags users
	// @Produce json
	// @Success 200 {array} User  "Success"
	// @Router /users [get]
	r.GET("/users", func(c *gin.Context) {
		handlers.GetUsers(c, rdb, ctx)
	})

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "https://c.tenor.com/CgGUXc-LDc4AAAAC/tenor.gif")
	})

	// POST Роуты
	r.POST("/post/seen", postSeenArticle)

	// Заготовка под swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Запуск сервера
	err := r.Run(":8080")
	if err != nil {
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

// POST запрос для записи просмотренной статьи
func postSeenArticle(c *gin.Context) {
	// Проверка капчи (условно)
	captchaToken := c.PostForm("captcha")
	if !verifyCaptcha(captchaToken) {
		c.String(http.StatusBadRequest, "Invalid captcha")
		return
	}

	// Получаем или создаем UUID пользователя
	userUUID, err := c.Cookie("user_uuid")
	if err != nil {
		userUUID = uuid.New().String()
		c.SetCookie("user_uuid", userUUID, 3600*24*365, "/", "localhost", false, true)
	}

	// Получаем ID статьи из запроса
	var requestBody struct {
		PostID int `json:"post_id"`
	}
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.String(http.StatusBadRequest, "Invalid request")
		return
	}

	// Записываем просмотренную статью (условно)
	recordSeenArticle(userUUID, requestBody.PostID)

	c.String(http.StatusOK, "Article seen recorded")
}

// Верификация капчи
func verifyCaptcha(token string) bool {
	// Здесь проверка капчи, возвращает true, если прошла
	return token == "valid-captcha-token"
}

// Запись просмотренной статьи
func recordSeenArticle(userUUID string, postID int) {
	fmt.Printf("User %s viewed post %d\n", userUUID, postID)
	// Запись в базу или in-memory store
}
