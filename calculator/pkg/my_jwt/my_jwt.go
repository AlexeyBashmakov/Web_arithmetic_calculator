package my_jwt

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"web_arithmetic_calculator/internal/constants"
	"web_arithmetic_calculator/internal/environ_vars"
)

const hmacSampleSecret = constants.HTTP_PORT + constants.RPC_PORT // константы используем как соль

// генерируем хеш от пароля
func GenerateHash(passwd string) string {
	str := constants.HTTP_PORT + passwd + constants.RPC_PORT // константы используем как соль
	return fmt.Sprintf("%x", sha256.Sum256([]byte(str)))
}

// проверяем - равны хеши или нет
func HashesAreEqual(a, b string) bool {
	return bytes.Equal([]byte(a), []byte(b))
}

// создаём JWT-токен
func CreateJWT_token(username string) (string, error) {
	// время, через которое агент запрашивает у сервера задачу
	tokenExpired, ok := environ_vars.GetValueInt(constants.TimeTokenExpired)
	if !ok {
		tokenExpired = 100
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"name": username,
		"exp":  now.Add(time.Duration(tokenExpired) * time.Minute).Unix(), // время жизни токена в минутах
		"iat":  now.Unix(),
	})

	tokenString, err := token.SignedString([]byte(hmacSampleSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// проверяем JWT-токен на валидность и получаем из него имя пользователя
func CheckJWT_token(tokenString string) (string, error) {
	tokenFromString, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(hmacSampleSecret), nil
	})

	if err != nil {
		return "", err
	}

	username := ""
	if claims, ok := tokenFromString.Claims.(jwt.MapClaims); ok {
		if u, ok := claims["name"].(string); ok {
			username = u
		} else {
			return "", errors.New("cannot get user name from JWT-token")
		}
	} else {
		return "", err
	}

	return username, nil
}
