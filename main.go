package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

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
        if err != nil {
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
    ID     string  `json:"id"`
    Text   string  `json:"text"`
    Done   bool    `json:"done"`
    Lat    float64 `json:"lat"`
    Lng    float64 `json:"lng"`
}

func createTodo(c *gin.Context) {
    var newTodo Todo
    if err := c.ShouldBindJSON(&newTodo); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    newTodo.ID = strconv.FormatInt(time.Now().UnixNano(), 10)

    jsonData, err := json.Marshal(newTodo)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if err := rdb.Set(ctx, fmt.Sprintf("todo:%s", newTodo.ID), jsonData, 0).Err(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 位置情報をRedisに保存
    if _, err := rdb.GeoAdd(ctx, "todos:locations", &redis.GeoLocation{
        Name:      newTodo.ID,
        Latitude:  newTodo.Lat,
        Longitude: newTodo.Lng,
    }).Result(); err != nil {
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

    val, err := rdb.Get(ctx, fmt.Sprintf("todo:%s", id)).Result()
    if err == redis.Nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
        return
    } else if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var existingTodo Todo
    if err := json.Unmarshal([]byte(val), &existingTodo); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    existingTodo.Text = updatedTodo.Text
    existingTodo.Done = updatedTodo.Done
 //   oldLat := existingTodo.Lat
 //   oldLng := existingTodo.Lng
    existingTodo.Lat = updatedTodo.Lat
    existingTodo.Lng = updatedTodo.Lng

    jsonData, err := json.Marshal(existingTodo)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if err := rdb.Set(ctx, fmt.Sprintf("todo:%s", existingTodo.ID), jsonData, 0).Err(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 位置情報を更新
    if _, err := rdb.ZRem(ctx, "todos:locations", id).Result(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if _, err := rdb.GeoAdd(ctx, "todos:locations", &redis.GeoLocation{
        Name:      existingTodo.ID,
        Latitude:  existingTodo.Lat,
        Longitude: existingTodo.Lng,
    }).Result(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, existingTodo)
}

func deleteTodo(c *gin.Context) {
    id := c.Param("id")

    _, err := rdb.Get(ctx, fmt.Sprintf("todo:%s", id)).Result()
    if err == redis.Nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Todo not found"})
        return
    } else if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if err := rdb.Del(ctx, fmt.Sprintf("todo:%s", id)).Err(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 位置情報を削除
    if _, err := rdb.ZRem(ctx, "todos:locations", id).Result(); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Todo deleted"})
}

func searchTodos(c *gin.Context) {
    lat := c.Query("lat")
    lng := c.Query("lng")
    radius := c.Query("radius")

    latF, err := strconv.ParseFloat(lat, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid latitude"})
        return
    }

    lngF, err := strconv.ParseFloat(lng, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid longitude"})
        return
    }

    radiusF, err := strconv.ParseFloat(radius, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid radius"})
        return
    }

    locations, err := rdb.GeoRadius(ctx, "todos:locations", lngF, latF, &redis.GeoRadiusQuery{
        Radius: radiusF,
        Unit:   "km",
    }).Result()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var result []Todo
    for _, location := range locations {
        val, err := rdb.Get(ctx, fmt.Sprintf("todo:%s", location.Name)).Result()
        if err == redis.Nil {
            continue
        } else if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        var todo Todo
        if err := json.Unmarshal([]byte(val), &todo); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }

        result = append(result, todo)
    }

    c.JSON(http.StatusOK, result)
}

func main() {
    initFirebase()
    initRedis()

    r := gin.Default()

    api := r.Group("/api")
    api.Use(authMiddleware())

    api.POST("/todos", createTodo)
    api.PUT("/todos/:id", updateTodo)
    api.DELETE("/todos/:id", deleteTodo)
    api.GET("/todos/search", searchTodos)

    r.Run(":8080")
}
