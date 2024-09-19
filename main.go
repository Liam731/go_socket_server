package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"
	"websocket/util"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var (
	ctx    = context.Background()
	config util.Config
)

func init() {
	var err error
	config, err = util.LoadConfig(".")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Configuration loaded successfully")
}

// InitializeRedis sets up and returns a new Redis client instance.
func InitRedis() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress,  // Address of the Redis server.
		Password: config.RedisPassword, // Password for the Redis server; empty by default.
		DB:       config.RedisDB,       // Use the default Redis database.
	})

	return rdb
}

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	rdb := InitRedis() // Initialize the Redis client.
	defer rdb.Close()  // Ensure the Redis client is properly closed upon exit.

	server := StartServer(rdb) // Start the HTTP server and pass the Redis client.

	// Wait for an interrupt signal to shut down the server.
	<-stop
	fmt.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Gracefully shut down the server.
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Error shutting down server: %v\n", err)
	}
}

// StartServer initializes and starts an HTTP server, returning the server instance.
func StartServer(rdb *redis.Client) *http.Server {
	g := gin.New()
	g.Use(gin.Recovery()) // Use the default recovery middleware to recover from panics.

	if err := g.SetTrustedProxies(nil); err != nil {
		panic(err) // Set trusted proxies, if any.
	}

	public := g.Group("/socket")
	public.GET("", func(c *gin.Context) {
		SocketHandler(c, rdb) // Pass the Redis client to the WebSocket handler.
	})

	server := &http.Server{
		Addr:    config.ServerAddress, // Server address.
		Handler: g,                    // Gin router as the server handler.
	}

	// Start the server in a goroutine to allow asynchronous operation.
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Error starting server: %v\n", err)
		}
	}()

	return server
}

// SocketHandler handles WebSocket connections and interacts with Redis.
func SocketHandler(c *gin.Context, rdb *redis.Client) {
	upGrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow connections from any origin.
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Printf("Error upgrading connection: %v\n", err)
		return
	}

	defer func() {
		if closeSocketErr := ws.Close(); closeSocketErr != nil {
			fmt.Printf("Error closing WebSocket: %v\n", closeSocketErr)
		}
	}()

	// Subscribe to the Redis channel.
	sub := rdb.Subscribe(ctx, "websocketChannel")
	defer sub.Close()

	ch := sub.Channel()

	// Start a goroutine to handle messages from Redis and send them to the WebSocket client.
	go func() {
		for msg := range ch {
			if err := ws.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				fmt.Printf("Error writing message to WebSocket: %v\n", err)
				return
			}
		}
	}()

	// Main loop to handle messages from the WebSocket client and publish them to Redis.
	for {
		msgType, msg, err := ws.ReadMessage()
		if err != nil {
			fmt.Printf("Error reading message: %v\n", err)
			return
		}

		fmt.Printf("Message Type: %d, Message: %s\n", msgType, string(msg))

		if err := rdb.Publish(ctx, "websocketChannel", msg).Err(); err != nil {
			fmt.Printf("Error publishing message to Redis: %v\n", err)
			return
		}
	}
}
