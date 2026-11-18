package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Deployment    Deployment    `mapstructure:"deployment"`
	Server        Server        `mapstructure:"server"`
	DB            DB            `mapstructure:"db"`
	Redis         Redis         `mapstructure:"redis"`
	Auth          Auth          `mapstructure:"auth"`
	Message       Message       `mapstructure:"message"`
	MQ            MQ            `mapstructure:"mq"`
	Cache         Cache         `mapstructure:"cache"`
	Search        Search        `mapstructure:"search"`
	Storage       Storage       `mapstructure:"storage"`
	Observability Observability `mapstructure:"observability"`
	RTC           RTC           `mapstructure:"rtc"`
	Log           Log           `mapstructure:"log"`
}

type Deployment struct {
	Profile  string `mapstructure:"profile"`  // lite | standard
	Mode     string `mapstructure:"mode"`     // singleNode | cluster
	Services string `mapstructure:"services"` // monolith | microservice
}

type Server struct {
	HTTPAddr string `mapstructure:"http_addr"`
	WSAddr   string `mapstructure:"ws_addr"`
	Mode     string `mapstructure:"mode"`
}

type DB struct {
	Driver                string `mapstructure:"driver"`         // 元数据库：mysql
	MessageDriver         string `mapstructure:"message_driver"` // 消息库：mysql | scylla（standard 默认 scylla）
	DSN                   string `mapstructure:"dsn"`
	MaxOpenConns          int    `mapstructure:"max_open_conns"`
	MaxIdleConns          int    `mapstructure:"max_idle_conns"`
	Shards                int    `mapstructure:"shards"`
	Replicas              int    `mapstructure:"replicas"`
	MessageSharedInstance bool   `mapstructure:"messageSharedInstance"`
	Scylla                Scylla `mapstructure:"scylla"`
}

// Scylla 消息库（standard 形态）：宽表存储，按 conv_id 分区、seq 聚类
type Scylla struct {
	Hosts             []string `mapstructure:"hosts"`
	Keyspace          string   `mapstructure:"keyspace"`
	Username          string   `mapstructure:"username"`
	Password          string   `mapstructure:"password"`
	Timeout           string   `mapstructure:"timeout"`
	ReplicationFactor int      `mapstructure:"replication_factor"`
}

type Redis struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	Cluster  bool   `mapstructure:"cluster"`
	AOF      bool   `mapstructure:"aof"`
}

