package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os/exec"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/stretchr/testify/require"
)

var _ = suite("database/config/get", func(t *testing.T, when spec.G, it spec.S) {
	var (
		expect *require.Assertions
		server *httptest.Server
	)

	it.Before(func() {
		expect = require.New(t)

		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v2/databases/mysql-database-id/config":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(databaseConfigMySQLGetResponse))
			case "/v2/databases/pg-database-id/config":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(databaseConfigPGGetResponse))
			case "/v2/databases/redis-database-id/config":
				auth := req.Header.Get("Authorization")
				if auth != "Bearer some-magic-token" {
					w.WriteHeader(http.StatusTeapot)
				}

				if req.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}

				w.Write([]byte(databaseConfigRedisGetResponse))
			default:
				dump, err := httputil.DumpRequest(req, true)
				if err != nil {
					t.Fatal("failed to dump request")
				}

				t.Fatalf("received unknown request: %s", dump)
			}
		}))
	})

	when("all required flags are passed", func() {
		it("gets the mysql database config", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"configuration",
				"get",
				"--engine", "mysql",
				"mysql-database-id",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databaseConfigMySQLGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("all required flags are passed", func() {
		it("gets the pg database config", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"configuration",
				"get",
				"--engine", "pg",
				"pg-database-id",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databaseConfigPGGetOutput), strings.TrimSpace(string(output)))
		})
	})

	when("all required flags are passed", func() {
		it("gets the redis database config", func() {
			cmd := exec.Command(builtBinaryPath,
				"-t", "some-magic-token",
				"-u", server.URL,
				"database",
				"configuration",
				"get",
				"--engine", "redis",
				"redis-database-id",
			)

			output, err := cmd.CombinedOutput()
			expect.NoError(err, fmt.Sprintf("received error output: %s", output))
			expect.Equal(strings.TrimSpace(databaseConfigRedisGetOutput), strings.TrimSpace(string(output)))
		})
	})
})

const (
	databaseConfigMySQLGetOutput = `
key                            value
DefaultTimeZone                UTC
MaxAllowedPacket               67108864
SQLMode                        ANSI,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION,NO_ZERO_DATE,NO_ZERO_IN_DATE,STRICT_ALL_TABLES
SQLRequirePrimaryKey           true
InnodbFtMinTokenSize           3
InnodbFtServerStopwordTable    
InnodbPrintAllDeadlocks        false
InnodbRollbackOnTimeout        false
SlowQueryLog                   false
LongQueryTime                  10
BackupHour                     18
BackupMinute                   3
`

	databaseConfigMySQLGetResponse = `
{
	"config": {
		"default_time_zone": "UTC",
		"max_allowed_packet": 67108864,
		"sql_mode": "ANSI,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION,NO_ZERO_DATE,NO_ZERO_IN_DATE,STRICT_ALL_TABLES",
		"sql_require_primary_key": true,
		"innodb_change_buffer_max_size": 25,
		"innodb_flush_neighbors": 1,
		"innodb_ft_min_token_size": 3,
		"innodb_ft_server_stopword_table": "",
		"innodb_print_all_deadlocks": false,
		"innodb_read_io_threads": 4,
		"innodb_rollback_on_timeout": false,
		"innodb_thread_concurrency": 0,
		"innodb_write_io_threads": 4,
		"net_buffer_length": 16384,
		"slow_query_log": false,
		"long_query_time": 10,
		"backup_hour": 18,
		"backup_minute": 3
	}
}`

	databaseConfigPGGetOutput = `
key                                 value
AutovacuumNaptime                   60
AutovacuumVacuumThreshold           50
AutovacuumAnalyzeThreshold          50
AutovacuumVacuumScaleFactor         0.2
AutovacuumAnalyzeScaleFactor        0.2
AutovacuumVacuumCostDelay           20
AutovacuumVacuumCostLimit           -1
BGWriterFlushAfter                  512
BGWriterLRUMaxpages                 100
BGWriterLRUMultiplier               2
IdleInTransactionSessionTimeout     0
JIT                                 true
LogAutovacuumMinDuration            -1
LogMinDurationStatement             -1
MaxPreparedTransactions             0
MaxParallelWorkers                  8
MaxParallelWorkersPerGather         2
TempFileLimit                       -1
WalSenderTimeout                    60000
PgBouncer.ServerResetQueryAlways    false
PgBouncer.MinPoolSize               0
PgBouncer.ServerIdleTimeout         0
PgBouncer.AutodbPoolSize            0
PgBouncer.AutodbMaxDbConnections    0
PgBouncer.AutodbIdleTimeout         0
BackupHour                          18
BackupMinute                        26`

	databaseConfigPGGetResponse = `{
"config": {
	"autovacuum_naptime": 60,
	"autovacuum_vacuum_threshold": 50,
	"autovacuum_analyze_threshold": 50,
	"autovacuum_vacuum_scale_factor": 0.2,
	"autovacuum_analyze_scale_factor": 0.2,
	"autovacuum_vacuum_cost_delay": 20,
	"autovacuum_vacuum_cost_limit": -1,
	"bgwriter_flush_after": 512,
	"bgwriter_lru_maxpages": 100,
	"bgwriter_lru_multiplier": 2,
	"idle_in_transaction_session_timeout": 0,
	"jit": true,
	"log_autovacuum_min_duration": -1,
	"log_min_duration_statement": -1,
	"max_prepared_transactions": 0,
	"max_parallel_workers": 8,
	"max_parallel_workers_per_gather": 2,
	"temp_file_limit": -1,
	"wal_sender_timeout": 60000,
	"pgbouncer": {
	"server_reset_query_always": false,
	"min_pool_size": 0,
	"server_idle_timeout": 0,
	"autodb_pool_size": 0,
	"autodb_max_db_connections": 0,
	"autodb_idle_timeout": 0
	},
	"backup_hour": 18,
	"backup_minute": 26,
	"timescaledb": {},
	"stat_monitor_enable": false
}
}`

	databaseConfigRedisGetOutput = `
key                          value
RedisMaxmemoryPolicy         volatile-lru
RedisLFULogFactor            10
RedisLFUDecayTime            1
RedisSSL                     true
RedisTimeout                 600
RedisNotifyKeyspaceEvents    
RedisPersistence             rdb
RedisACLChannelsDefault      allchannels
`

	databaseConfigRedisGetResponse = `{
	"config": {
	  "redis_maxmemory_policy": "volatile-lru",
	  "redis_lfu_log_factor": 10,
	  "redis_lfu_decay_time": 1,
	  "redis_ssl": true,
	  "redis_timeout": 600,
	  "redis_notify_keyspace_events": "",
	  "redis_persistence": "rdb",
	  "redis_acl_channels_default": "allchannels"
	}
  }
`
)
