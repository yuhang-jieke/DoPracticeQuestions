package main

import (
	"log"

	"interview-platform/config"
	"interview-platform/database"
	"interview-platform/handlers"
	"interview-platform/middleware"
	"interview-platform/routes"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	// Init database
	if err := database.Init(cfg.DSN()); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	log.Println("数据库连接成功")

	// Init JWT
	middleware.InitJWT(cfg.JWTSecret)

	// Init category matcher
	if err := handlers.InitCategoryMatcher("data/category_map.json"); err != nil {
		log.Printf("警告: 加载分类规则失败: %v（将使用默认分类）", err)
	}

	// Init gin
	r := gin.Default()

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Routes
	routes.Setup(r)

	log.Printf("服务启动于 :%s", cfg.ServerPort)
	r.Run(":" + cfg.ServerPort)
}
