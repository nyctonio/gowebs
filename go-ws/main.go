package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"

	"github.com/go-redis/redis/v8"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

var rdb *redis.Client
var ctx = context.Background()

func InitRedis(redisURL string) {
	u, err := url.Parse(redisURL)
	if err != nil {
		log.Fatalf("Error parsing Redis URL: %v", err)
	}

	password, _ := u.User.Password()

	rdb = redis.NewClient(&redis.Options{
		Addr:     u.Host,
		Password: password,
		Username: u.User.Username(),
		DB:       0,
	})

	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Error connecting to Redis: %v", err)
	}
}

var clients []net.Conn

func RegisterClient(conn net.Conn) {
	clients = append(clients, conn)
}

func UnregisterClient(conn net.Conn) {
	for i, client := range clients {
		if client == conn {
			clients = append(clients[:i], clients[i+1:]...)
			break
		}
	}
}


func BroadcastMessage(msg []byte) {
	buffer := bufferPool.Get().([]byte)
	defer bufferPool.Put(buffer)

	// Ensure the message fits in the buffer. If not, adjust or allocate a bigger buffer as needed.
	if len(msg) > len(buffer) {
		// Allocate a new buffer or adjust strategy.
		// For simplicity, we'll just allocate a new bigger buffer.
		buffer = make([]byte, len(msg))
	}
	copy(buffer, msg)

	activeConnections := 0
	for _, client := range clients {
		if client != nil {
			activeConnections++
			err := wsutil.WriteServerMessage(client, ws.OpText, buffer[:len(msg)])
			if err != nil {
				log.Printf("Error sending message to client: %v", err)
			}
		}
	}
	log.Printf("Broadcast message to %d active connections", activeConnections)
}

func SetUlimit() error {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return fmt.Errorf("error getting rlimit: %v", err)
	}
	err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return fmt.Errorf("error setting rlimit: %v", err)
	}
	return nil
}

// initialising buffer to save memory
var bufferPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096) // Assuming average message size to be 4KB
	},
}


func main() {
	err := SetUlimit()
	if err != nil {
		log.Fatalf("Error setting ulimit: %v", err)
	}

	InitRedis("redis://nio:expo_nio23@143.198.147.232:6379")

	pubsub := rdb.Subscribe(ctx, "channel")
	_, err = pubsub.Receive(ctx)
	if err != nil {
		log.Fatalf("Error subscribing to Redis channel: %v", err)
	}

	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			log.Printf("Received message from Redis: %s", msg.Payload)
			BroadcastMessage([]byte(msg.Payload))
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World!")
	})

	http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			log.Printf("Error upgrading HTTP connection to WebSocket: %v", err)
			return
		}
		go func() {
			RegisterClient(conn)
			defer UnregisterClient(conn)
			defer conn.Close()

			for {
				buffer := bufferPool.Get().([]byte)
				msg, op, err := wsutil.ReadClientData(conn)
				if err != nil {
					log.Printf("Error reading message from client: %v", err)
					bufferPool.Put(buffer)
					return
				}
				err = wsutil.WriteServerMessage(conn, op, msg)
				if err != nil {
					log.Printf("Error sending message to client: %v", err)
					bufferPool.Put(buffer)
					return
				}
				bufferPool.Put(buffer)
			}
		}()
	}))

	os.Exit(0)
}
