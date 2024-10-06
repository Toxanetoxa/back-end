package handlers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Post Структура для поста
type Post struct {
	Body   string `json:"body"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	UserID int    `json:"userId"`
}

// Geo Структура для геолокации
type Geo struct {
	Lat string `json:"lat"`
	Lng string `json:"lng"`
}

// Address Структура для адреса
type Address struct {
	Street  string `json:"street"`
	Suite   string `json:"suite"`
	City    string `json:"city"`
	Zipcode string `json:"zipcode"`
	Geo     Geo    `json:"geo"`
}

// Company Структура для компании
type Company struct {
	Name        string `json:"name"`
	CatchPhrase string `json:"catchPhrase"`
	Bs          string `json:"bs"`
}

// User Структура для пользователя
type User struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Username string  `json:"username"`
	Email    string  `json:"email"`
	Address  Address `json:"address"`
	Phone    string  `json:"phone"`
	Website  string  `json:"website"`
	Company  Company `json:"company"`
}

type RequestBody struct {
	IDs []int `json:"ids"`
}

// GetPosts Прокси для получения постов
func GetPosts(c *gin.Context, redisClient *redis.Client, ctx context.Context) {
	cacheKey := "posts"

	data, err := redisClient.Get(ctx, cacheKey).Result()
	if errors.Is(err, redis.Nil) {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}

		resp, httpErr := client.Get("https://jsonplaceholder.typicode.com/posts") // Todo: вынести в .env
		if httpErr != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error fetching posts from external API: %v", httpErr))
			return
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				c.String(http.StatusInternalServerError, fmt.Sprintf("Error closing body: %v", err))
			}
		}(resp.Body)

		body, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			c.JSON(http.StatusInternalServerError, fmt.Sprintf("Error reading response body: %v", readErr))
			return
		}

		var posts []map[string]interface{}
		if jsonErr := json.Unmarshal(body, &posts); jsonErr != nil {
			c.JSON(http.StatusInternalServerError, fmt.Sprintf("Error unmarshalling response body: %v", jsonErr))
			return
		}

		fmt.Println("Caching data in Redis...")
		setErr := redisClient.Set(ctx, cacheKey, body, 2*time.Minute).Err()
		if setErr != nil {
			c.JSON(http.StatusInternalServerError, fmt.Sprintf("Error saving posts to Redis: %v", setErr))
			return
		}

		c.JSON(http.StatusOK, posts)
	} else if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error fetching posts from Redis: %v", err))
		return
	} else {
		var cachedPosts []map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(data), &cachedPosts); jsonErr != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error unmarshalling cached data: %v", jsonErr))
			return
		}

		c.JSON(http.StatusOK, cachedPosts)
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error fetching posts from external API: %v", httpErr)})
			return
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				// Логируем ошибку, если нужно
			}
		}(resp.Body)

		// Проверяем статус ответа
		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch post, status code: %d", resp.StatusCode)})
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error reading response body: %v", err)})
			return
		}

		// Сохраняем данные в Redis
		err = redisClient.Set(ctx, cacheKey, body, 5*time.Minute).Err()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error saving to cache: %v", err)})
			return
		}

		// Возвращаем данные как объект JSON
		c.JSON(http.StatusOK, json.RawMessage(body))
	} else if err == nil {
		// Если данные найдены в Redis, возвращаем их как объект JSON
		c.JSON(http.StatusOK, json.RawMessage(data))
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error fetching post detail from Redis: %v", err)})
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

		resp, httpErr := client.Get("https://jsonplaceholder.typicode.com/users") //Todo  URL вынесите в .env
		if httpErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching users from external API"})
			return
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching users from external API"})
			}
		}(resp.Body)

		body, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading response body"})
			return
		}

		var users []User
		if unmarshalErr := json.Unmarshal(body, &users); unmarshalErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding JSON response"})
			return
		}

		redisErr := redisClient.Set(ctx, cacheKey, body, 5*time.Minute).Err()
		if redisErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving to Redis"})
			return
		}

		c.JSON(http.StatusOK, users)
	} else if err == nil {
		var users []User
		if unmarshalErr := json.Unmarshal([]byte(data), &users); unmarshalErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding cached JSON"})
			return
		}

		c.JSON(http.StatusOK, users)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching users from Redis"})
	}
}

// GetUser Прокси для получения детальной информации user с кэшем 5 минут
func GetUser(c *gin.Context, redisClient *redis.Client, ctx context.Context) {
	id := c.Param("id")
	cacheKey := fmt.Sprintf("user-%s", id)
	data, err := redisClient.Get(ctx, cacheKey).Result()
	if errors.Is(err, redis.Nil) {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}

		resp, httpErr := client.Get("https://jsonplaceholder.typicode.com/users/" + id)
		if httpErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user from external API"})
			return
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user from external API"})
			}
		}(resp.Body)

		body, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading response body"})
			return
		}

		var user User
		if unmarshalErr := json.Unmarshal(body, &user); unmarshalErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding JSON response"})
			return
		}

		redisErr := redisClient.Set(ctx, cacheKey, body, 5*time.Minute).Err()
		if redisErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving to Redis"})
			return
		}

		c.JSON(http.StatusOK, user)
	} else if err == nil {
		c.JSON(http.StatusOK, json.RawMessage(data))
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user from Redis"})
	}
}

// GetUserPosts Прокси для получения списка постов определённого user-а
func GetUserPosts(c *gin.Context, redisClient *redis.Client, ctx context.Context) {
	id := c.Param("id")
	cacheKey := fmt.Sprintf("user-%s", id, "posts")
	data, err := redisClient.Get(ctx, cacheKey).Result()

	if errors.Is(err, redis.Nil) {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		resp, httpErr := client.Get("https://jsonplaceholder.typicode.com/posts?userId=" + id)
		if httpErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user from external API"})
			return
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user from external API"})
			}
		}(resp.Body)

		body, readErr := ioutil.ReadAll(resp.Body)
		if readErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading response body"})
			return
		}

		var posts []Post
		if unmarshalErr := json.Unmarshal(body, &posts); unmarshalErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding JSON response"})
			return
		}

		redisErr := redisClient.Set(ctx, cacheKey, body, 5*time.Minute).Err()
		if redisErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving to Redis"})
			return
		}

		c.JSON(http.StatusOK, posts)
	} else if err == nil {
		var posts []Post
		if unmarshalErr := json.Unmarshal([]byte(data), &posts); unmarshalErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding cached JSON"})
			return
		}
		c.JSON(http.StatusOK, posts)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user from Redis"})
		return
	}
}

// GetRecentPosts получение просмотренных постов
func GetRecentPosts(c *gin.Context, redisClient *redis.Client, ctx context.Context) {
	var requestBody RequestBody
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, "Invalid request body")
		return
	}
	ids := requestBody.IDs
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, "IDs list cannot be empty")
		return
	}

	idsStr := make([]string, len(ids))
	for i, id := range ids {
		idsStr[i] = strconv.Itoa(id)
	}
	cacheKey := fmt.Sprintf("recentPosts:%s", strings.Join(idsStr, ","))

	data, err := redisClient.Get(ctx, cacheKey).Result()

	if errors.Is(err, redis.Nil) {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		resultChan := make(chan Post, len(ids))

		var wg sync.WaitGroup
		wg.Add(len(ids))

		for _, id := range ids {
			go func(id string) {
				defer wg.Done()

				if id == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "ID cannot be empty"})
					return
				}

				resp, err := client.Get(fmt.Sprintf("https://jsonplaceholder.typicode.com/posts/%s", id))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch post for ID %s: %v", id, err)})
					return
				}
				defer func(Body io.ReadCloser) {
					err := Body.Close()
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch post for ID %s: %v", id, err)})
					}
				}(resp.Body)

				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read response for ID %s: %v", id, err)})
					return
				}

				var post Post
				if err := json.Unmarshal(body, &post); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to unmarshal response for ID %s: %v", id, err)})
					return
				}

				resultChan <- post
			}(strconv.Itoa(id))
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		var posts []Post
		for post := range resultChan {
			posts = append(posts, post)
		}

		postsJSON, jsonErr := json.Marshal(posts)
		if jsonErr != nil {
			c.JSON(http.StatusInternalServerError, fmt.Sprintf("Error marshalling posts: %v", jsonErr))
			return
		}

		setErr := redisClient.Set(ctx, cacheKey, postsJSON, 2*time.Minute).Err()
		if setErr != nil {
			c.JSON(http.StatusInternalServerError, fmt.Sprintf("Error saving recent posts to Redis: %v", setErr))
			return
		}

		c.JSON(http.StatusOK, posts)
	} else if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Error fetching posts from Redis: %v", err))
		return
	} else {
		var cachedPosts []map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(data), &cachedPosts); jsonErr != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("Error unmarshalling cached data: %v", jsonErr))
			return
		}

		c.JSON(http.StatusOK, cachedPosts)
	}
}

// ClearCache метод для отчистки кеша в редисе
func ClearCache(c *gin.Context, redisClient *redis.Client, ctx context.Context) {
	err := redisClient.Del(ctx, "posts").Err()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to clear cache")
		return
	}
	c.String(http.StatusOK, "Cache cleared successfully")
}
