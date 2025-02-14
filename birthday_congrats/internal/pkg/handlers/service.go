package handlers

import (
	"birthday_congrats/internal/pkg/session"
	"birthday_congrats/internal/pkg/user"
	service "birthday_congrats/internal/services/congrats_service"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type ServiceHandler struct {
	tmpl    *template.Template
	service service.CongratulationsService
	sm      session.SessionsManager
	logger  *zap.SugaredLogger
}

func NewServiceHandler(
	tmpl *template.Template,
	service service.CongratulationsService,
	sm session.SessionsManager,
	logger *zap.SugaredLogger,
) *ServiceHandler {
	return &ServiceHandler{
		tmpl:    tmpl,
		service: service,
		sm:      sm,
		logger:  logger,
	}
}

func (h *ServiceHandler) execErrorTemplate(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)

	err := h.tmpl.ExecuteTemplate(w, "error.html", struct {
		Message string
	}{
		Message: message,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Errorf("Template error: %v", err)
	}
}

func (h *ServiceHandler) Index(w http.ResponseWriter, r *http.Request) {
	sess, err := h.sm.Check(r)
	if err == nil {
		ctx := session.ContextWithSession(r.Context(), sess)
		http.Redirect(w, r.WithContext(ctx), "/users", http.StatusFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = h.tmpl.ExecuteTemplate(w, "login.html", nil)
	if err != nil {
		h.logger.Errorf("template error: %v", err)
		http.Redirect(w, r, "/error", http.StatusFound)
	}
}

func (h *ServiceHandler) ErrorPage(w http.ResponseWriter, r *http.Request) {
	h.execErrorTemplate(w, "Произошла ошибка", http.StatusInternalServerError)
}

func (h *ServiceHandler) Register(w http.ResponseWriter, r *http.Request) {
	sess, err := h.service.Register(
		r.Context(),
		r.FormValue("username"),
		r.FormValue("password"),
		r.FormValue("email"),
		r.FormValue("birth"),
	)
	if err != nil && err != user.ErrUserExists {
		h.logger.Errorf("Error while registration: %v", err)
		http.Redirect(w, r, "/error", http.StatusFound)
		return
	}
	if err == user.ErrUserExists {
		h.execErrorTemplate(w, "Пользоваель с таким именем уже существует", http.StatusForbidden)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "session_id",
		Value:   sess.SessID,
		Expires: time.Unix(sess.Expires, 0),
	})

	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *ServiceHandler) Login(w http.ResponseWriter, r *http.Request) {
	sess, err := h.service.Login(
		r.Context(),
		r.FormValue("username"),
		r.FormValue("password"),
	)
	if err != nil && err != user.ErrNoUser {
		h.logger.Errorf("Error while registration: %v", err)
		http.Redirect(w, r, "/error", http.StatusFound)
		return
	}
	if err == user.ErrNoUser {
		h.execErrorTemplate(w, "Неверный логин или пароль", http.StatusForbidden)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "session_id",
		Value:   sess.SessID,
		Expires: time.Unix(sess.Expires, 0),
	})

	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *ServiceHandler) Users(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.GetSubscriptionsByUser(r.Context())
	if err != nil {
		h.logger.Errorf("Error getting all users: %v", err)
		http.Redirect(w, r, "/error", http.StatusFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	err = h.tmpl.ExecuteTemplate(w, "users.html", struct {
		Users []*user.User
	}{
		Users: users,
	})
	if err != nil {
		h.logger.Errorf("Template error: %v", err)
		http.Redirect(w, r, "/error", http.StatusFound)
	}
}

func (h *ServiceHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	subscriptionID, err := strconv.Atoi(mux.Vars(r)["user_id"])
	if err != nil {
		h.logger.Errorf("Error converting string to int: %v", err)
		http.Redirect(w, r, "/error", http.StatusFound)
		return
	}

	daysAlert, err := strconv.Atoi(r.FormValue("days_alert"))
	if err != nil {
		h.logger.Errorf("Error converting string to int: %v", err)
		http.Redirect(w, r, "/error", http.StatusFound)
		return
	}

	err = h.service.Subscribe(r.Context(), uint32(subscriptionID), daysAlert)
	if err != nil {
		h.logger.Errorf("Error while subscribing: %v", err)
		http.Redirect(w, r, "/error", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *ServiceHandler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	subscriptionID, err := strconv.Atoi(mux.Vars(r)["user_id"])
	if err != nil {
		h.logger.Errorf("Error converting string to int: %v", err)
		http.Redirect(w, r, "/error", http.StatusFound)
		return
	}

	err = h.service.Unsubscribe(r.Context(), uint32(subscriptionID))
	if err != nil {
		h.logger.Errorf("Error while unsubscribing: %v", err)
		http.Redirect(w, r, "/error", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/users", http.StatusFound)
}

func (h *ServiceHandler) Logout(w http.ResponseWriter, r *http.Request) {
	err := h.service.Logout(r.Context())
	if err != nil && err != session.ErrNotDestroyed {
		h.logger.Errorf("Error while logout: %v", err)
		http.Redirect(w, r, "/error", http.StatusFound)
		return
	}
	if err == session.ErrNotDestroyed {
		h.logger.Warnf("Session was not destroyed")
	}

	cookie, err := r.Cookie("session_id")
	if err != nil {
		h.logger.Warnf("No cookie found")
		http.Redirect(w, r, "/error", http.StatusFound)
		return
	}

	cookie.Expires = time.Now().AddDate(0, 0, -1)
	http.SetCookie(w, cookie)

	http.Redirect(w, r, "/", http.StatusFound)
}
