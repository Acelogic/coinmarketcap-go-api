package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/jamespearly/loggly"
	"github.com/microcosm-cc/bluemonday"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"

	"os"
)

type CoinList struct {
	Data []struct {
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
		Rank   int    `json:"cmc_rank"`
		Quote  struct {
			USD struct {
				Price              float64 `json:"price"`
				MarketCap          float64 `json:"market_cap"`
				MarketCapDominance float64 `json:"market_cap_dominance"`
			} `json:"USD"`
		} `json:"quote"`
	} `json:"data"`
}
type DBitem struct {
	CoinRank   int     `json:"coinRank"`
	CoinName   string  `json:"coinName"`
	CoinSymbol string  `json:"coinSymbol"`
	CoinPrice  float64 `json:"coinPrice"`
}
type DBStats struct {
	DBName  string `json:"TableName"`
	DBCount *int64 `json:"ItemCount"`
}

func checkEnv() {
	if len(strings.TrimSpace(os.Getenv("AWS_ACCESS_KEY_ID"))) == 0 {
		fmt.Println("\nNO AWS KEY ID LOADED")
	} else {
		fmt.Println("AWS_ACCESS_KEY_ID: " + os.Getenv("AWS_ACCESS_KEY_ID"))

	}

	if len(strings.TrimSpace(os.Getenv("AWS_SECRET_ACCESS_KEY"))) == 0 {
		fmt.Println("\nNO AWS SECRET LOADED")
	} else {
		fmt.Println("AWS_SECRET_ACCESS_KEY: " + os.Getenv("AWS_SECRET_ACCESS_KEY"))

	}

	if len(strings.TrimSpace(os.Getenv("LOGGLY_TOKEN"))) == 0 {
		fmt.Println("\nNO LOGGY TOKEN LOADED")
	} else {
		fmt.Println("LOGGLY TOKEN: " + os.Getenv("LOGGLY_TOKEN"))
	}

	if len(strings.TrimSpace(os.Getenv("COINMARKETCAP_API_KEY"))) == 0 {
		fmt.Println("\nNO COINMARKETCAP_API_KEY LOADED")
	} else {
		fmt.Println("COINMARKETCAP_API_KEY: " + os.Getenv("COINMARKETCAP_API_KEY"))
	}
}

func all(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var tag string
	client := loggly.New(tag)

	client.EchoSend("info", "HTTP REQUEST RECEIVED: "+string(req.Method))

	client.EchoSend("info", "REQUEST PATH: /all")

	ip := req.RemoteAddr
	client.EchoSend("info", "SourceIP:"+ip)

	w.Write(generateAllDBJson())
}

func status(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var tag string
	client := loggly.New(tag)

	client.EchoSend("info", "HTTP REQUEST RECEIVED: "+string(req.Method))

	client.EchoSend("info", "REQUEST PATH: /status")

	ip := req.RemoteAddr
	client.EchoSend("info", "SourceIP:"+ip)

	w.Write(generateDBStatus())
}

func search(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var tag string
	client := loggly.New(tag)

	client.EchoSend("info", "HTTP REQUEST RECEIVED: "+string(req.Method))
	client.EchoSend("info", "REQUEST PATH: /search")

	sanitizer := bluemonday.StrictPolicy()

	query := req.URL.Query()

	coinName := sanitizer.Sanitize(query.Get("coinName"))

	fmt.Println("Search Term Entered:" + coinName)

	if coinName == "" || len(query) == 0 {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(400)
		w.Write([]byte("Status 400:  Bad Request"))
		return
	}

	

	searchedDB := []DBitem{}

	err := json.Unmarshal([]byte(generateAllDBJson()), &searchedDB)

	if err != nil {
		fmt.Println(err)
	}

	results := []DBitem{}

	for _, item := range searchedDB {
		if strings.ToLower(item.CoinName) == coinName || strings.ToUpper(item.CoinName) == coinName || item.CoinName == coinName {
			results = append(results, item)
		}
		if strings.ToLower(item.CoinSymbol) == coinName || strings.ToUpper(item.CoinSymbol) == coinName || item.CoinSymbol == coinName {
			results = append(results, item)
		}

	}
	if len(results) == 0 {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(404)
		w.Write([]byte("404 Search Query Not Found"))
	} else {
		client.EchoSend("info", "RETRIVED: "+coinName+" FROM SEARCH")
		w.Write(generateSearchJson(results))
	}

}

func DumpDB() []DBitem {
	// Start AWS Session in us east
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")}))

	// Create DynamoDB client
	svc := dynamodb.New(sess)
	_ = svc

	//Using Scan API and Query Projections to scan the DB
	projection := expression.NamesList(expression.Name("coinRank"), expression.Name("coinName"), expression.Name("coinSymbol"), expression.Name("coinPrice"))
	expr, err := expression.NewBuilder().WithProjection(projection).Build()
	if err != nil {
		fmt.Println("Got error building expression:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Build the query input parameters
	params := &dynamodb.ScanInput{
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ProjectionExpression:      expr.Projection(),
		TableName:                 aws.String("mcruz-CoinMarketCap"),
	}

	// Make the DynamoDB Query API call
	result, err := svc.Scan(params)
	if err != nil {
		fmt.Println("Query API call failed:")
		fmt.Println((err.Error()))
		os.Exit(1)
	}

	DB := make([]DBitem, 0)

	for _, i := range result.Items {
		item := DBitem{}
		err = dynamodbattribute.UnmarshalMap(i, &item)
		if err != nil {
			fmt.Println("Got error unmarshalling:")
			fmt.Println(err.Error())
			os.Exit(1)
		}

		DB = append(DB, item)

	}
	return DB
}

func generateSearchJson(SearchResults []DBitem) []byte {
	var buff bytes.Buffer
	buff.WriteByte('[')
	for i, item := range SearchResults {
		marsh, _ := json.Marshal(item)
		if i != 0 {
			buff.WriteByte(',')
		}
		buff.Write(marsh)
	}

	buff.WriteByte(']')
	return buff.Bytes()
}

func generateAllDBJson() []byte {
	// Generate Json
	var buff bytes.Buffer
	buff.WriteByte('[')
	for i, item := range DumpDB() {

		marsh, _ := json.Marshal(item)

		if i != 0 {
			buff.WriteByte(',')
		}
		buff.Write(marsh)
	}
	buff.WriteByte(']')

	return buff.Bytes()
}

func generateDBStatus() []byte {
	// Start AWS Session in us east
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String("us-east-1")}))

	// Create DynamoDB client
	svc := dynamodb.New(sess)
	_ = svc

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("mcruz-CoinMarketCap"),
	}

	result, _ := svc.DescribeTable(input)

	var Stat = DBStats{DBName: *result.Table.TableName, DBCount: result.Table.ItemCount}

	marshalledData, _ := json.Marshal(&Stat)

	return marshalledData

}

func main() {
	checkEnv()
	gMux := mux.NewRouter()
	gMux.HandleFunc("/mcruz/all", all).Methods("GET")
	gMux.HandleFunc("/mcruz/status", status).Methods("GET")
	gMux.HandleFunc("/mcruz/search", search).Methods("GET")

	log.Println("Listening...")
	log.Fatal(http.ListenAndServe(":8080", gMux))

}
