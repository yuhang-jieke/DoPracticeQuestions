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
		api.POST("/questions/upload", middleware.AuthRequired(), middleware.RequireRole("teacher", "director", "principal"), handlers.UploadQuestions)
		api.POST("/questions/preview", middleware.AuthRequired(), middleware.RequireRole("teacher", "director", "principal"), handlers.PreviewUpload)
		api.POST("/questions/import", middleware.AuthRequired(), middleware.RequireRole("teacher", "director", "principal"), handlers.ConfirmImport)
		api.GET("/categories/names", handlers.GetCategoriesForUpload)
		api.GET("/questions/:id", handlers.GetQuestion)
			api.DELETE("/questions/:id", middleware.AuthRequired(), middleware.RequireRole("teacher", "director", "principal"), handlers.DeleteQuestion)

		// Answers (auth required)
		api.POST("/questions/:id/answers", middleware.AuthRequired(), middleware.RateLimit(1), handlers.SubmitAnswer)
			api.POST("/questions/:id/answers/stream", middleware.AuthRequired(), middleware.RateLimit(1), handlers.SubmitAnswerStream)
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
			user.PUT("/password", handlers.ChangePassword)
		}

		// Classes
		class := api.Group("/classes")
		class.Use(middleware.AuthRequired(), middleware.RequireRole("teacher", "director", "principal"))
		{
			class.POST("", middleware.RequireRole("director", "principal"), handlers.CreateClass)
			class.GET("", handlers.GetClasses)
			class.PUT("/:id", handlers.UpdateClass)
			class.DELETE("/:id", middleware.RequireRole("director", "principal"), handlers.DeleteClass)
			class.POST("/:id/students", handlers.AddStudentToClass)
			class.DELETE("/:id/students/:userId", handlers.RemoveStudentFromClass)
			class.GET("/:id/students", handlers.GetClassStudents)
			class.PUT("/:id/teacher", middleware.RequireRole("director", "principal"), handlers.AssignTeacher)
			class.GET("/student-template", handlers.DownloadStudentTemplate)
			class.POST("/import-students", handlers.PreviewImportStudents)
			class.POST("/import-students/confirm", handlers.ConfirmImportStudents)
		}

		// Admin
		admin := api.Group("/admin")
		admin.Use(middleware.AuthRequired(), middleware.RequireRole("director", "principal"))
		{
			admin.GET("/users", handlers.GetUsers)
			admin.POST("/users", handlers.CreateUser)
			admin.PUT("/users/:id/role", handlers.UpdateUserRole)
			admin.PUT("/users/:id/reset-password", handlers.ResetUserPassword)
			admin.DELETE("/users/:id", middleware.PrincipalRequired(), handlers.DeleteUser)
			admin.POST("/categories", middleware.PrincipalRequired(), handlers.CreateCategory)
			admin.PUT("/categories/:id", middleware.PrincipalRequired(), handlers.UpdateCategory)
			admin.DELETE("/categories/:id", middleware.PrincipalRequired(), handlers.DeleteCategory)
		}

		// Teacher
		teacher := api.Group("/teacher")
		teacher.Use(middleware.AuthRequired(), middleware.TeacherRequired())
		{
			teacher.GET("/overview", handlers.GetTeacherOverview)
			teacher.GET("/students", handlers.GetTeacherStudents)
			teacher.GET("/students/:id/answers", handlers.GetStudentAnswers)
			teacher.GET("/hot-questions", handlers.GetHotQuestions)
			teacher.POST("/questions/:id/analyze-errors", handlers.AnalyzeQuestionErrors)
			teacher.GET("/questions/:id/answers", handlers.GetQuestionAnswers)
			teacher.GET("/category-stats", handlers.GetCategoryStats)
			teacher.GET("/categories/:id/questions", handlers.GetCategoryQuestions)
		}
	}
}
