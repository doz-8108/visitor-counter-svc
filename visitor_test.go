package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/doz-8108/visitor-counter-svc/mocks"
	"github.com/doz-8108/visitor-counter-svc/pb"
	"github.com/doz-8108/visitor-counter-svc/utils"
	"github.com/emirpasic/gods/sets/hashset"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/types/known/emptypb"
)

type VisitorCounterTestSuite struct {
	suite.Suite
	redisMock       *miniredis.Miniredis
	redisClientMock *redis.Client
	serverMock      *Server
	require         *require.Assertions
	httpClientMock  *mocks.HttpClient
	clockMock       *mocks.Clock
}

func TestVisitorCounterTestSuite(t *testing.T) {
	suite.Run(t, &VisitorCounterTestSuite{})
}

func (vts *VisitorCounterTestSuite) SetupSuite() {
	t := vts.T()
	redisMock, err := miniredis.Run()
	if err != nil {
		vts.FailNow(err.Error())
	}

	redisClientMock := redis.NewClient(&redis.Options{
		Addr: redisMock.Addr(),
	})
	loggerMock := zaptest.NewLogger(t).Sugar()
	s := &Server{
		IpToGeoCodeConfig: IpToGeoCodeConfig{
			TargetedCountries: hashset.New("US", "CA"),
		},
		RedisClient: redisClientMock,
		Utils:       Utils{loggerMock, utils.Err{Logger: loggerMock}},
	}

	ctx := context.Background()
	err = redisClientMock.Ping(ctx).Err()
	if err != nil {
		vts.FailNow(err.Error())
	}

	vts.require = require.New(t)
	vts.redisMock = redisMock
	vts.redisClientMock = redisClientMock
	vts.serverMock = s
}

func (vts *VisitorCounterTestSuite) SetupTest() {
	httpClientMock := &mocks.HttpClient{}
	clockMock := &mocks.Clock{}
	vts.httpClientMock = httpClientMock
	vts.clockMock = clockMock
	vts.serverMock.HttpClient = httpClientMock
	vts.redisClientMock.FlushAll(context.Background())
}

func (vts *VisitorCounterTestSuite) TearDownSuite() {
	vts.redisClientMock.Close()
	vts.redisMock.Close()
}

func (vts *VisitorCounterTestSuite) TestIpValidation() {
	ctx := context.Background()
	for _, ipAddr := range []string{"invalid", "%$", "196.$$.abc", ""} {
		_, err := vts.serverMock.IncrementVisitorCount(ctx, &pb.IncrementVisitorCountRequest{IpAddr: ipAddr})
		vts.require.Equal("rpc error: code = InvalidArgument desc = invalid ip address", err.Error())
	}
}

func (vts *VisitorCounterTestSuite) TestDuplicateIpCheck() {
	ctx := context.Background()
	ipAddr := "192.168.0.1"
	year, month := time.Now().Year(), int(time.Now().Month())
	currMonthKey := fmt.Sprintf("visitors:%d-%02d", year, month)
	err := vts.redisClientMock.SAdd(ctx, currMonthKey, ipAddr, time.Second*10).Err()
	if err != nil {
		vts.FailNow(err.Error())
	}
	resp, err := vts.serverMock.IncrementVisitorCount(ctx, &pb.IncrementVisitorCountRequest{IpAddr: ipAddr})
	vts.require.Nil(err)
	vts.require.Equal(&emptypb.Empty{}, resp)
}

func (vts *VisitorCounterTestSuite) TestGetGeoInfoErr() {
	ctx := context.Background()
	vts.httpClientMock.On("Get", mock.Anything).Return(nil, fmt.Errorf("Error getting geo info"))
	_, err := vts.serverMock.IncrementVisitorCount(ctx, &pb.IncrementVisitorCountRequest{IpAddr: "127.0.0.1"})
	vts.require.ErrorContains(err, "Error getting geo info")
}

func (vts *VisitorCounterTestSuite) TestVisitorInsertion() {
	mockGeoInfo := IpToGeoRespBody{
		IP:          "127.0.0.1",
		CountryCode: "US",
		CountryName: "United States",
		RegionName:  "California",
		CityName:    "Mountain View",
		Latitude:    37.3861,
		Longitude:   -122.0839,
		ZipCode:     "94043",
		TimeZone:    "PST",
		ASN:         "AS15169",
		AS:          "Google LLC",
		IsProxy:     false,
	}
	jsonData, _ := json.Marshal(mockGeoInfo)
	mockGeoInfoResp := &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBuffer(jsonData)),
	}
	mockGeoInfoResp.Header.Set("Content-Type", "application/json")
	ctx := context.Background()

	vts.httpClientMock.On("Get", mock.Anything).Return(mockGeoInfoResp, nil)
	_, err := vts.serverMock.IncrementVisitorCount(ctx, &pb.IncrementVisitorCountRequest{IpAddr: "127.0.0.1"})
	vts.require.Nil(err)

	year, month := time.Now().Year(), int(time.Now().Month())
	key := fmt.Sprintf("visitors:%d-%02d:%s", year, month, mockGeoInfo.CountryCode)
	visitorCount, _ := vts.redisMock.PfCount(key)
	ttl := vts.redisMock.TTL(key)
	vts.require.Equal(1, visitorCount)
	vts.require.True(ttl > 0 && ttl <= KEY_DURATION)
}

func (vts *VisitorCounterTestSuite) TestGetVisitorCount() {
	ctx := context.Background()
	year, month := time.Now().Year(), int(time.Now().Month())

	for _, code := range []string{"US", "CA"} {
		key := fmt.Sprintf("visitors:%d-%02d:%s", year, month, code)
		err := vts.redisClientMock.PFAdd(ctx, key, "127.0.0.1").Err()
		if err != nil {
			vts.FailNow(err.Error())
		}
	}

	resp, err := vts.serverMock.GetVisitorCounts(ctx, &emptypb.Empty{})
	vts.require.Nil(err)
	vts.require.Equal(int64(1), resp.VisitorCounts["US"])
	vts.require.Equal(int64(1), resp.VisitorCounts["CA"])
}

// edge case
func (vts *VisitorCounterTestSuite) TestGetVisitorMonthJan() {
	ctx := context.Background()
	keys := []string{
		"visitors:2025-01:US",
		"visitors:2024-12:CA",
	}
	vts.clockMock.On("CurrentTime").Return(time.Date(2025, 1, 30, 1, 1, 1, 1000, time.UTC))

	for _, key := range keys {
		err := vts.redisClientMock.PFAdd(ctx, key, "127.0.0.1").Err()
		if err != nil {
			vts.FailNow(err.Error())
		}
	}

	resp, err := vts.serverMock.GetVisitorCounts(ctx, &emptypb.Empty{})
	vts.require.Nil(err)
	vts.require.Equal(int64(1), resp.VisitorCounts["US"])
	vts.require.Equal(int64(1), resp.VisitorCounts["CA"])
}
