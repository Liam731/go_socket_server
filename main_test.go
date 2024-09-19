package main

import (
	"context"
	"testing"
	"time"
	"websocket/util"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var testCtx = context.Background()

func init() {
	var err error
	config, err = util.LoadConfig(".")
	if err != nil {
		panic("Failed to load config: " + err.Error())
	}
}

// createRedisClient initializes a new Redis client instance for testing.
func createRedisClient() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddress,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})
	return rdb
}

// createWebSocketConnection establishes a WebSocket connection to the test server.
func createWebSocketConnection(t *testing.T) *websocket.Conn {
	// Construct the WebSocket URL using the server's address.、
	wsURL := "ws://localhost:8080/socket"

	// Dial to the WebSocket URL.
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err, "WebSocket connection should be established")

	return ws
}

// TestRedisConnection checks if a connection to Redis can be established.
func TestRedisConnection(t *testing.T) {
	rdb := createRedisClient()
	defer rdb.Close()

	// PING the Redis server to test the connection.
	_, err := rdb.Ping(testCtx).Result()
	assert.NoError(t, err, "Redis should be reachable")
}

// TestWebSocketConnection checks if the WebSocket connection can be established.
func TestWebSocketConnection(t *testing.T) {
	rdb := createRedisClient()
	// defer rdb.Close()

	gin.SetMode(gin.TestMode)
	server := StartServer(rdb)
	defer server.Shutdown(context.Background()) // Ensure the server is closed properly.

	// Create WebSocket connection using the helper function.
	ws := createWebSocketConnection(t)

	// Send a test message to verify the connection is open.
	err := ws.WriteMessage(websocket.TextMessage, []byte("test"))
	assert.NoError(t, err, "Message should be sent via WebSocket")

	// Read the message to check the connection.
	_, _, err = ws.ReadMessage()
	assert.NoError(t, err, "Message should be read via WebSocket")
}

// TestWebSocketMessageExchange checks the message exchange between WebSocket and Redis.
func TestWebSocketMessageExchange(t *testing.T) {
	rdb := createRedisClient()
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	server := StartServer(rdb)
	defer server.Shutdown(context.Background())

	// Subscribe to Redis channel for testing message relay.
	sub := rdb.Subscribe(context.Background(), "websocketChannel")
	defer sub.Close()
	ch := sub.Channel()

	// Create WebSocket connection using the helper function.
	ws := createWebSocketConnection(t)

	// Send a test message to the WebSocket.
	testMessage := "hello from client"
	err := ws.WriteMessage(websocket.TextMessage, []byte(testMessage))
	assert.NoError(t, err, "Message should be sent via WebSocket")

	// Verify if the message is received via Redis.
	select {
	case msg := <-ch:
		assert.Equal(t, testMessage, msg.Payload, "Redis should receive the correct message")
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for Redis message")
	}
}
