package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/doz-8108/visitor-counter-svc/pb"
	"github.com/doz-8108/visitor-counter-svc/utils"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// really make me nuts to create interface only for testing purpose...
type Clock interface {
	CurrentTime() time.Time
}

type HttpClient interface {
	Get(url string) (*http.Response, error)
}

type CustomClock struct {
}

func (c *CustomClock) CurrentTime() time.Time {
	return time.Now()
}

type IpToGeoCodeConfig struct {
	ApiKey            string
	TargetedCountries *hashset.Set
}

type Utils struct {
	*zap.SugaredLogger
	utils.Err
}

type Server struct {
	pb.UnimplementedVisitorCounterServiceServer
	RedisClient       *redis.Client
	IpToGeoCodeConfig IpToGeoCodeConfig
	HttpClient        HttpClient
	Clock             Clock
	Utils             Utils
}

func main() {
	godotenv.Load()
	logger := utils.SetUpLogger()
	defer logger.Sync()

	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: "",
		DB:       0,
		Protocol: 2,
	})

	ctx := context.Background()
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		logger.Fatal(err)
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	countries_strs := strings.Split(os.Getenv("TARGETED_COUNTRIES"), ",")
	countries_intfs := make([]interface{}, len(countries_strs))
	for i, v := range countries_strs {
		countries_intfs[i] = v
	}

	s := grpc.NewServer()
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Fatal(err)
	} else {
		logger.Infof("Listening at port %s", port)
	}

	pb.RegisterVisitorCounterServiceServer(s, &Server{
		RedisClient: redisClient,
		IpToGeoCodeConfig: IpToGeoCodeConfig{
			ApiKey:            os.Getenv("IP2LOCATION_API_KEY"),
			TargetedCountries: hashset.New(countries_intfs...),
		},
		HttpClient: http.DefaultClient,
		Clock:      &CustomClock{},
		Utils: Utils{
			logger,
			utils.Err{Logger: logger},
		},
	})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		logger.Fatal(err)
	}
}
