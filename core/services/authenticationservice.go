package services

import (
	"fmt"
	"github.com/WilsonSayago/authBase/core"
	domain "github.com/WilsonSayago/authBase/core/domain"
	"github.com/WilsonSayago/authBase/core/port"
	"github.com/WilsonSayago/authBase/infra/config/properties"
	"github.com/WilsonSayago/initModules"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type AuthenticationService[T domain.IUserGeneric] struct {
	port         port.GenericPort[T]
	validatePort port.ValidationPort
	prop         *properties.JwtProp
}

func GetAuthenticationInstance[T domain.IUserGeneric](port port.GenericPort[T], validatePort port.ValidationPort, prop *properties.JwtProp) core.AuthenticationUseCase {
	instance := initModules.GetInstance("AuthenticationService", func() interface{} {
		return &AuthenticationService[T]{
			port:         port,
			validatePort: validatePort,
			prop:         prop,
		}
	})
	return instance.(*AuthenticationService[T])
}

type MyCustomClaims struct {
	User string `json:"user"`
	jwt.RegisteredClaims
}

func (a AuthenticationService[T]) GetToken(id string) (string, string, error) {
	claims := MyCustomClaims{
		id,
		jwt.RegisteredClaims{
			ID: id,
			// A usual scenario is to set the expiration time relative to the current time
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(a.prop.Jwt.ExpirationTime) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			//NotBefore: jwt.NewNumericDate(time.Now()),
			//Issuer:    "test",
			//Subject:   "somebody",

			//Audience:  []string{"somebody_else"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(a.prop.Jwt.SecretKey))

	if err != nil {
		fmt.Println(err)
		return "", "", err
	}

	refreshClaims := MyCustomClaims{
		id,
		jwt.RegisteredClaims{
			ID:        id,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(a.prop.Jwt.RefreshTokenTime) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(a.prop.Jwt.RefreshSecret))
	if err != nil {
		return "", "", err
	}

	return tokenString, refreshTokenString, nil
}

func (a AuthenticationService[T]) Login(username, password string) (string, string, error) {
	// Find the user by username
	user, err := a.port.FindByEmail(username)
	if err != nil {
		return "", "", fmt.Errorf("user not found: %v", err)
	}

	// Validate the password
	if !a.validatePort.CheckPassword(user.GetPassword(), password) {
		return "", "", fmt.Errorf("invalid password")
	}

	token, refreshToken, err := a.GetToken(user.GetId())
	if err != nil {
		return "", "", fmt.Errorf("failed to generate token: %v", err)
	}

	return token, refreshToken, nil

}

func (a AuthenticationService[T]) RefreshToken(refreshToken string) (string, string, error) {
	claims := &jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.prop.Jwt.RefreshSecret), nil
	})

	if err != nil || !token.Valid {
		return "", "", fmt.Errorf("invalid token")
	}

	userId := (*claims)["user"].(string)

	tokenString, refreshToken, err := a.GetToken(userId)

	if err != nil {
		return "", "", fmt.Errorf("failed to generate token: %v", err)
	}

	return tokenString, refreshToken, nil
}

func (a AuthenticationService[T]) ValidateToken(tokenString string) (domain.IUserGeneric, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.prop.Jwt.SecretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userId := claims["user"].(string)
		user, err := a.port.FindFullById(userId)
		if err != nil {
			return nil, fmt.Errorf("user not found: %v", err)
		}
		return user, nil
	} else {
		return nil, fmt.Errorf("invalid token")
	}
}

func (a AuthenticationService[T]) ValidateTokenAndRefresh() (string, error) {
	//TODO implement me
	panic("implement me")
}
