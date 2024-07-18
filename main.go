package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strings"
    "os"

    firebase "firebase.google.com/go"
    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
    "google.golang.org/api/option"
)

var (
    firebaseApp *firebase.App
    rdb         *redis.Client
    ctx         = context.Background()
)

func initFirebase() {
    opt := option.WithCredentialsFile("serviceAccountKey.json")
    app, err := firebase.NewApp(context.Background(), nil, opt)
    if err != nil {
        log.Fatalf("error initializing firebase app: %v\n", err)
    }
    firebaseApp = app
}

func initRedis() {
    redisPassword := os.Getenv("REDIS_PASSWORD")
    rdb = redis.NewClient(&redis.Options{
        Addr:     "redis:6379",
        Password: redisPassword,
        DB:       0,  // デフォルトDB
    })

    _, err := rdb.Ping(ctx).Result()
    if err != nil {
        log.Fatalf("error connecting to redis: %v\n", err)
    }
}

func authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        bearerToken := c.GetHeader("Authorization")
        splitToken := strings.Split(bearerToken, "Bearer ")
        idToken := splitToken[1]
        if idToken == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }

        ctx := context.Background()
        client, err := firebaseApp.Auth(ctx)
        if (err != nil) {
            log.Fatalf("error getting Auth client: %v\n", err)
        }

        token, err := client.VerifyIDToken(ctx, idToken)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }

        c.Set("uid", token.UID)
        c.Next()
    }
}

type Todo struct {
    ID   string `json:"id"`
    Text string `json:"text"`
    Done bool   `json:"done"`
}

func getTodos(c *gin.Context) {
    val, err := rdb.Get(ctx, "todos").Result()
    if err == redis.Nil {
        c.JSON(http.StatusOK, []Todo{})
        return
    } else if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var todos []Todo
    if err := json.Unmarshal([]byte(val), &todos); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, todos)
}

func createTodo(c *gin.Context) {
    var newTodo Todo
    if err := c.ShouldBindJSON(&newTodo); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    val, err := rdb.Get(ctx, "todos").Result()
    if err != nil && err != redis.Nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var todos []Todo
    if val != "" {
        if err := json.Unmarshal([]byte(val), &todos); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
    }

    newTodo.ID = fmt.Sprintf("%d", len(todos)+1)
    todos = append(todos, newTodo)

    jsonData, err := json.Marshal(todos)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if err := rdb.Set(ctx, "todos", jsonData, 0).Err(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, newTodo)
}

func updateTodo(c *gin.Context) {
    id := c.Param("id")
    var updatedTodo Todo
    if err := c.ShouldBindJSON(&updatedTodo); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    val, err := rdb.Get(ctx, "todos").Result()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var todos []Todo
    if err := json.Unmarshal([]byte(val), &todos); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    for i, todo := range todos {
        if todo.ID == id {
            todos[i].Text = updatedTodo.Text
            todos[i].Done = updatedTodo.Done

            jsonData, err := json.Marshal(todos)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }

            if err := rdb.Set(ctx, "todos", jsonData, 0).Err(); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }

            c.JSON(http.StatusOK, todos[i])
            return
        }
    }

    c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
}

func deleteTodo(c *gin.Context) {
    id := c.Param("id")

    val, err := rdb.Get(ctx, "todos").Result()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var todos []Todo
    if err := json.Unmarshal([]byte(val), &todos); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    for i, todo := range todos {
        if todo.ID == id {
            todos = append(todos[:i], todos[i+1:]...)

            jsonData, err := json.Marshal(todos)
            if err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }

            if err := rdb.Set(ctx, "todos", jsonData, 0).Err(); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
            }

            c.JSON(http.StatusOK, gin.H{"message": "Todo deleted"})
            return
        }
    }

    c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
}

func main() {
    initFirebase()
    initRedis()

    r := gin.Default()

    api := r.Group("/api")
    api.Use(authMiddleware())

    api.GET("/todos", getTodos)
    api.POST("/todos", createTodo)
    api.PUT("/todos/:id", updateTodo)
    api.DELETE("/todos/:id", deleteTodo)

    r.Run(":8080")
}
