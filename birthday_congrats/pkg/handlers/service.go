package handlers

import (
	"birthday_congrats/pkg/service"
	"birthday_congrats/pkg/session"
	"birthday_congrats/pkg/user"
	"html/template"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type ServiceHandler struct {
	tmpl    *template.Template
	service *service.CongratulationsService
	sm      session.SessionsManager
	logger  *zap.SugaredLogger
}

func NewServiceHandler(
	tmpl *template.Template,
	service *service.CongratulationsService,
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

func (h *ServiceHandler) execErrorTemplate(w http.ResponseWriter, message string) {
	err := h.tmpl.ExecuteTemplate(w, "error.html", struct {
		Message string
	}{
		Message: message,
	})
	if err != nil {
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

	err = h.tmpl.ExecuteTemplate(w, "login.html", nil)
	if err != nil {
		h.logger.Errorf("template error: %v", err)
	}
}

func (h *ServiceHandler) Error(w http.ResponseWriter, r *http.Request) {
	h.execErrorTemplate(w, "Произошла ошибка")
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
	}
	if err == user.ErrUserExists {
		h.execErrorTemplate(w, "Пользоваель с таким именем уже существует")

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
	}
	if err == user.ErrNoUser {
		h.execErrorTemplate(w, "Неверный логин или пароль")

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
	users, err := h.service.GetAll(r.Context())
	if err != nil {
		h.logger.Errorf("Error getting all users: %v", err)
	}

	err = h.tmpl.ExecuteTemplate(w, "users.html", struct {
		Users []*user.User
	}{
		Users: users,
	})
	if err != nil {
		h.logger.Errorf("Template error: %v", err)
	}
}
