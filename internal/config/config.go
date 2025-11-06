package config

import "os"

type Config struct {
    Port  string
    DBPath string
}

func Load() Config {
    cfg := Config{
        Port: ":8080",
        DBPath: "example-demo.db",
    }
    if v := os.Getenv("PORT"); v != "" {
        cfg.Port = v
    }
    if v := os.Getenv("DB_PATH"); v != "" {
        cfg.DBPath = v
    }
    return cfg
}