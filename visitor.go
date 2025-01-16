package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/doz-8108/visitor-counter-svc/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	IpToGeoRespBody struct {
		IP          string  `json:"ip"`
		CountryCode string  `json:"country_code"`
		CountryName string  `json:"country_name"`
		RegionName  string  `json:"region_name"`
		CityName    string  `json:"city_name"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
		ZipCode     string  `json:"zip_code"`
		TimeZone    string  `json:"time_zone"`
		ASN         string  `json:"asn"`
		AS          string  `json:"as"`
		IsProxy     bool    `json:"is_proxy"`
	}
)

var (
	KEY_DURATION = time.Hour * 24 * 31 * 2
	ipv4Regex    = regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`)
	ipv6Regex    = regexp.MustCompile(`^([0-9a-fA-F]{1,4}:){7}([0-9a-fA-F]{1,4}|:)$`)
)

func (s *Server) IncrementVisitorCount(_ context.Context, in *pb.IncrementVisitorCountRequest) (_ *emptypb.Empty, errToBeReturn error) {
	defer s.Utils.HandleError(&errToBeReturn)

	ipAddr := in.GetIpAddr()
	isIpEmpty := strings.Trim(ipAddr, " ") == ""
	isIpValid := ipv4Regex.MatchString(ipAddr) || ipv6Regex.MatchString(ipAddr)
	if isIpEmpty || !isIpValid {
		s.Utils.CatchErrorWithCode(fmt.Errorf("invalid ip address"), codes.InvalidArgument)
	}

	ctx := context.Background()
	year, month := time.Now().Year(), int(time.Now().Month())
	currMonthKey := fmt.Sprintf("visitors:%d-%02d", year, month)

	// check the existence of that IP from set before sending API call
	pipe := s.RedisClient.TxPipeline()
	success := pipe.SAdd(ctx, currMonthKey, ipAddr)
	pipe.ExpireNX(ctx, currMonthKey, KEY_DURATION)
	_, err := pipe.Exec(ctx)
	s.Utils.CatchError(err)
	if success.Val() == 0 {
		return &emptypb.Empty{}, nil
	}

	// convert ip to geo information
	resp, err := s.HttpClient.Get(fmt.Sprintf("https://api.ip2location.io/?key=%s&ip=%s", s.IpToGeoCodeConfig.ApiKey, ipAddr))
	s.Utils.CatchError(err)
	decodedResp, err := io.ReadAll(resp.Body)
	s.Utils.CatchError(err)
	var geoInfo IpToGeoRespBody
	err = json.Unmarshal(decodedResp, &geoInfo)
	s.Utils.CatchError(err)

	// add that new ip to hyperloglog for counting
	code := "others"
	if s.IpToGeoCodeConfig.TargetedCountries.Contains(geoInfo.CountryCode) {
		code = geoInfo.CountryCode
	}
	key := fmt.Sprintf("visitors:%d-%02d:%s", year, month, code)
	pipe = s.RedisClient.TxPipeline()
	pipe.PFAdd(ctx, key, ipAddr)
	pipe.ExpireNX(ctx, key, KEY_DURATION)
	_, err = pipe.Exec(ctx)
	s.Utils.CatchError(err)

	s.Utils.Infof("New visitor %s from country: %s", ipAddr, code)
	return &emptypb.Empty{}, errToBeReturn
}

func (s *Server) GetVisitorCounts(_ context.Context, _ *emptypb.Empty) (out *pb.GetVisitorCountResponse, errToBeReturn error) {
	defer s.Utils.HandleError(&errToBeReturn)

	ctx := context.Background()
	currMonth := int(time.Now().Month())
	currMonthYear := time.Now().Year()

	var prevMonth int
	var prevMonthYear int
	if currMonth == 1 {
		prevMonth = 12
		prevMonthYear = currMonthYear - 1
	} else {
		prevMonth = currMonth - 1
		prevMonthYear = currMonthYear
	}

	keys := make([]string, 0)
	for _, key := range []string{
		fmt.Sprintf("visitors:%d-%02d:*", prevMonthYear, prevMonth),
		fmt.Sprintf("visitors:%d-%02d:*", currMonthYear, currMonth),
	} {
		keysByMonth, err := s.RedisClient.Keys(ctx, key).Result()
		s.Utils.CatchError(err)
		keys = append(keys, keysByMonth...)
	}

	script := `
        local results = {}
        for i, key in ipairs(KEYS) do
			if results[i] then
				results[i] = results[i] + tonumber(redis.call("PFCOUNT", key))
			else
				results[i] = tonumber(redis.call("PFCOUNT", key))
			end
        end
        return results
    `
	rawCounts, err := s.RedisClient.Eval(ctx, script, keys).Result()
	s.Utils.CatchError(err)

	// integer is typed as int64 in lua
	visitorCounts := make(map[string]int64)
	for i, count := range rawCounts.([]interface{}) {
		countryCode := strings.Split(keys[i], ":")[2]
		if err == nil {
			visitorCounts[countryCode] += count.(int64)
		}
	}

	return &pb.GetVisitorCountResponse{
		VisitorCounts: visitorCounts,
	}, errToBeReturn
}
