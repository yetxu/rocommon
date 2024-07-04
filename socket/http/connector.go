package http

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"rocommon"
	"rocommon/socket"
	"strings"
	"time"
)

type httpConnector struct {
	socket.NetServerNodeProperty
	socket.NetContextSet
	socket.NetProcessorRPC //事件处理相关
}

func (this *httpConnector) Start() rocommon.ServerNode {
	return this
}

func (this *httpConnector) Stop() {
}

func (this *httpConnector) TypeOfName() string {
	return "httpConnector"
}
func (this *httpConnector) Request(method, path string, param *rocommon.HTTPRequest) error {
	codecProc := rocommon.GetHttpCodec(param.ReqCodecName)
	if method == "POST" {
		data, err := codecProc.Marshal(param.ReqMsg)
		if err != nil {
			return err
		}

		url := fmt.Sprintf("http://%s%s", this.GetAddr(), path)
		if strings.Contains(this.GetAddr(), "http") {
			url = fmt.Sprintf("%s%s", this.GetAddr(), path)
		}

		req, err := http.NewRequest(method, url, data.(io.Reader))
		if err != nil {
			return nil
		}

		mimeType := codecProc.(interface {
			MimeType() string
		}).MimeType()
		req.Header.Set("Content-Type", mimeType)

		resp, err := defaultHttpClient.Do(req)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			return err
		}

		//log.Println("[header]:", resp.Header, resp.Status, resp.Body)
		return codecProc.Unmarshal(resp.Body, param.ResMsg)
	} else {
		url := fmt.Sprintf("http://%s%s", this.GetAddr(), path)
		if strings.Contains(this.GetAddr(), "http") {
			url = fmt.Sprintf("%s%s", this.GetAddr(), path)
		}
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return nil
		}

		mimeType := codecProc.(interface {
			MimeType() string
		}).MimeType()
		req.Header.Set("Content-Type", mimeType)

		resp, err := defaultHttpClient.Do(req)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			return err
		}

		//log.Println("[header]:", resp.Header, resp.Status, resp.Body)
		return codecProc.Unmarshal(resp.Body, param.ResMsg)
	}
}

var defaultHttpClient *http.Client = nil

func defaultClient() {
	defaultHttpClient = &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				deadline := time.Now().Add(time.Second * 30)
				c, err := net.DialTimeout(network, addr, time.Second*30)
				if err != nil {
					return nil, err
				}
				c.SetDeadline(deadline)
				return c, nil
			},
		},
	}
}

func init() {
	log.Println("httpConnector server node register")
	socket.RegisterServerNode(func() rocommon.ServerNode {
		node := &httpConnector{}
		return node
	})
	defaultClient()
}
