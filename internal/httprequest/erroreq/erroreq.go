package erroreq

import (
	"fmt"
	"net/http"

	"github.com/greyfox12/GoDiplom/internal/api/logmy"
)

func ErrorReq(res http.ResponseWriter, req *http.Request) {

	logmy.OutLogInfo(fmt.Errorf("enter in ErrorPage"))

	res.WriteHeader(http.StatusNotFound)
}
