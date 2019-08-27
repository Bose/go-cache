package cache

import (
	"testing"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func TestRedisConnectionInfo_New(t *testing.T) {
	requestLogger := log.WithFields(log.Fields{"hi-mom": "yo!"})
	defExpSeconds := 10
	selectDatabase := 3
	sentinelURL := "redis://localhost:26379"
	redisURL := "redis://localhost:6379"
	type fields struct {
		MasterIdentifier              string
		Password                      string
		SentinelURL                   string
		RedisURL                      string
		UseSentinel                   bool
		DefaultExpSeconds             int
		ConnectionTimeoutMilliseconds int
		ReadWriteTimeoutMilliseconds  int
		SelectDatabase                int
		testDelay                     int
	}
	type args struct {
		logger *logrus.Entry
	}
	tests := []struct {
		name      string
		fields    fields
		testWrite bool
		wantErr   bool
	}{
		{"use default localhost URL", fields{
			MasterIdentifier:              "",
			Password:                      "",
			SentinelURL:                   "",
			RedisURL:                      "",
			UseSentinel:                   true,
			DefaultExpSeconds:             defExpSeconds,
			ConnectionTimeoutMilliseconds: defExpSeconds,
			ReadWriteTimeoutMilliseconds:  defExpSeconds,
			SelectDatabase:                selectDatabase,
		},
			true,
			false},
		{"use good sentinel URL", fields{
			MasterIdentifier:              "",
			Password:                      "",
			SentinelURL:                   sentinelURL,
			RedisURL:                      redisURL,
			UseSentinel:                   true,
			DefaultExpSeconds:             defExpSeconds,
			ConnectionTimeoutMilliseconds: defExpSeconds,
			ReadWriteTimeoutMilliseconds:  defExpSeconds,
			SelectDatabase:                selectDatabase,
		},
			true,
			false},
		{"use bad sentinel URL", fields{
			MasterIdentifier:              "",
			Password:                      "",
			SentinelURL:                   "localhost:26379",
			RedisURL:                      redisURL,
			UseSentinel:                   true,
			DefaultExpSeconds:             defExpSeconds,
			ConnectionTimeoutMilliseconds: defExpSeconds,
			ReadWriteTimeoutMilliseconds:  defExpSeconds,
			SelectDatabase:                selectDatabase,
		},
			true,
			true},
		{"use bad redis URL", fields{
			MasterIdentifier:              "",
			Password:                      "",
			SentinelURL:                   "",
			RedisURL:                      "localhost:6379",
			UseSentinel:                   false,
			DefaultExpSeconds:             defExpSeconds,
			ConnectionTimeoutMilliseconds: defExpSeconds,
			ReadWriteTimeoutMilliseconds:  defExpSeconds,
			SelectDatabase:                selectDatabase,
		},
			false,
			true},
		{"good redis URL", fields{
			MasterIdentifier:              "",
			Password:                      "",
			SentinelURL:                   "",
			RedisURL:                      redisURL,
			UseSentinel:                   false,
			DefaultExpSeconds:             defExpSeconds,
			ConnectionTimeoutMilliseconds: defExpSeconds,
			ReadWriteTimeoutMilliseconds:  defExpSeconds,
			SelectDatabase:                selectDatabase,
		},
			false,
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connInfo := &RedisConnectionInfo{
				MasterIdentifier:              tt.fields.MasterIdentifier,
				Password:                      tt.fields.Password,
				SentinelURL:                   tt.fields.SentinelURL,
				RedisURL:                      tt.fields.RedisURL,
				UseSentinel:                   tt.fields.UseSentinel,
				DefaultExpSeconds:             tt.fields.DefaultExpSeconds,
				ConnectionTimeoutMilliseconds: tt.fields.ConnectionTimeoutMilliseconds,
				ReadWriteTimeoutMilliseconds:  tt.fields.ReadWriteTimeoutMilliseconds,
				SelectDatabase:                tt.fields.SelectDatabase,
			}
			got, err := connInfo.New(tt.testWrite, requestLogger)
			if err != nil && tt.wantErr == false {
				t.Errorf("RedisConnectionInfo.New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err == nil) && (got == nil) {
				t.Errorf("RedisConnectionInfo.New() = %v, expected a valid connection", got)
			}
		})
	}
}
