package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config 全局配置。
type Config struct {
	LogLevel   string // debug / info / warn / error
	AppEnv     string
	AppPort    string
	AppBaseURL string
	WebBaseURL string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	RedisAddr     string
	RedisPassword string

	JWTSecret      string
	JWTExpireHours int

	AdminJWTSecret      string
	AdminJWTExpireHours int

	AuthProviders      []string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	Renderer     string
	ChromiumPath string

	JobQueue                 string
	JobWorkers               int
	JobPollIntervalMs        int
	JobStaleThresholdMinutes int

	Notifier     string
	ResendAPIKey string
	NotifyFrom   string

	LLMProvider     string
	DeepSeekAPIKey  string
	DeepSeekBaseURL string
	DeepSeekModel   string
	OpenAIAPIKey    string
	OpenAIModel     string

	PaymentProviders      []string
	PaymentSuccessURL     string
	PaymentCancelURL      string
	OrderReportPriceCents int
	ReportUnlockCredits   int
	StripeSecretKey       string
	StripeWebhookSecret   string

	PaymentMockEnabled bool
	MockWebhookSecret  string

	AlipayAppID      string
	AlipayPrivateKey string
	AlipayPublicKey  string
	AlipayNotifyURL  string
	AlipayReturnURL  string
	AlipayProduction bool
	PaypalClientID   string
	PaypalSecret     string
	PaypalWebhookID  string
	PaypalProduction bool

	QuotaDailyLimit int

	RateLimitEnabled       bool
	RateLimitReadingPerMin int
	RateLimitOrderPerMin   int
	RateLimitAuthPerMin    int
	RateLimitDefaultPerMin int

	AdminEmails []string

	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Bucket          string
	R2PublicBase      string
	LocalStorageDir   string
}

// Validate checks mandatory config fields and returns all missing keys.
func (c *Config) Validate() []string {
	var missing []string

	require := func(val, name string) {
		if val == "" {
			missing = append(missing, name)
		}
	}

	require(c.DBUser, "DB_USER")
	require(c.DBPassword, "DB_PASSWORD")
	require(c.DBName, "DB_NAME")
	require(c.JWTSecret, "JWT_SECRET")

	if contains(c.PaymentProviders, "stripe") {
		require(c.StripeSecretKey, "STRIPE_SECRET_KEY")
		require(c.StripeWebhookSecret, "STRIPE_WEBHOOK_SECRET")
	}
	if contains(c.PaymentProviders, "alipay") {
		require(c.AlipayAppID, "ALIPAY_APP_ID")
		require(c.AlipayPrivateKey, "ALIPAY_PRIVATE_KEY")
		require(c.AlipayPublicKey, "ALIPAY_PUBLIC_KEY")
	}
	if contains(c.PaymentProviders, "paypal") {
		require(c.PaypalClientID, "PAYPAL_CLIENT_ID")
		require(c.PaypalSecret, "PAYPAL_SECRET")
		require(c.PaypalWebhookID, "PAYPAL_WEBHOOK_ID")
	}

	require(c.DeepSeekAPIKey, "DEEPSEEK_API_KEY")

	if c.R2AccountID != "" {
		require(c.R2AccessKeyID, "R2_ACCESS_KEY_ID")
		require(c.R2SecretAccessKey, "R2_SECRET_ACCESS_KEY")
		require(c.R2Bucket, "R2_BUCKET")
	}

	return missing
}

