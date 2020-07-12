package limiter

import (
	"errors"
	"sync"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"go-rate-limiter/internal/cache"
	mockCache "go-rate-limiter/internal/cache/mock"
)

var testTime = time.Now()
var testRequestWindow = time.Minute

type limiterSuite struct {
	suite.Suite
	ctrl     *gomock.Controller
	cache    *mockCache.MockCache
	nowPatch *monkey.PatchGuard

	limiter Limiter
}

func Test_limiterSuite(t *testing.T) {
	suite.Run(t, &limiterSuite{})
}

func (s *limiterSuite) SetupSuite() {
	s.nowPatch = monkey.Patch(time.Now, func() time.Time {
		return testTime
	})
}

func (s *limiterSuite) DearDownSuite() {
	if s.nowPatch != nil {
		s.nowPatch.Unpatch()
	}
}

func (s *limiterSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.cache = mockCache.NewMockCache(s.ctrl)
	s.limiter = NewTokenBucketLimiter(testRequestWindow, 60, s.cache)
}

func (s *limiterSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *limiterSuite) mockThatCacheNoData() {
	s.cache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(
			func(arg0, arg1, arg2 interface{}) error {
				return cache.ErrCacheMiss
			})
}

func (s *limiterSuite) mockThatCacheReturnBucket(currentCnt int, exp int64) {
	data := TokenBucket{
		Count:      currentCnt,
		Expiration: exp,
	}
	s.cache.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		DoAndReturn(
			func(arg0, arg1, arg2 interface{}) error {
				*arg2.(*TokenBucket) = data
				return nil
			})
}

func (s *limiterSuite) mockThatCacheSetSucc() {
	s.cache.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(nil)
}

func (s *limiterSuite) mockThatCacheSetErr() {
	s.cache.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(errors.New("something went wrong"))
}

func (s *limiterSuite) TestTake_NewRequest_CurrentCntEq1() {
	// given
	testKey := "test_key"
	expectedCnt := 1
	expectedExp := testTime.Add(time.Minute).Unix()

	s.mockThatCacheNoData()
	s.mockThatCacheSetSucc()

	// when
	currentCnt, expiration, err := s.limiter.Take(testKey)

	// then
	s.Nil(err)
	s.Equal(expectedCnt, currentCnt)
	s.Equal(expectedExp, expiration)
}

func (s *limiterSuite) TestTake_NewRequestSetCacheFail_ErrInternal() {
	// given
	testKey := "test_key"
	expectedCnt := 0
	expectedExp := int64(0)
	expectedErr := ErrInternal

	s.mockThatCacheNoData()
	s.mockThatCacheSetErr()

	// when
	currentCnt, expiration, err := s.limiter.Take(testKey)

	// then
	if err == nil {
		s.FailNow("should get an error")
	}
	s.EqualError(expectedErr, err.Error())
	s.Equal(expectedCnt, currentCnt)
	s.Equal(expectedExp, expiration)
}

func (s *limiterSuite) TestCheck_NewRequest_CurrentCntEq0() {
	// given
	testKey := "test_key"
	expectedCnt := 0
	expectedExp := testTime.Add(time.Minute).Unix()

	s.mockThatCacheNoData()

	// when
	currentCnt, expiration, err := s.limiter.Check(testKey)

	// then
	s.Nil(err)
	s.Equal(expectedCnt, currentCnt)
	s.Equal(expectedExp, expiration)
}

func (s *limiterSuite) TestTake_ValidRequest_CurrentCntMatch() {
	// given
	testKey := "test_key"
	testCases := []struct {
		currentCnt  int
		expectedCnt int
		expectedExp int64
	}{
		{
			currentCnt:  5,
			expectedCnt: 6,
			expectedExp: testTime.Add(s.limiter.GetRequestWindow()).Unix(),
		}, {
			currentCnt:  25,
			expectedCnt: 26,
			expectedExp: testTime.Add(s.limiter.GetRequestWindow()).Unix(),
		}, {
			currentCnt:  44,
			expectedCnt: 45,
			expectedExp: testTime.Add(s.limiter.GetRequestWindow()).Unix(),
		},
	}

	for _, tc := range testCases {
		s.mockThatCacheReturnBucket(tc.currentCnt, tc.expectedExp)
		s.mockThatCacheSetSucc()

		// when
		currentCnt, expiration, err := s.limiter.Take(testKey)

		// then
		s.Nil(err)
		s.Equal(tc.expectedCnt, currentCnt)
		s.Equal(tc.expectedExp, expiration)
	}
}

