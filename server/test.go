// package main

// import (
// 	"context"
// 	"fmt"
// 	"log" 
// 	"time"

// 	"github.com/brianvoe/gofakeit"
// 	"github.com/tywin1104/mc-gatekeeper/db"
// 	"github.com/tywin1104/mc-gatekeeper/types"
// 	"go.mongodb.org/mongo-driver/mongo"
// 	"go.mongodb.org/mongo-driver/mongo/options"
// )

// func main() {
// 	gofakeit.Seed(0)
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()
// 	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb+srv://"))
// 	if err != nil {
// 		log.Fatal("Unable to connect to mongodb: " + err.Error())
// 	}
// 	dbSvc := db.NewService(client)
// 	for n := 0; n <= 14000; n++ {
// 		fmt.Printf("inserting %d \n", n)
// 		request := types.WhitelistRequest{
// 			Username: gofakeit.Name(),
// 			Email:    gofakeit.Email(),
// 			Age:      int64(gofakeit.Number(10, 40)),
// 			Gender:   gofakeit.Gender(),
// 			Info: map[string]interface{}{
// 				"applicationText": gofakeit.HackerPhrase(),
// 			},
// 		}
// 		_, err := dbSvc.CreateRequest(request)
// 		if err != nil {
// 			fmt.Println(err)
// 		}
// 	}

// }
