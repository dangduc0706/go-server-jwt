package controllers

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber/v2"
	"go-server-jwt/constant"
	"go-server-jwt/database"
	"go-server-jwt/models"
	"go-server-jwt/utils"
	"golang.org/x/crypto/bcrypt"
	"strconv"
	"strings"
	"time"
)

const SecretKey = "secret"

func Register(c *fiber.Ctx) error {
	var data map[string]string

	err := c.BodyParser(&data)
	if err != nil {
		return err
	}

	password, _ := bcrypt.GenerateFromPassword([]byte(data["password"]), 14)
	user := models.User{
		Name:     data["name"],
		Email:    data["email"],
		Password: password,
	}

	database.DB.Create(&user)

	return c.JSON(user)
}

func Login(c *fiber.Ctx) error {
	var data map[string]string

	err := c.BodyParser(&data)
	if err != nil {
		return err
	}

	var user models.User

	database.DB.Where("email = ? && name = ?", data["email"], data["name"]).First(&user)

	// Check user by id
	if user.Id == 0 {
		c.Status(fiber.StatusNotFound)
		return utils.ResponseMessage(c, "User not found")
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword(user.Password, []byte(data["password"]))
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return utils.ResponseMessage(c, "Incorrect password")
	}

	// JWT Token
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Issuer:    strconv.Itoa(int(user.Id)),
		ExpiresAt: time.Now().Add(time.Hour * 24).Unix(), // 1 day
	})

	token, e := claims.SignedString([]byte(SecretKey))

	if e != nil {
		c.Status(fiber.StatusInternalServerError)
		return utils.ResponseMessage(c, "Could not login "+e.Error())
	}

	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(time.Hour * 24),
		HTTPOnly: true,
	}

	c.Cookie(&cookie)

	return c.JSON(fiber.Map{
		"token": token,
	})
}

func GetUserWithCookie(c *fiber.Ctx) error {
	cookie := c.Cookies("jwt")

	token, err := jwt.ParseWithClaims(cookie, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return utils.ResponseMessage(c, "Unauthenticated "+err.Error())
	}

	claims := token.Claims.(*jwt.StandardClaims)

	var user models.User
	database.DB.Where("id = ?", claims.Issuer).First(&user)

	return c.JSON(user)
}

func UpdatePassword(c *fiber.Ctx) error {
	user, err := ExistUser(c)

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return utils.ResponseMessage(c, "Unauthenticated "+err.Error())
	}

	var data map[string]string // "password" : ""
	err = c.BodyParser(&data)
	if err != nil {
		return err
	}

	updatePasswordModel := models.UpdatePassword{
		Password: []byte(data[constant.Password]),
	}

	if pass := updatePasswordModel.Password; len(pass) == 0 {
		c.Status(fiber.StatusNotAcceptable)
		return utils.ResponseMessage(c, "Invalid password")
	}

	newPassword, _ := bcrypt.GenerateFromPassword(updatePasswordModel.Password, 14)

	database.DB.Model(&user).Updates(models.User{Password: newPassword})

	return c.JSON(fiber.Map{
		"user": user,
		"pass": updatePasswordModel.Password,
	})
}

func UpdateUserInfo(c *fiber.Ctx) error {
	user, err := ExistUser(c)
	if err != nil {
		return err
	}

	var data map[string]string

	err = c.BodyParser(&data)
	if err != nil {
		return err
	}

	updateUserModel := models.User{
		Name: data[constant.Name],
	}

	if name := updateUserModel.Name; len(name) == 0 {
		return utils.ResponseMessage(c, "Name is not valid")
	}

	database.DB.Model(&user).Updates(models.User{Name: updateUserModel.Name})

	return c.JSON(user)
}

func GetUserWithToken(c *fiber.Ctx) error {
	user, err := ExistUser(c)

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return utils.ResponseMessage(c, "Unauthenticated "+err.Error())
	}

	return c.JSON(user)
}

func ExistUser(c *fiber.Ctx) (models.User, error) {
	bearerToken := c.GetReqHeaders()[constant.Authorization]
	tokenRaw := strings.Split(bearerToken, constant.Bearer)[1]
	tokenRaw = strings.TrimSpace(tokenRaw)

	token, err := jwt.ParseWithClaims(tokenRaw, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})

	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return models.User{}, err
	}

	claims := token.Claims.(*jwt.StandardClaims)

	var user models.User
	database.DB.First(&user, "id = ?", claims.Issuer)
	return user, nil
}

func LogoutWithCookie(c *fiber.Ctx) error {
	cookie := fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
	}
	c.Cookie(&cookie)

	return utils.ResponseMessage(c, "LogoutWithCookie success")
}

func DeleteUser(c *fiber.Ctx) error {
	user, err := ExistUser(c)

	if err != nil {
		return utils.ResponseMessage(c, "User is not found")
	}

	database.DB.Delete(&user, user.Id)

	return utils.ResponseMessage(c, "User is deleted")
}
