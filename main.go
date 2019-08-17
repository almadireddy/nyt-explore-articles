package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"time"
)

var redisClient *redis.Client

func main() {
	port := os.Getenv("PORT")
	registerHandlers()
	client := initStore()
	initRedisClient()
	defer client.Close()
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func initStore() *firestore.Client {
	bg := context.Background()

	var s FirestoreDb
	client, err := firestore.NewClient(bg, "nyt-explore-prd")
	if err != nil {
		log.Fatalf("could not create firestore client %v", err)
	}

	s.client = client
	InitArticleStore(&s)
	return client
}

func registerHandlers() {
	r := mux.NewRouter()
	r.HandleFunc("/articles", Logger(GetArticles, "Articles")).Methods("GET")
	r.HandleFunc("/articles", Logger(CreateArticle, "Create Articles")).Methods("POST")
	r.HandleFunc("/images", Logger(GetImages, "Images")).Methods("GET")
	http.Handle("/", r)
}

func initRedisClient() {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	redisClient = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port),
		Password: "",
		DB: 0,
	})

	pong, err := redisClient.Ping().Result()
	if err != nil {
		log.Println("err in ping", err)
	}
	log.Println(pong)
}

func Logger(inner http.HandlerFunc, name string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		inner.ServeHTTP(w, r)

		log.Printf(
			"%s\t%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}
