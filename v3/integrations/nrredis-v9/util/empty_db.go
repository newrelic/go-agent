package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

func main() {
	db := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx := context.Background()
	size, err := db.DBSize(ctx).Result()
	if err != nil {
		log.Fatalf("unable to determine database size: %v", err)
	}
	if size == 0 {
		log.Fatalf("database is already empty; Nothing to do here")
	}

	fmt.Printf("database size: %v. Empty database? [y/N] ", size)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(input)) != "y" {
		log.Fatalf("abandoning operation at user request")
	}

	// we'll retrieve and remove the contents individually so we also show what they were
	// since we also use this as a debugging tool of sorts for the data in the database.
	keys, err := db.Keys(ctx, "*").Result()
	if err != nil {
		log.Fatalf("unable to retrieve the keys: %v", err)
	}

	for i, key := range keys {
		v, err := db.Get(ctx, key).Result()
		if err != nil {
			log.Fatalf("unable to retrieve value #%d (key \"%s\"): %v", i, key, err)
		}
		fmt.Printf("#%d: %s: %s\n", i, key, v)
		if err = db.Del(ctx, key).Err(); err != nil {
			log.Fatalf("unable to delete value #%d (key \"%s\"): %v", i, key, err)
		}
	}
}
