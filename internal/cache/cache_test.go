package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

const testExpiryTime = time.Duration(1) * time.Millisecond

type cacheSuite struct {
	suite.Suite
	cache Cache
}

func Test_cacheSuite(t *testing.T) {
	suite.Run(t, &cacheSuite{})
}

func (s *cacheSuite) SetupTest() {
	s.cache = NewInMemoryCache(time.Minute)
}

func (s *cacheSuite) TestGet_CacheMiss_ErrCacheMiss() {
	// given
	ctx := context.TODO()
	expectedErr := ErrCacheMiss

	// when
	value := ""
	err := s.cache.Get(ctx, "test_key", &value)

	// then
	if err == nil {
		s.FailNow("should got an error")
	}
	s.EqualError(expectedErr, err.Error())
}

func (s *cacheSuite) TestSet_SetCache_NoErr() {
	// given
	ctx := context.TODO()

	// when
	value := "test_value"
	err := s.cache.Set(ctx, "test_key", value, testExpiryTime)

	// then
	s.Nil(err)
}

func (s *cacheSuite) TestGetAndSet_CacheHit_NoErr() {
	// given
	ctx := context.TODO()
	cacheKey := "test_key"
	expected := "test_value"

	// when
	err := s.cache.Set(ctx, cacheKey, expected, testExpiryTime)
	s.Nil(err)

	// then
	value := ""
	err = s.cache.Get(ctx, cacheKey, &value)
	s.Nil(err)
	s.Equal(expected, value)
}

func (s *cacheSuite) TestGetAndSet_CacheExpired_ErrCacheMiss() {
	// given
	ctx := context.TODO()
	cacheKey := "test_key"
	cacheValue := "test_value"
	expectedErr := ErrCacheMiss

	// when
	err := s.cache.Set(ctx, cacheKey, cacheValue, testExpiryTime)
	s.Nil(err)
	time.Sleep(testExpiryTime)
	value := ""
	err = s.cache.Get(ctx, cacheKey, &value)

	// then
	if err == nil {
		s.FailNow("should got an error")
	}
	s.EqualError(expectedErr, err.Error())
}

func (s *cacheSuite) TestGetAndSet_CacheCustomeStruct_NoErr() {
	// given
	ctx := context.TODO()
	cacheKey := "test_key"
	type customObject struct {
		ID  int
		Val []string
	}
	expected := customObject{
		ID:  1234,
		Val: []string{"hello", "word", "!"},
	}

	// when
	err := s.cache.Set(ctx, cacheKey, expected, testExpiryTime)
	s.Nil(err)

	// then
	value := customObject{}
	err = s.cache.Get(ctx, cacheKey, &value)
	s.Nil(err)
	s.Equal(expected, value)
}
