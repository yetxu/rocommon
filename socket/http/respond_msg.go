package http

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"rocommon"
)

type MessageRespond struct {
	Msg        interface{}
	StatusCode int
}

func (this *MessageRespond) WriteRespond(sess *httpSession) error {
	this.StatusCode = http.StatusOK

	httpCodec := rocommon.GetHttpCodec("httpjson")
	if httpCodec == nil {
		return errors.New("ResponseCodec not found httpjson")
	}

	data, err := httpCodec.Marshal(this.Msg)
	if err != nil {
		return err
	}

	sess.resp.Header().Set("Content-Type", "application/json"+";charset=UTF-8")
	sess.resp.WriteHeader(this.StatusCode)
	bodyData, err := ioutil.ReadAll(data.(io.Reader))
	if err != nil {
		return err
	}

	sess.resp.Write(bodyData)

	return nil
}