func (s *limiterSuite) TestCheck_ValidRequest_CurrentCntMatch() {
	// given
	testKey := "test_key"
	testCases := []struct {
		currentCnt  int
		expectedCnt int
		expectedExp int64
	}{
		{
			currentCnt:  5,
			expectedCnt: 5,
			expectedExp: testTime.Add(s.limiter.GetRequestWindow()).Unix(),
		}, {
			currentCnt:  30,
			expectedCnt: 30,
			expectedExp: testTime.Add(s.limiter.GetRequestWindow()).Unix(),
		}, {
			currentCnt:  51,
			expectedCnt: 51,
			expectedExp: testTime.Add(s.limiter.GetRequestWindow()).Unix(),
		},
	}

	for _, tc := range testCases {
		s.mockThatCacheReturnBucket(tc.currentCnt, tc.expectedExp)

		// when
		currentCnt, expiration, err := s.limiter.Check(testKey)

		// then
		s.Nil(err)
		s.Equal(tc.expectedCnt, currentCnt)
		s.Equal(tc.expectedExp, expiration)
	}
}

func (s *limiterSuite) TestTake_ExceedRequest_ErrReachLimit() {
	// given
	testKey := "test_key"
	expectedCnt := s.limiter.GetReuqestLimit()
	expectedExp := testTime.Add(s.limiter.GetRequestWindow()).Unix()
	expectedErr := ErrReachLimit

	s.mockThatCacheReturnBucket(s.limiter.GetReuqestLimit(), expectedExp)

	// when
	currentCnt, expiration, err := s.limiter.Take(testKey)

	// then
	if err == nil {
		s.FailNow("should get an error")
	}
	s.EqualError(expectedErr, err.Error())
	s.Equal(expectedCnt, currentCnt)
	s.Equal(expectedExp, expiration)
}

func (s *limiterSuite) TestCheck_ExceedRequest_CurrentCntEqRequestLimit() {
	// given
	testKey := "test_key"
	expectedCnt := s.limiter.GetReuqestLimit()
	expectedExp := testTime.Add(s.limiter.GetRequestWindow()).Unix()

	s.mockThatCacheReturnBucket(s.limiter.GetReuqestLimit(), expectedExp)

	// when
	currentCnt, expiration, err := s.limiter.Check(testKey)

	// then
	s.Nil(err)
	s.Equal(expectedCnt, currentCnt)
	s.Equal(expectedExp, expiration)
}

func (s *limiterSuite) TestTake_ConcurrentRequest_CurrentCntEqRequestCnt() {
	cache := cache.NewInMemoryCache(testRequestWindow)
	s.limiter = NewTokenBucketLimiter(testRequestWindow, 60, cache)

	// given
	testKey := "test_key"
	expectedCnt := s.limiter.GetReuqestLimit()
	expectedExp := testTime.Add(s.limiter.GetRequestWindow()).Unix()
	expectedErr := ErrReachLimit

	// when
	var wg sync.WaitGroup
	concurrentNum := 10
	wg.Add(concurrentNum)

	for i := 0; i < concurrentNum; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < int(s.limiter.GetReuqestLimit()/concurrentNum); j++ {
				_, _, _ = s.limiter.Take(testKey)
			}
		}()
	}
	wg.Wait()

	currentCnt, expiration, err := s.limiter.Take(testKey)

	// then
	if err == nil {
		s.FailNow("should get an error")
	}
	s.EqualError(expectedErr, err.Error())
	s.Equal(expectedCnt, currentCnt)
	s.Equal(expectedExp, expiration)
}

func (s *limiterSuite) TestCheck_ConcurrentRequest_CurrentCntEqRequestCnt() {
	cache := cache.NewInMemoryCache(testRequestWindow)
	s.limiter = NewTokenBucketLimiter(testRequestWindow, 60, cache)

	// given
	testKey := "test_key"
	expectedCnt := s.limiter.GetReuqestLimit()
	expectedExp := testTime.Add(s.limiter.GetRequestWindow()).Unix()

	// when
	var wg sync.WaitGroup
	concurrentNum := 10
	wg.Add(concurrentNum)

	for i := 0; i < concurrentNum; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < int(s.limiter.GetReuqestLimit()/concurrentNum); j++ {
				_, _, _ = s.limiter.Take(testKey)
			}
		}()
	}
	wg.Wait()

	currentCnt, expiration, err := s.limiter.Check(testKey)

	// then
	s.Nil(err)
	s.Equal(expectedCnt, currentCnt)
	s.Equal(expectedExp, expiration)
}
