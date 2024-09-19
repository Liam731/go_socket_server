run_server:
	go run main.go
test_all:
	go test -run TestRedisConnection
	go test -run TestWebSocketConnection
	go test -run TestWebSocketMessageExchange