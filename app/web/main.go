package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/go-redis/redis"
	"github.com/joho/godotenv"

	"github.com/satriahrh/letter-block"
	data_dictionary "github.com/satriahrh/letter-block/data/dictionary"
	"github.com/satriahrh/letter-block/data/transactional"
	"github.com/satriahrh/letter-block/dictionary"
	"github.com/satriahrh/letter-block/dictionary/id_id"
	"github.com/satriahrh/letter-block/graph"
	"github.com/satriahrh/letter-block/graph/generated"
	"github.com/satriahrh/letter-block/middleware/auth"
)

const defaultPort = "8080"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Error loading .env, will be using system's environment variables instead: %v\n", err)
	}

	db, err := sql.Open(
		"mysql",
		os.Getenv("MYSQL_DSN"),
	)
	if err != nil {
		panic(err)
	}

	redisOptions, err := redis.ParseURL(os.Getenv("REDIS_URL"))
	if err != nil {
		panic(err)
	}
	redisClient := redis.NewClient(redisOptions)

	tran := transactional.NewTransactional(db)
	dataDict := data_dictionary.NewDictionary(72*time.Hour, redisClient)
	dictionaries := map[string]dictionary.Dictionary{
		"id-id": id_id.NewIdId(dataDict, http.DefaultClient),
	}

	app := letter_block.NewApplication(tran, dictionaries)
	graphqlResolver := graph.NewResolver(app)

	authentication := auth.New(tran)
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ","),
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"*"},
		AllowCredentials: true,
		// MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	router.Handle("/",
		playground.Handler("GraphQL playground", "/graphql"),
	)
	router.HandleFunc("/register", authentication.Register)
	router.HandleFunc("/authenticate", authentication.Authenticate)
	router.With(authentication.HttpMiddleware).Handle("/graphql",
		handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: graphqlResolver})),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
