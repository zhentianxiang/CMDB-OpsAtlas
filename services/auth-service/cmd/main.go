package main

import (
	"cmdb-v2/pkg/logging"
	"cmdb-v2/services/auth-service/internal/handlers"
	"cmdb-v2/services/auth-service/internal/models"
	"cmdb-v2/services/auth-service/internal/router"
	"cmdb-v2/services/auth-service/internal/service"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	if err := logging.Init("auth-service"); err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}

	// 1. 连接主数据库 (cmdb_auth)
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:rootpassword@tcp(mysql:3306)/cmdb_auth?charset=utf8mb4&parseTime=True&loc=Local"
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// 2. 连接资源数据库 (cmdb_resource) 用于统一存储审计日志
	resourceDSN := os.Getenv("RESOURCE_DB_DSN")
	if resourceDSN == "" {
		resourceDSN = "root:rootpassword@tcp(mysql:3306)/cmdb_resource?charset=utf8mb4&parseTime=True&loc=Local"
	}
	dbResource, err := gorm.Open(mysql.Open(resourceDSN), &gorm.Config{})
	if err != nil {
		log.Printf("Warning: failed to connect resource database for audit: %v", err)
		dbResource = db // 如果连接失败，回退到主库，确保服务不挂
	}

	// 自动迁移所有模型表
	db.AutoMigrate(&models.User{}, &models.Dept{}, &models.Role{}, &models.Menu{}, &models.RoleMenu{})
	dbResource.AutoMigrate(&models.AuditLog{})

	// 预置初始数据
	var deptCount int64
	db.Model(&models.Dept{}).Count(&deptCount)
	if deptCount == 0 {
		db.Create(&models.Dept{Name: "总公司", ParentID: 0, Sort: 1, Principal: "CEO", Status: 1})
		db.Create(&models.Dept{Name: "运维部", ParentID: 1, Sort: 1, Principal: "运维总监", Status: 1})
		db.Create(&models.Dept{Name: "开发部", ParentID: 1, Sort: 2, Principal: "开发总监", Status: 1})
	}

	var roleCount int64
	db.Model(&models.Role{}).Count(&roleCount)
	if roleCount == 0 {
		roles := []models.Role{
			{Name: "超级管理员", Code: "admin", Sort: 1, Status: 1, Remark: "拥有系统所有操作权限"},
			{Name: "资产管理员", Code: "asset_mgr", Sort: 2, Status: 1, Remark: "负责 CMDB 资产管理，可增删改主机/应用等"},
			{Name: "应用开发运维", Code: "dev_ops", Sort: 3, Status: 1, Remark: "负责具体应用的运维，可管理关联的域名和依赖"},
			{Name: "安全审计员", Code: "auditor", Sort: 4, Status: 1, Remark: "拥有全局只读权限及操作日志查看权限"},
			{Name: "普通用户", Code: "common", Sort: 5, Status: 1, Remark: "拥有基本的资产查看权限"},
		}
		for _, r := range roles {
			db.Create(&r)
		}
		log.Println("Initialized default system roles")
	}

	if err := service.SeedRBACData(db); err != nil {
		log.Fatalf("failed to seed rbac data: %v", err)
	}

	var adminUser models.User
	err = db.Where("username = ?", "admin").First(&adminUser).Error
	if err == gorm.ErrRecordNotFound {
		hashedPassword, err := service.HashPassword("admin123")
		if err == nil {
			var firstDept models.Dept
			db.First(&firstDept) // 获取第一个部门 (总公司)

			db.Create(&models.User{
				Username: "admin",
				Password: hashedPassword,
				Role:     "admin",
				Nickname: "管理员",
				Status:   1,
				DeptID:   firstDept.ID, // 分配有效部门 ID
			})
			log.Println("Created default admin user: admin / admin123")
		}
	} else if err != nil {
		log.Printf("Error checking admin user: %v", err)
	}

	avatarDir := os.Getenv("AVATAR_DIR")
	if avatarDir == "" {
		avatarDir = "/app/uploads/avatars"
	}
	if err := os.MkdirAll(avatarDir, 0o755); err != nil {
		log.Fatalf("failed to create avatar directory: %v", err)
	}

	// 3. 初始化 Service 和 Handler
	userService := service.NewUserService(db)
	systemService := service.NewSystemService(db)
	// 审计专用 Service 指向资源数据库
	auditService := service.NewSystemService(dbResource)

	authHandler := handlers.NewAuthHandler(userService, systemService, avatarDir)
	systemHandler := handlers.NewSystemHandler(systemService)
	// 关键：将 systemHandler 的审计查询指向 auditService (即 dbResource)
	systemHandler.SetAuditService(auditService)

	// 4. 设置路由
	r := gin.Default()
	r.Static("/api/v1/auth/avatars", avatarDir)
	// 中间件使用 dbResource 写入日志，确保所有日志统一存放在 cmdb_resource
	router.InitRouter(r, authHandler, systemHandler, dbResource)

	// 5. 启动服务
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Auth Service starting on :%s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
