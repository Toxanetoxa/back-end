package handlers

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

func GetPosts(c *gin.Context, redisClient *redis.Client, ctx context.Context) {
	cacheKey := "posts"

	// Попытка получить данные из Redis
	data, err := redisClient.Get(ctx, cacheKey).Result()
	if errors.Is(err, redis.Nil) {
		// Создаем кастомный HTTP клиент с игнорированием сертификатов
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Важно: использовать только для разработки!
		}
		client := &http.Client{Transport: tr}

		resp, httpErr := client.Get("https://jsonplaceholder.typicode.com/posts") //Todo вынести в .env
		if httpErr != nil {
			// Выводим ошибку, если запрос к внешнему API не удался
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error fetching posts from external API: %v", httpErr))
			return
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {

			}
		}(resp.Body)

		// Чтение тела ответа
		body, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			// Выводим ошибку при чтении тела ответа
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error reading response body: %v", readErr))
			return
		}

		// Кешируем данные в Redis на 2 минуты
		var setErr = redisClient.Set(ctx, cacheKey, body, 2*time.Minute).Err()
		if setErr != nil {
			// Выводим ошибку, если не удалось сохранить данные в Redis
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error saving posts to Redis: %v", setErr))
			return
		}

		// Возвращаем результат запроса
		c.String(http.StatusOK, string(body))
	} else if err != nil {
		// Выводим ошибку, если Redis вернул ошибку
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error fetching posts from Redis: %v", err))
	} else {
		// Возвращаем кэшированные данные
		c.String(http.StatusOK, data)
	}
}

// GetPostDetail Прокси для детальной статьи с кэшем 5 минут
func GetPostDetail(c *gin.Context, redisClient *redis.Client, ctx context.Context) {
	id := c.Param("id")
	cacheKey := fmt.Sprintf("post_%s", id)

	// Проверяем наличие данных в Redis
	data, err := redisClient.Get(ctx, cacheKey).Result()
	if errors.Is(err, redis.Nil) {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		// Если данных нет, делаем HTTP-запрос
		resp, httpErr := client.Get(fmt.Sprintf("https://jsonplaceholder.typicode.com/posts/%s", id)) // Todo: вынести в .env
		if httpErr != nil {
			// Выводим ошибку, если запрос к внешнему API не удался
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error fetching posts from external API: %v", httpErr))
			return
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {

			}
		}(resp.Body)

		// Проверяем статус ответа
		if resp.StatusCode != http.StatusOK {
			c.String(http.StatusInternalServerError, "Failed to fetch post, status code: %d", resp.StatusCode)
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, "Error reading response body: %v", err)
			return
		}

		// Сохраняем данные в Redis
		err = redisClient.Set(ctx, cacheKey, body, 5*time.Minute).Err()
		if err != nil {
			c.String(http.StatusInternalServerError, "Error saving to cache: %v", err)
			return
		}

		c.String(http.StatusOK, string(body))
	} else if err == nil {
		// Если данные найдены в Redis
		c.String(http.StatusOK, data)
	} else {
		c.String(http.StatusInternalServerError, "Error fetching post detail from Redis: %v", err)
	}
}

// GetUsers Прокси для получения списка пользователей с кэшем 5 минут
func GetUsers(c *gin.Context, redisClient *redis.Client, ctx context.Context) {
	cacheKey := "users"
	data, err := redisClient.Get(ctx, cacheKey).Result()
	if errors.Is(err, redis.Nil) {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}

		resp, _ := client.Get("https://jsonplaceholder.typicode.com/users") //Todo вынести в .env
		body, _ := ioutil.ReadAll(resp.Body)
		redisClient.Set(ctx, cacheKey, body, 5*time.Minute)
		c.String(http.StatusOK, string(body))
	} else if err == nil {
		c.String(http.StatusOK, data)
	} else {
		c.String(http.StatusInternalServerError, "Error fetching users")
	}
}
