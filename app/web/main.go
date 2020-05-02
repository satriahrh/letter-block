package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-redis/redis"

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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	db, err := sql.Open(
		"mysql",
		"root:rootpw@/letter_block_development",
	)
	if err != nil {
		panic(err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	tran := transactional.NewTransactional(db)
	dataDict := data_dictionary.NewDictionary(72*time.Hour, redisClient)
	dictionaries := map[string]dictionary.Dictionary{
		"id-id": id_id.NewIdId(dataDict, http.DefaultClient),
	}

	app := letter_block.NewApplication(tran, dictionaries)
	graphqlResolver := graph.NewResolver(app)
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: graphqlResolver}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", auth.Middleware(srv))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
