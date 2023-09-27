package erroreq

import (
	"net/http"
)

func ErrorReq(res http.ResponseWriter, req *http.Request) {

	//		fmt.Printf("Error page %v\n", http.MethodPost)

	res.WriteHeader(http.StatusNotFound)
}
