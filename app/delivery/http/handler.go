package http

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"rankr/app/service/user"
	errmsg "rankr/pkg/err_msg"
	"rankr/pkg/statuscode"
	"rankr/pkg/validator"
	types "rankr/type"
)

type Handler struct {
	userService user.Service
}

func NewHandler(userSrv user.Service) Handler {
	return Handler{
		userService: userSrv,
	}
}

func (h Handler) healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{"status": "ok"})
}

func (h Handler) importUsers(c echo.Context) error {
	summary, err := h.userService.Import(c.Request().Context(), c.Request().Body)
	if err != nil {
		return h.handleError(c, err)
	}
	return c.JSON(http.StatusOK, summary)
}

func (h Handler) getUser(c echo.Context) error {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user id"})
	}

	res, err := h.userService.GetUser(c.Request().Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(http.StatusOK, res)
}

func (h Handler) handleError(c echo.Context, err error) error {
	if vErr, ok := err.(validator.Error); ok {
		return c.JSON(vErr.StatusCode(), vErr)
	}
	if eResp, ok := err.(errmsg.ErrorResponse); ok {
		return c.JSON(statuscode.MapToHTTPStatusCode(eResp), eResp)
	}
	return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
}

func parseIDParam(raw string) (types.ID, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return types.ID(id), nil
}
