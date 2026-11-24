package main

import (
	"log"
	"time"

	"github.com/newrelic/go-agent/v3/integrations/nrgorm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai"
	_, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		Plugins: map[string]gorm.Plugin{
			nrgorm.APMPlugin{}.Name(): nrgorm.APMPlugin{},
		},
	})
	if err != nil {
		log.Printf("gorm open failed: %v", err)
	}
}
