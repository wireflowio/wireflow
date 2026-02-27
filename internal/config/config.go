package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Database DatabaseConfig `mapstructure:"database"`
	Dex      DexConfig      `mapstructure:"dex"`
}

type AppConfig struct {
	Listen     string        `mapstructure:"listen"`
	Name       string        `mapstructure:"name"`
	InitAdmins []AdminConfig `mapstructure:"init_admins"`
}

type AdminConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type DatabaseConfig struct {
	Driver string `mapstructure:"driver"`
	DSN    string `mapstructure:"dsn"`
}

type DexConfig struct {
	Issuer      string `mapstructure:"issuer"`
	ProviderUrl string `mapstructure:"providerUrl"`
}

var (
	GlobalConfig *Config
)

// InitConfig 加载配置文件
func InitConfig(env string) *Config {
	serverOnce.Do(func() {
		v := viper.New()
		// 1. 设置文件目录与类型
		v.AddConfigPath("./deploy")
		v.SetConfigType("yaml") // 配置文件类型

		// 2. 首先读取公共配置
		v.SetConfigName("conf")
		if err := v.ReadInConfig(); err != nil {
			panic(fmt.Errorf("Warning: base config.yaml not found: %v", err))
		}

		// 3. 读取环境特定配置并覆盖
		v.SetConfigName("config." + env)
		if err := v.MergeInConfig(); err != nil {
			// 如果文件不存在，我们保持沉默，因为我们有环境变量作为后备
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				fmt.Printf("未找到 config.%s.yaml，将完全使用环境变量", env)
			} else {
				fmt.Printf("合并配置失败: %v", err)
			}
		}

		// 4. 允许环境变量覆盖 (例如 APP_DATABASE_DSN 会覆盖配置文件)
		v.SetEnvPrefix("APP")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		if err := v.Unmarshal(&GlobalConfig); err != nil {
			panic(fmt.Errorf("解析配置文件失败: %w", err))
		}
	})
	fmt.Printf("Successfully loaded [%s] config. Database: %s\n", env, GlobalConfig.Database.DSN)
	return GlobalConfig
}
