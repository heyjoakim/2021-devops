package ui

import (
	"fmt"
	"log"
	"net/http"

	"github.com/heyjoakim/devops-21/services"
)

// BeforeRequest checks if the user is logged in.
func BeforeRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := GetSession(w, r)
		userID := session.Values["user_id"]
		if userID != nil {
			id := userID.(uint)
			tmpUser := services.GetUser(id)
			session.Values["user_id"] = tmpUser.UserID
			session.Values["username"] = tmpUser.Username
			session.Save(r, w)
		}
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}

// AfterRequest logs endpoint requests.
func AfterRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(fmt.Sprintf("[%s] --> %s", r.Method, r.RequestURI))
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
