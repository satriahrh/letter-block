package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-redis/redis"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/satriahrh/letter-block/service"

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

	svc := service.NewService(tran, dictionaries)
	graphqlResolver := graph.NewResolver(svc)
	graphqlHandler := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: graphqlResolver}))

	graphqlHandler.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	})

	authentication := auth.New(tran)
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", os.Getenv("CORS_ALLOWED_ORIGINS"))
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			next.ServeHTTP(w, r)
		})
	})

	router.Handle("/",
		playground.Handler("GraphQL playground", "/graphql"),
	)
	router.HandleFunc("/register", authentication.Register)
	router.HandleFunc("/authenticate", authentication.Authenticate)
	router.With(authentication.HttpMiddleware).Handle("/graphql", graphqlHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
