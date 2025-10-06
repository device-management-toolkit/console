package v1

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/entity/dto/v1"
	"github.com/device-management-toolkit/console/pkg/consoleerrors"
)

var ErrLogin = consoleerrors.CreateConsoleError("LoginHandler")

type LoginRoute struct {
	Config   *config.Config
	Verifier *oidc.IDTokenVerifier
}

// NewVersionRoute creates a new version route
func NewLoginRoute(configData *config.Config) *LoginRoute {
	lr := &LoginRoute{
		Config: configData,
	}

	if config.ConsoleConfig.ClientID != "" {
		provider, err := oidc.NewProvider(context.Background(), config.ConsoleConfig.Issuer)
		if err != nil {
			return nil
		}

		lr.Verifier = provider.Verifier(&oidc.Config{
			ClientID: config.ConsoleConfig.ClientID,
		})
	}

	return lr
}

// Login checks configured credentials and returns a JWT token for basic auth
func (lr LoginRoute) Login(c *gin.Context) {
	var creds dto.Credentials

	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})

		return
	}

	lr.handleBasicAuth(creds, c)
}

func (lr LoginRoute) handleBasicAuth(creds dto.Credentials, c *gin.Context) {
	if creds.Username != lr.Config.AdminUsername || creds.Password != lr.Config.AdminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})

		return
	}

	// Create JWT token
	expirationTime := time.Now().Add(config.ConsoleConfig.JWTExpiration)
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expirationTime),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(lr.Config.JWTKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create token"})

		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// JWT Middleware
func (lr LoginRoute) JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		tokenString = strings.Replace(tokenString, "Bearer ", "", 1)

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "request does not contain an access token"})
			c.Abort()

			return
		}

		// if clientID is set, use the oidc verifier
		if config.ConsoleConfig.ClientID != "" {
			_, err := lr.Verifier.Verify(c.Request.Context(), tokenString)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
				c.Abort()
			}
		} else {
			claims := &jwt.MapClaims{}

			token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (interface{}, error) {
				return []byte(lr.Config.JWTKey), nil
			})

			if err != nil || !token.Valid {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
				c.Abort()

				return
			}
		}

		c.Next()
	}
}
