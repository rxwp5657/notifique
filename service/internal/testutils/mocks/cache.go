// Code generated by MockGen. DO NOT EDIT.
// Source: ../shared/cache/redis.go
//
// Generated by this command:
//
//	mockgen -source=../shared/cache/redis.go -destination=./internal/testutils/mocks/cache.go -package=mocks
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"
	time "time"

	cache "github.com/notifique/shared/cache"
	redis "github.com/redis/go-redis/v9"
	gomock "go.uber.org/mock/gomock"
)

// MockRedisConfigurator is a mock of RedisConfigurator interface.
type MockRedisConfigurator struct {
	ctrl     *gomock.Controller
	recorder *MockRedisConfiguratorMockRecorder
	isgomock struct{}
}

// MockRedisConfiguratorMockRecorder is the mock recorder for MockRedisConfigurator.
type MockRedisConfiguratorMockRecorder struct {
	mock *MockRedisConfigurator
}

// NewMockRedisConfigurator creates a new mock instance.
func NewMockRedisConfigurator(ctrl *gomock.Controller) *MockRedisConfigurator {
	mock := &MockRedisConfigurator{ctrl: ctrl}
	mock.recorder = &MockRedisConfiguratorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRedisConfigurator) EXPECT() *MockRedisConfiguratorMockRecorder {
	return m.recorder
}

// GetRedisUrl mocks base method.
func (m *MockRedisConfigurator) GetRedisUrl() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRedisUrl")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetRedisUrl indicates an expected call of GetRedisUrl.
func (mr *MockRedisConfiguratorMockRecorder) GetRedisUrl() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRedisUrl", reflect.TypeOf((*MockRedisConfigurator)(nil).GetRedisUrl))
}

// MockCacheRedisApi is a mock of CacheRedisApi interface.
type MockCacheRedisApi struct {
	ctrl     *gomock.Controller
	recorder *MockCacheRedisApiMockRecorder
	isgomock struct{}
}

// MockCacheRedisApiMockRecorder is the mock recorder for MockCacheRedisApi.
type MockCacheRedisApiMockRecorder struct {
	mock *MockCacheRedisApi
}

// NewMockCacheRedisApi creates a new mock instance.
func NewMockCacheRedisApi(ctrl *gomock.Controller) *MockCacheRedisApi {
	mock := &MockCacheRedisApi{ctrl: ctrl}
	mock.recorder = &MockCacheRedisApiMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCacheRedisApi) EXPECT() *MockCacheRedisApiMockRecorder {
	return m.recorder
}

// Del mocks base method.
func (m *MockCacheRedisApi) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	m.ctrl.T.Helper()
	varargs := []any{ctx}
	for _, a := range keys {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Del", varargs...)
	ret0, _ := ret[0].(*redis.IntCmd)
	return ret0
}

// Del indicates an expected call of Del.
func (mr *MockCacheRedisApiMockRecorder) Del(ctx any, keys ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx}, keys...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Del", reflect.TypeOf((*MockCacheRedisApi)(nil).Del), varargs...)
}

// Get mocks base method.
func (m *MockCacheRedisApi) Get(ctx context.Context, key string) *redis.StringCmd {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, key)
	ret0, _ := ret[0].(*redis.StringCmd)
	return ret0
}

// Get indicates an expected call of Get.
func (mr *MockCacheRedisApiMockRecorder) Get(ctx, key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCacheRedisApi)(nil).Get), ctx, key)
}

// Scan mocks base method.
func (m *MockCacheRedisApi) Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Scan", ctx, cursor, match, count)
	ret0, _ := ret[0].(*redis.ScanCmd)
	return ret0
}

// Scan indicates an expected call of Scan.
func (mr *MockCacheRedisApiMockRecorder) Scan(ctx, cursor, match, count any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Scan", reflect.TypeOf((*MockCacheRedisApi)(nil).Scan), ctx, cursor, match, count)
}

// Set mocks base method.
func (m *MockCacheRedisApi) Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Set", ctx, key, value, expiration)
	ret0, _ := ret[0].(*redis.StatusCmd)
	return ret0
}

// Set indicates an expected call of Set.
func (mr *MockCacheRedisApiMockRecorder) Set(ctx, key, value, expiration any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Set", reflect.TypeOf((*MockCacheRedisApi)(nil).Set), ctx, key, value, expiration)
}

// MockCache is a mock of Cache interface.
type MockCache struct {
	ctrl     *gomock.Controller
	recorder *MockCacheMockRecorder
	isgomock struct{}
}

// MockCacheMockRecorder is the mock recorder for MockCache.
type MockCacheMockRecorder struct {
	mock *MockCache
}

// NewMockCache creates a new mock instance.
func NewMockCache(ctrl *gomock.Controller) *MockCache {
	mock := &MockCache{ctrl: ctrl}
	mock.recorder = &MockCacheMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCache) EXPECT() *MockCacheMockRecorder {
	return m.recorder
}

// Del mocks base method.
func (m *MockCache) Del(ctx context.Context, k cache.Key) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Del", ctx, k)
	ret0, _ := ret[0].(error)
	return ret0
}

// Del indicates an expected call of Del.
func (mr *MockCacheMockRecorder) Del(ctx, k any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Del", reflect.TypeOf((*MockCache)(nil).Del), ctx, k)
}

// DelWithPrefix mocks base method.
func (m *MockCache) DelWithPrefix(ctx context.Context, prefix cache.Key) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DelWithPrefix", ctx, prefix)
	ret0, _ := ret[0].(error)
	return ret0
}

// DelWithPrefix indicates an expected call of DelWithPrefix.
func (mr *MockCacheMockRecorder) DelWithPrefix(ctx, prefix any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DelWithPrefix", reflect.TypeOf((*MockCache)(nil).DelWithPrefix), ctx, prefix)
}

// Get mocks base method.
func (m *MockCache) Get(ctx context.Context, k cache.Key) (string, error, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, k)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	ret2, _ := ret[2].(bool)
	return ret0, ret1, ret2
}

// Get indicates an expected call of Get.
func (mr *MockCacheMockRecorder) Get(ctx, k any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCache)(nil).Get), ctx, k)
}

// Set mocks base method.
func (m *MockCache) Set(ctx context.Context, k cache.Key, value string, ttl time.Duration) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Set", ctx, k, value, ttl)
	ret0, _ := ret[0].(error)
	return ret0
}

// Set indicates an expected call of Set.
func (mr *MockCacheMockRecorder) Set(ctx, k, value, ttl any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Set", reflect.TypeOf((*MockCache)(nil).Set), ctx, k, value, ttl)
}
