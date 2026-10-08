package main

import (
	"fmt"
	"log"

	"trueone-anubis/config"
	"trueone-anubis/internal/router"
	"trueone-anubis/internal/service"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("           ⚡ TrueOne Harness - Anubis ⚡          ")
	fmt.Println("             Next-Gen High Speed Backend          ")
	fmt.Println("==================================================")

	// 1. Load Configuration
	cfgPath := "config/config.yaml"
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config from %s: %v", cfgPath, err)
	}
	fmt.Printf("✅ Config loaded: listening on port %d\n", cfg.Server.Port)

	// 2. Initialize MySQL Database Connection
	db, err := config.InitDB()
	if err != nil {
		log.Fatalf("❌ Failed to connect to MySQL database: %v", err)
	}
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
	}
	fmt.Printf("✅ Database connected: %s:%d/%s (MySQL 3307)\n",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// 3. System Bootstrap & Admin Account Initialization
	service.BootstrapSystem()

	// 4. Setup Routes
	r := router.SetupRouter()

	// 5. Start Server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("🚀 Anubis server started successfully at http://localhost%s\n", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}
