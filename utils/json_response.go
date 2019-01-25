package utils

import "encoding/json"

type response struct {
	Code int `json:"code"`
	Msg string `json:"msg"`
	Data string `json:"data"`
}

func JsonResponse(code int, msg string, data string) ([]byte, error) {
	resp := response{code, msg, data}
	return json.Marshal(&resp)
}
