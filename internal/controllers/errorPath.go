package controllers

import (
	"fmt"
	"net/http"
)

func (c *BaseController) errorPath(res http.ResponseWriter, req *http.Request) {

	c.Loger.OutLogInfo(fmt.Errorf("enter in ErrorPage"))

	res.WriteHeader(http.StatusNotFound)
}
