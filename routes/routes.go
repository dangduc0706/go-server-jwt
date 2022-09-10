package routes

import (
	"github.com/gofiber/fiber/v2"
	"go-server-jwt/controllers"
)

func Setup(app *fiber.App) {
	app.Post("/api/register", controllers.Register)
	app.Post("/api/login", controllers.Login)
	app.Get("/api/user", controllers.GetUserWithToken)
	app.Post("/api/logout", controllers.LogoutWithCookie)
}