// Load 从 .env / 环境变量加载配置。
// 若工作目录存在 .env 文件则先加载，仅补充未设置的变量，已有环境变量优先，符合 12-factor；
// 生产环境无 .env 时静默跳过，依赖容器或 systemd 注入的真实环境变量。
func Load() (*Config, error) {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			fmt.Printf("warning: failed to load .env file: %v\n", err)
		}
	}

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetDefault("LOG_LEVEL", "info")
	// 默认值
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("DB_HOST", "127.0.0.1")
	viper.SetDefault("DB_PORT", "3306")
	viper.SetDefault("DB_CHARSET", "utf8mb4")
	viper.SetDefault("JWT_EXPIRE_HOURS", 168)
	viper.SetDefault("ADMIN_JWT_EXPIRE_HOURS", 2)
	viper.SetDefault("JOB_QUEUE", "goroutine")
	viper.SetDefault("JOB_WORKERS", 4)
	viper.SetDefault("JOB_POLL_INTERVAL_MS", 1000)
	viper.SetDefault("JOB_STALE_THRESHOLD_MINUTES", 5)
	viper.SetDefault("NOTIFIER", "noop")
	viper.SetDefault("LLM_PROVIDER", "deepseek")
	viper.SetDefault("DEEPSEEK_BASE_URL", "https://api.deepseek.com/v1")
	viper.SetDefault("DEEPSEEK_MODEL", "deepseek-chat")
	viper.SetDefault("OPENAI_MODEL", "gpt-4o-mini")
	viper.SetDefault("RENDERER", "chromedp")
	viper.SetDefault("CHROMIUM_PATH", "/usr/bin/chromium")
	viper.SetDefault("QUOTA_DAILY_LIMIT", 3)
	viper.SetDefault("ORDER_REPORT_PRICE_CENTS", 999)
	viper.SetDefault("REPORT_UNLOCK_CREDITS", 30)
	viper.SetDefault("MOCK_WEBHOOK_SECRET", "dev-mock-secret")
	viper.SetDefault("RATELIMIT_ENABLED", true)
	viper.SetDefault("RATELIMIT_READING_PER_MIN", 5)
	viper.SetDefault("RATELIMIT_ORDER_PER_MIN", 10)
	viper.SetDefault("RATELIMIT_AUTH_PER_MIN", 10)
	viper.SetDefault("RATELIMIT_DEFAULT_PER_MIN", 60)

	cfg := &Config{
		LogLevel:   viper.GetString("LOG_LEVEL"),
		AppEnv:     viper.GetString("APP_ENV"),
		AppPort:    viper.GetString("APP_PORT"),
		AppBaseURL: viper.GetString("APP_BASE_URL"),
		WebBaseURL: viper.GetString("WEB_BASE_URL"),

		DBHost:     viper.GetString("DB_HOST"),
		DBPort:     viper.GetString("DB_PORT"),
		DBUser:     viper.GetString("DB_USER"),
		DBPassword: viper.GetString("DB_PASSWORD"),
		DBName:     viper.GetString("DB_NAME"),

		RedisAddr:     viper.GetString("REDIS_ADDR"),
		RedisPassword: viper.GetString("REDIS_PASSWORD"),

		JWTSecret:      viper.GetString("JWT_SECRET"),
		JWTExpireHours: viper.GetInt("JWT_EXPIRE_HOURS"),

		AdminJWTSecret:      viper.GetString("ADMIN_JWT_SECRET"),
		AdminJWTExpireHours: viper.GetInt("ADMIN_JWT_EXPIRE_HOURS"),

		AuthProviders:      splitEnv("AUTH_PROVIDERS"),
		GoogleClientID:     viper.GetString("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: viper.GetString("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  viper.GetString("GOOGLE_REDIRECT_URL"),

		Renderer:     viper.GetString("RENDERER"),
		ChromiumPath: viper.GetString("CHROMIUM_PATH"),

		QuotaDailyLimit: viper.GetInt("QUOTA_DAILY_LIMIT"),

		JobQueue:                 viper.GetString("JOB_QUEUE"),
		JobWorkers:               viper.GetInt("JOB_WORKERS"),
		JobPollIntervalMs:        viper.GetInt("JOB_POLL_INTERVAL_MS"),
		JobStaleThresholdMinutes: viper.GetInt("JOB_STALE_THRESHOLD_MINUTES"),

		Notifier:     viper.GetString("NOTIFIER"),
		ResendAPIKey: viper.GetString("RESEND_API_KEY"),
		NotifyFrom:   viper.GetString("NOTIFY_FROM"),

		LLMProvider:     viper.GetString("LLM_PROVIDER"),
		DeepSeekAPIKey:  viper.GetString("DEEPSEEK_API_KEY"),
		DeepSeekBaseURL: viper.GetString("DEEPSEEK_BASE_URL"),
		DeepSeekModel:   viper.GetString("DEEPSEEK_MODEL"),
		OpenAIAPIKey:    viper.GetString("OPENAI_API_KEY"),
		OpenAIModel:     viper.GetString("OPENAI_MODEL"),

		PaymentProviders:      splitEnv("PAYMENT_PROVIDERS"),
		PaymentSuccessURL:     viper.GetString("PAYMENT_SUCCESS_URL"),
		PaymentCancelURL:      viper.GetString("PAYMENT_CANCEL_URL"),
		OrderReportPriceCents: viper.GetInt("ORDER_REPORT_PRICE_CENTS"),
		ReportUnlockCredits:   viper.GetInt("REPORT_UNLOCK_CREDITS"),
		StripeSecretKey:       viper.GetString("STRIPE_SECRET_KEY"),

		PaymentMockEnabled: viper.GetBool("PAYMENT_MOCK_ENABLED"),
		MockWebhookSecret:  viper.GetString("MOCK_WEBHOOK_SECRET"),

		AlipayAppID:      viper.GetString("ALIPAY_APP_ID"),
		AlipayPrivateKey: viper.GetString("ALIPAY_PRIVATE_KEY"),
		AlipayPublicKey:  viper.GetString("ALIPAY_PUBLIC_KEY"),
		AlipayNotifyURL:  viper.GetString("ALIPAY_NOTIFY_URL"),
		AlipayReturnURL:  viper.GetString("ALIPAY_RETURN_URL"),
		AlipayProduction: viper.GetBool("ALIPAY_PRODUCTION"),
		PaypalClientID:   viper.GetString("PAYPAL_CLIENT_ID"),
		PaypalSecret:     viper.GetString("PAYPAL_SECRET"),
		PaypalWebhookID:  viper.GetString("PAYPAL_WEBHOOK_ID"),
		PaypalProduction: viper.GetBool("PAYPAL_PRODUCTION"),

		RateLimitEnabled:       viper.GetBool("RATELIMIT_ENABLED"),
		RateLimitReadingPerMin: viper.GetInt("RATELIMIT_READING_PER_MIN"),
		RateLimitOrderPerMin:   viper.GetInt("RATELIMIT_ORDER_PER_MIN"),
		RateLimitAuthPerMin:    viper.GetInt("RATELIMIT_AUTH_PER_MIN"),
		RateLimitDefaultPerMin: viper.GetInt("RATELIMIT_DEFAULT_PER_MIN"),

		AdminEmails:         splitEnv("ADMIN_EMAILS"),
		StripeWebhookSecret: viper.GetString("STRIPE_WEBHOOK_SECRET"),

		R2AccountID:       viper.GetString("R2_ACCOUNT_ID"),
		R2AccessKeyID:     viper.GetString("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey: viper.GetString("R2_SECRET_ACCESS_KEY"),
		R2Bucket:          viper.GetString("R2_BUCKET"),
		R2PublicBase:      viper.GetString("R2_PUBLIC_BASE"),
		LocalStorageDir:   viper.GetString("LOCAL_STORAGE_DIR"),
	}
	return cfg, nil
}

// DSN 返回 MySQL 连接串。
func (c *Config) DSN() string {
	charset := viper.GetString("DB_CHARSET")
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=true&loc=UTC",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, charset)
}

// contains 检测字符串切片是否包含目标元素。
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func splitEnv(key string) []string {
	val := viper.GetString(key)
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// IsAdminEmail 检测邮箱是否在管理员列表中（大小写不敏感）。
func (c *Config) IsAdminEmail(email string) bool {
	lower := strings.ToLower(strings.TrimSpace(email))
	for _, e := range c.AdminEmails {
		if strings.ToLower(e) == lower {
			return true
		}
	}
	return false
}
