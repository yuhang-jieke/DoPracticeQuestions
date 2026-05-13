package routes

import (
	"interview-platform/handlers"
	"interview-platform/middleware"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine) {
	api := r.Group("/api")
	{
		// Auth
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
			auth.GET("/me", middleware.AuthRequired(), handlers.GetMe)
		}

		// Categories
		api.GET("/categories", handlers.GetCategories)

		// Questions
		api.GET("/questions", handlers.GetQuestions)
		api.GET("/questions/template", handlers.DownloadTemplate)
		api.POST("/questions/upload", middleware.AuthRequired(), handlers.UploadQuestions)
		api.GET("/questions/:id", handlers.GetQuestion)

		// Answers (auth required)
		api.POST("/questions/:id/answers", middleware.AuthRequired(), middleware.RateLimit(1), handlers.SubmitAnswer)
		api.GET("/questions/:id/answers", middleware.AuthRequired(), handlers.GetUserAnswer)
		api.GET("/answers/:id/history", middleware.AuthRequired(), handlers.GetAnswerHistory)

		// Top Answers
		api.GET("/questions/:id/top-answers", middleware.OptionalAuth(), handlers.GetTopAnswers)

		// Comments
		api.POST("/top-answers/:id/comments", middleware.AuthRequired(), handlers.AddComment)
		api.GET("/top-answers/:id/comments", handlers.GetComments)

		// Likes
		api.POST("/top-answers/:id/like", middleware.AuthRequired(), handlers.LikeAnswer)

		// Bookmarks
		api.POST("/questions/:id/bookmark", middleware.AuthRequired(), handlers.BookmarkQuestion)
		api.GET("/questions/:id/bookmark", middleware.AuthRequired(), handlers.CheckBookmark)

		// User Center
		user := api.Group("/user")
		user.Use(middleware.AuthRequired())
		{
			user.GET("/answers", handlers.GetUserAnswers)
			user.GET("/wrong-answers", handlers.GetWrongAnswers)
			user.GET("/bookmarks", handlers.GetUserBookmarks)
			user.GET("/stats", handlers.GetUserStats)
			user.GET("/question-scores", handlers.GetQuestionScores)
			user.GET("/uploads", handlers.GetUserUploads)
		user.GET("/ai-config", handlers.GetAIConfig)
		user.PUT("/ai-config", handlers.UpdateAIConfig)
		}
	}
}