type Auth struct {
	JWTSecret       string        `mapstructure:"jwt_secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	VerifyCodeTTL   time.Duration `mapstructure:"verify_code_ttl"`
}

type Message struct {
	HistoryLimit  int `mapstructure:"history_limit"`
	RetentionDays int `mapstructure:"retention_days"`
}

// MQ 消息队列（standard: NATS JetStream；lite: 不启用，本地直推）
type MQ struct {
	Driver    string `mapstructure:"driver"` // jetstream | nats | kafka
	URL       string `mapstructure:"url"`    // 例：nats://nats:4222
	Stream    string `mapstructure:"stream"` // JetStream 流名
	Replicas  int    `mapstructure:"replicas"`
	Retention string `mapstructure:"retention"`
	Brokers   string `mapstructure:"brokers"` // 兼容旧配置（kafka 形态）
}

// Cache 缓存/路由（standard: Redis Cluster）
type Cache struct {
	Driver  string `mapstructure:"driver"`
	Cluster bool   `mapstructure:"cluster"`
	AOF     bool   `mapstructure:"aof"`
}

// Search 检索（standard/lite 同引擎 Meilisearch；standard 多副本）
type Search struct {
	Driver   string `mapstructure:"driver"`
	Replicas int    `mapstructure:"replicas"`
	URL      string `mapstructure:"url"`
	APIKey   string `mapstructure:"api_key"`
}

// Storage 对象存储（standard: rustfs/s3；lite: localFS）
type Storage struct {
	Driver    string `mapstructure:"driver"`
	Bucket    string `mapstructure:"bucket"`
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Region    string `mapstructure:"region"`
	Path      string `mapstructure:"path"`
	LocalDir  string `mapstructure:"local_dir"`
}

// Observability 可观测（standard: prometheus + jaeger + elk）
type Observability struct {
	Metrics       string  `mapstructure:"metrics"`
	TracingSample float64 `mapstructure:"tracingSample"`
	Jaeger        bool    `mapstructure:"jaeger"`
	ELK           bool    `mapstructure:"elk"`
	Endpoint      string  `mapstructure:"endpoint"`
}

// RTC 实时音视频（WebRTC 信令 + ICE Servers；standard: SFU 集群与 TURN）
type RTC struct {
	SFUNodes       int      `mapstructure:"sfuNodes"`
	ClusterEnabled bool     `mapstructure:"clusterEnabled"`
	CoturnAddr     string   `mapstructure:"coturnAddr"`
	STUNServers    []string `mapstructure:"stunServers"`
	TurnURL        string   `mapstructure:"turnUrl"`
	TurnUsername   string   `mapstructure:"turnUsername"`
	TurnPassword   string   `mapstructure:"turnPassword"`
}

type Log struct {
	Level    string `mapstructure:"level"`
	Encoding string `mapstructure:"encoding"`
}

var C Config

// 环境变量显式绑定，确保容器注入生效
var envKeys = []string{
	"deployment.profile", "deployment.mode", "deployment.services",
	"db.dsn", "db.driver", "db.shards", "db.replicas", "db.message_driver",
	"redis.addr", "redis.password", "redis.cluster", "cache.cluster",
	"mq.driver", "mq.url", "mq.stream", "mq.brokers", "search.driver", "search.url",
	"storage.driver", "storage.bucket", "storage.endpoint", "storage.access_key", "storage.secret_key", "storage.region",
	"observability.metrics",
	"rtc.coturnAddr", "rtc.turnUrl", "rtc.turnUsername", "rtc.turnPassword",
	"auth.jwt_secret", "server.http_addr", "server.ws_addr", "log.level",
}

func Load(path string) error {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	for _, k := range envKeys {
		_ = viper.BindEnv(k)
	}
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	// ScyllaDB 地址支持逗号分隔的环境变量（容器编排注入）
	if v := os.Getenv("SCYLLA_HOSTS"); v != "" {
		viper.Set("db.scylla.hosts", strings.Split(v, ","))
	}
	// STUN 列表支持逗号分隔的环境变量
	if v := os.Getenv("STUN_SERVERS"); v != "" {
		viper.Set("rtc.stunServers", strings.Split(v, ","))
	}
	setDefaults()
	if err := viper.Unmarshal(&C); err != nil {
		return err
	}
	// 运行时派生：按 profile 调整缺省行为
	applyProfileDerived()
	// 兼容旧配置：对象存储已由 MinIO 切换为 S3 兼容的 RustFS
	if strings.EqualFold(C.Storage.Driver, "minio") {
		C.Storage.Driver = "rustfs"
	}
	if C.DB.MessageDriver == "" {
		C.DB.MessageDriver = "mysql"
	}
	return Validate()
}

func setDefaults() {
	viper.SetDefault("deployment.profile", "lite")
	viper.SetDefault("deployment.mode", "singleNode")
	viper.SetDefault("deployment.services", "monolith")
	viper.SetDefault("db.driver", "mysql")
	viper.SetDefault("db.message_driver", "")
	viper.SetDefault("db.scylla.keyspace", "nexim")
	viper.SetDefault("db.scylla.timeout", "5s")
	viper.SetDefault("db.scylla.replication_factor", 1)
	viper.SetDefault("db.shards", 1)
	viper.SetDefault("db.replicas", 1)
	viper.SetDefault("db.messageSharedInstance", true)
	viper.SetDefault("db.max_open_conns", 20)
	viper.SetDefault("db.max_idle_conns", 5)
	viper.SetDefault("cache.driver", "redis")
	viper.SetDefault("mq.driver", "nats")
	viper.SetDefault("mq.url", "")
	viper.SetDefault("mq.stream", "IM_EVENT")
	viper.SetDefault("mq.replicas", 1)
	viper.SetDefault("mq.retention", "7d")
	viper.SetDefault("search.driver", "meilisearch")
	viper.SetDefault("search.replicas", 1)
	viper.SetDefault("storage.driver", "localFS")
	viper.SetDefault("storage.local_dir", "data/files")
	viper.SetDefault("storage.region", "us-east-1")
	viper.SetDefault("observability.metrics", "victoriametrics")
	viper.SetDefault("observability.tracingSample", 0.1)
	viper.SetDefault("rtc.sfuNodes", 1)
	viper.SetDefault("rtc.clusterEnabled", false)
	viper.SetDefault("rtc.stunServers", []string{"stun:stun.l.google.com:19302", "stun:stun1.l.google.com:19302"})
	viper.SetDefault("message.history_limit", 200)
	viper.SetDefault("message.retention_days", 90)
	viper.SetDefault("server.mode", "release")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.encoding", "json")
}

// envProvided 判断某配置项是否被环境变量显式指定
func envProvided(key string) bool {
	_, ok := os.LookupEnv(strings.ToUpper(strings.ReplaceAll(key, ".", "_")))
	return ok
}

// applyProfileDerived 按 profile 派生运行时参数（仅当配置/环境未显式覆盖时）
func applyProfileDerived() {
	if !IsStandard() {
		return
	}
	// 标准版：分片、副本、组件按标准形态设定；显式配置或环境变量优先
	if C.DB.Shards == 1 && !envProvided("db.shards") {
		C.DB.Shards = 16
	}
	if C.Search.Replicas == 1 && !envProvided("search.replicas") {
		C.Search.Replicas = 3
	}
	if C.Cache.Driver == "" {
		C.Cache.Driver = "redis"
	}
	if !envProvided("cache.cluster") {
		C.Cache.Cluster = true
	}
	// 标准版统一使用 NATS JetStream 作为事件总线
	if (C.MQ.Driver == "nats" || C.MQ.Driver == "kafka" || C.MQ.Driver == "") && !envProvided("mq.driver") {
		C.MQ.Driver = "jetstream"
	}
	if C.MQ.Stream == "im-event" || C.MQ.Stream == "" {
		C.MQ.Stream = "IM_EVENT"
	}
	if C.MQ.URL == "" && !envProvided("mq.url") {
		C.MQ.URL = "nats://nats:4222"
	}
	if C.Storage.Driver == "localFS" && !envProvided("storage.driver") {
		C.Storage.Driver = "rustfs"
	}
	if C.Observability.Metrics == "victoriametrics" && !envProvided("observability.metrics") {
		C.Observability.Metrics = "prometheus"
	}
	if C.DB.MaxOpenConns == 20 && !envProvided("db.max_open_conns") {
		C.DB.MaxOpenConns = 100
	}
	// 标准版消息库独立部署在 ScyllaDB（宽表，按会话分区）
	if C.DB.MessageDriver == "" && !envProvided("db.message_driver") {
		C.DB.MessageDriver = "scylla"
	}
	if len(C.DB.Scylla.Hosts) == 0 && !envProvided("db.scylla.hosts") && os.Getenv("SCYLLA_HOSTS") == "" {
		C.DB.Scylla.Hosts = []string{"scylla:9042"}
	}
	if C.DB.Scylla.ReplicationFactor <= 0 {
		C.DB.Scylla.ReplicationFactor = 1
	}
}

func IsLite() bool {
	return strings.EqualFold(C.Deployment.Profile, "lite") || C.Deployment.Profile == ""
}
func IsStandard() bool { return strings.EqualFold(C.Deployment.Profile, "standard") }
func IsCluster() bool {
	return IsStandard() && (strings.EqualFold(C.Deployment.Mode, "cluster") || C.Deployment.Mode == "")
}

// Validate 校验 profile 与关键配置合法性
func Validate() error {
	p := strings.ToLower(C.Deployment.Profile)
	if p != "" && p != "lite" && p != "standard" {
		return fmt.Errorf("invalid deployment.profile %q (lite|standard)", C.Deployment.Profile)
	}
	m := strings.ToLower(C.Deployment.Mode)
	if m != "" && m != "singlenode" && m != "cluster" {
		return fmt.Errorf("invalid deployment.mode %q (singleNode|cluster)", C.Deployment.Mode)
	}
	if strings.EqualFold(C.MQ.Driver, "jetstream") && C.MQ.URL == "" {
		return fmt.Errorf("mq.url is required when mq.driver=jetstream")
	}
	switch strings.ToLower(C.DB.MessageDriver) {
	case "mysql":
	case "scylla":
		if len(C.DB.Scylla.Hosts) == 0 {
			return fmt.Errorf("db.scylla.hosts is required when db.message_driver=scylla")
		}
	default:
		return fmt.Errorf("invalid db.message_driver %q (mysql|scylla)", C.DB.MessageDriver)
	}
	if C.DB.DSN == "" {
		return fmt.Errorf("db.dsn is required")
	}
	if C.Redis.Addr == "" {
		return fmt.Errorf("redis.addr is required")
	}
	return nil
}

// Summary 返回部署形态摘要，供启动日志输出
func Summary() string {
	return fmt.Sprintf("profile=%s mode=%s services=%s mq=%s cache=%s(cluster=%v) db=%s(shards=%d,replicas=%d) msgdb=%s search=%s(x%d) storage=%s metrics=%s",
		C.Deployment.Profile, C.Deployment.Mode, C.Deployment.Services,
		C.MQ.Driver, C.Cache.Driver, C.Cache.Cluster,
		C.DB.Driver, C.DB.Shards, C.DB.Replicas, C.DB.MessageDriver,
		C.Search.Driver, C.Search.Replicas, C.Storage.Driver, C.Observability.Metrics)
}
