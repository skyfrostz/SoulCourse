package middleware

import "github.com/gin-gonic/gin"

func SecurityHeaders(production bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if production || c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		contentSecurityPolicy := "default-src 'self'; img-src 'self' data: blob: https:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; font-src 'self' data: https://static.figma.com; media-src 'self' https://d8j0ntlcm91z4.cloudfront.net; frame-src 'self' https://view.officeapps.live.com; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'; manifest-src 'self'; worker-src 'self' blob:"
		if production {
			contentSecurityPolicy += "; upgrade-insecure-requests"
		}
		c.Header("Content-Security-Policy", contentSecurityPolicy)
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Permitted-Cross-Domain-Policies", "none")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	}
}
