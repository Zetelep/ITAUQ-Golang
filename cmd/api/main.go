package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/itauq-golang/internal/delivery/http/route"
	appRepo "github.com/itauq-golang/internal/repository/application"
	elRepo "github.com/itauq-golang/internal/repository/evaluationlink"
	qRepo "github.com/itauq-golang/internal/repository/questionnaire"
	respRepo "github.com/itauq-golang/internal/repository/respondent"
	tsRepo "github.com/itauq-golang/internal/repository/taskscenario"
	appUsecase "github.com/itauq-golang/internal/usecase/application"
	elUsecase "github.com/itauq-golang/internal/usecase/evaluationlink"
	pfUsecase "github.com/itauq-golang/internal/usecase/publicflow"
	qUsecase "github.com/itauq-golang/internal/usecase/questionnaire"
	tsUsecase "github.com/itauq-golang/internal/usecase/taskscenario"
	"github.com/itauq-golang/pkg/itauq"
	"github.com/itauq-golang/pkg/sus"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if present
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or failed to load; falling back to OS environment variables")
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// CORS configuration: allow frontend origins
	corsConfig := cors.Config{
		AllowOrigins:     []string{"https://itauq.site", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	r.Use(cors.New(corsConfig))

	var applications appRepo.Repository
	var questionnaires qRepo.Repository
	var taskScenarios tsRepo.Repository
	var evaluationLinks elRepo.Repository
	var respondents respRepo.Repository
	var provisioner appUsecase.Provisioner
	var db *pgxpool.Pool
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Println("WARNING: DATABASE_URL is not set; using in-memory repositories")
		applications = appRepo.NewMemoryRepository()
		questionnaires = qRepo.NewMemoryRepository()
		taskScenarios = tsRepo.NewMemoryRepository()
		evaluationLinks = elRepo.NewMemoryRepository()
		respondents = respRepo.NewMemoryRepository()
		provisioner = appUsecase.MemoryProvisioner{}
	} else {
		var err error
		db, err = pgxpool.New(context.Background(), databaseURL)
		if err != nil {
			log.Fatal("connect to Supabase Postgres: ", err)
		}
		defer db.Close()
		if err := db.Ping(context.Background()); err != nil {
			log.Fatal("ping Supabase Postgres: ", err)
		}
		applications = appRepo.NewPostgresRepository(db)
		questionnaires = qRepo.NewPostgresRepository(db)
		taskScenarios = tsRepo.NewPostgresRepository(db)
		evaluationLinks = elRepo.NewPostgresRepository(db)
		respondents = respRepo.NewPostgresRepository(db)
		provisioner = &appUsecase.SupabaseProvisioner{URL: os.Getenv("SUPABASE_URL"), ServiceRoleKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"), DB: db}
	}

	instrument, err := itauq.Load()
	if err != nil {
		log.Fatal("load itauq instrument: ", err)
	}

	susInstrument, err := sus.Load()
	if err != nil {
		log.Fatal("load sus instrument: ", err)
	}

	appUC := appUsecase.NewUsecase(applications, provisioner)
	qUC := qUsecase.NewUsecase(questionnaires)
	tsUC := tsUsecase.NewUsecase(taskScenarios, questionnaires)
	elUC := elUsecase.NewUsecase(evaluationLinks, questionnaires, os.Getenv("PUBLIC_EVALUATION_URL_BASE"))
	pfUC := pfUsecase.NewUsecase(evaluationLinks, questionnaires, taskScenarios, respondents, instrument, susInstrument)

	// Pass the db pool (may be nil) to route.Setup so middleware can look up
	// roles from the profiles table when available; otherwise middleware will
	// fall back to header-based behavior for in-memory/dev mode.
	route.Setup(r, appUC, qUC, tsUC, elUC, pfUC, instrument, susInstrument, db)
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("ITAUQ API listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
