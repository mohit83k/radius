package redisclient

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/mohit83k/radius/internal/model"
)

func TestRedisStore_Save_Success(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := &RedisStore{client: db}

	timestamp := time.Date(2025, 6, 21, 10, 0, 0, 0, time.UTC)
	record := model.AccountingRecord{
		Username:       "testuser",
		AcctStatusType: "Start",
		AcctSessionID:  "abc123",
		Timestamp:      timestamp,
	}

	expectedKey := "radius:acct:testuser:abc123:20250621T100000"
	val, _ := json.Marshal(record)

	mock.ExpectSet(expectedKey, string(val), 24*time.Hour).SetVal("OK")

	err := store.Save(context.Background(), record)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestRedisStore_Save_Failure(t *testing.T) {
	db, mock := redismock.NewClientMock()
	store := &RedisStore{client: db}

	timestamp := time.Date(2025, 6, 21, 11, 30, 0, 0, time.UTC)
	record := model.AccountingRecord{
		Username:       "failuser",
		AcctStatusType: "Stop",
		AcctSessionID:  "xyz999",
		Timestamp:      timestamp,
	}

	expectedKey := "radius:acct:failuser:xyz999:20250621T113000"
	val, _ := json.Marshal(record)

	mock.ExpectSet(expectedKey, string(val), 24*time.Hour).
		SetErr(fmt.Errorf("redis is down"))

	err := store.Save(context.Background(), record)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
