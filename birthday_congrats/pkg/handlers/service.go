package handlers

import (
	"birthday_congrats/pkg/service"
	"birthday_congrats/pkg/session"
	"birthday_congrats/pkg/user"
	"html/template"
	"net/http"

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

func (h *ServiceHandler) Index(w http.ResponseWriter, r *http.Request) {
	_, err := session.SessionFromContext(r.Context())
	if err == nil {
		http.Redirect(w, r, "/users", http.StatusFound)
		return
	}

	err = h.tmpl.ExecuteTemplate(w, "login.html", nil)
	if err != nil {
		h.logger.Errorf("template error: %v", err)
	}
}

func (h *ServiceHandler) Error(w http.ResponseWriter, r *http.Request) {
	err := h.tmpl.ExecuteTemplate(w, "error.html", nil)
	if err != nil {
		h.logger.Errorf("Template error: %v", err)
	}
}

func (h *ServiceHandler) Register(w http.ResponseWriter, r *http.Request) {
	_, err := h.service.Register(
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
		err = h.tmpl.ExecuteTemplate(w, "error.html", nil)
		if err != nil {
			h.logger.Errorf("Template error: %v", err)
		}

		return
	}

	http.Redirect(w, r, "/user", http.StatusFound)
}

func (h *ServiceHandler) Login(w http.ResponseWriter, r *http.Request) {
	_, err := h.service.Login(
		r.Context(),
		r.FormValue("username"),
		r.FormValue("password"),
	)
	if err != nil && err != user.ErrNoUser {
		h.logger.Errorf("Error while registration: %v", err)
	}
	if err == user.ErrNoUser {
		err = h.tmpl.ExecuteTemplate(w, "error.html", nil)
		if err != nil {
			h.logger.Errorf("Template error: %v", err)
		}

		return
	}

	http.Redirect(w, r, "/user", http.StatusFound)
}
