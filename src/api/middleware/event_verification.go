package middleware

import (
	"github.com/gin-gonic/gin"
)

func EventVerification() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Next()
	}
}
