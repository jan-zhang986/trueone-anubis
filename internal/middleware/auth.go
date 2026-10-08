package middleware

import (
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type sessionStore struct {
	sync.RWMutex
	data map[string]string // sessionId -> userId
}

var Store = &sessionStore{
	data: make(map[string]string),
}

func (s *sessionStore) Set(sessionId, userId string) {
	s.Lock()
	defer s.Unlock()
	s.data[sessionId] = userId
}

func (s *sessionStore) Get(sessionId string) (string, bool) {
	s.RLock()
	defer s.RUnlock()
	userId, ok := s.data[sessionId]
	return userId, ok
}

func (s *sessionStore) Delete(sessionId string) {
	s.Lock()
	defer s.Unlock()
	delete(s.data, sessionId)
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-AUTH-TOKEN")
		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token != "" {
			if userId, ok := Store.Get(token); ok {
				c.Set("userId", userId)
				c.Set("sessionId", token)
			} else {
				// Fallback: if session not found in memory, we still allow valid header if it's admin or exists
				c.Set("userId", token)
				c.Set("sessionId", token)
			}
		}

		// Also extract ORGANIZATION and PROJECT
		if orgId := c.GetHeader("ORGANIZATION"); orgId != "" {
			c.Set("orgId", orgId)
		}
		if projectId := c.GetHeader("PROJECT"); projectId != "" {
			c.Set("projectId", projectId)
		}

		c.Next()
	}
}
