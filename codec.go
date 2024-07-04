package rocommon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

type Codec interface {
	Marshal(msg interface{}) (interface{}, error) //todo...上下文Context

	Unmarshal(data interface{}, msg interface{}) error

	TypeOfName() string
}

var registerCodec Codec //后续有别的解析部分这边可以添加
var httpCodec Codec

func init() {
	//注册protobuf解析
	RegisterCodec(new(pbCodec))
	httpCodec = &httpJsonCodec{}
	//httpCodec = &httpFormCodec{}
}

func RegisterCodec(c Codec) {
	log.Println("RegisterCodec pbcodec")
	registerCodec = c
}

func GetCodec() Codec {
	return registerCodec
}
func GetHttpCodec(codecName string) Codec {
	if codecName == "" {
		return httpCodec
	}
	switch codecName {
	case "httpform":
		return &httpFormCodec{}
	case "httpjson":
		return &httpJsonCodec{}
	}
	return httpCodec
}

//pbCodec
type pbCodec struct {
}

func (this *pbCodec) TypeOfName() string {
	return "protobuf"
}
func (this *pbCodec) Marshal(msg interface{}) (interface{}, error) {
	return proto.Marshal(msg.(proto.Message))
}
func (this *pbCodec) Unmarshal(data interface{}, msg interface{}) error {
	return proto.Unmarshal(data.([]byte), msg.(proto.Message))
}

//http json
type httpJsonCodec struct {
}

func (this *httpJsonCodec) TypeOfName() string {
	return "httpjson"
}
func (this *httpJsonCodec) MimeType() string {
	return "application/json"
}
func (this *httpJsonCodec) Marshal(msg interface{}) (interface{}, error) {
	httpData, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	//log.Printf("httpData:%v", httpData)

	return bytes.NewReader(httpData), nil
}
func (this *httpJsonCodec) Unmarshal(data interface{}, msg interface{}) error {
	var reader io.Reader
	switch v := data.(type) {
	case *http.Request:
		reader = v.Body
	case io.Reader:
		reader = v
	}
	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	log.Println("httpJsonCodec:", string(body))
	return json.Unmarshal(body, msg)
}

//httpForm
const defaultMemory = 32 * 1024 * 1024

type httpFormCodec struct {
}

func (this *httpFormCodec) TypeOfName() string {
	return "httpform"
}
func (this *httpFormCodec) MimeType() string {
	return "application/x-www-form-urlencoded"
}
func (this *httpFormCodec) Marshal(msg interface{}) (interface{}, error) {
	return strings.NewReader(this.form2UrlValues(msg).Encode()), nil
}
func (this *httpFormCodec) Unmarshal(data interface{}, msg interface{}) error {
	//todo...
	/*
		var reader io.Reader
		switch v := data.(type) {
		case *http.Request:
			reader = v.Body
		case io.Reader:
			reader = v
		}
		body,err := ioutil.ReadAll(reader)
		if err != nil {
			return err
		}
		type aast struct{
			Ret int
			Msg string
		}
		var aa aast
		json.Unmarshal(body,&aa)
		log.Println("body11:", string(body), aa)
	*/

	//log.Println("type:", reflect.TypeOf(data))

	//reader, err := gzip.NewReader(data.(io.Reader))
	if msg != nil {
		body, err := ioutil.ReadAll(data.(io.Reader))
		if err != nil {
			return err
		}
		//log.Println("body11:", string(body))

		msgValue := reflect.ValueOf(msg)
		if msgValue.Kind() == reflect.Ptr {
			msgValue = msgValue.Elem()
		}
		msgValue.Field(0).SetString(string(body))
	}

	//msg = this.value2String(string(body))

	//
	//resp := data.(*http.Request)
	//err = resp.ParseForm()
	//if err != nil {
	//	return nil
	//}
	//log.Println("[httpFormCodec]body:", resp.Form)
	//resp.ParseMultipartForm(defaultMemory)

	return nil
}
func (this *httpFormCodec) form2UrlValues(obj interface{}) url.Values {
	objValue := reflect.Indirect(reflect.ValueOf(obj))
	objType := reflect.TypeOf(obj)

	formValues := url.Values{}
	for i := 0; i < objValue.NumField(); i++ {
		field := objType.Field(i)
		val := objValue.Field(i)
		//if field {
		formValues.Add(field.Name, this.value2String(val.Interface()))
		//}
	}
	return formValues
}
func (this *httpFormCodec) value2String(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(int64(v), 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		panic("Unknown type to convert to string")
	}
}

/////////////////////
type MessageInfo struct {
	Codec        Codec
	Type         reflect.Type
	ID           int
	ConfirmMsgId int //需要确认的req消息，如果info是req则是ack的id，如果是ack则是req的id
}

var (
	messageByID   = map[int]*MessageInfo{}
	messageByType = map[reflect.Type]*MessageInfo{}
	messageByName = map[string]*MessageInfo{}
)

func RegisterMessageInfo(info *MessageInfo) {
	//注册时统一为非指针类型
	if info.Type.Kind() == reflect.Ptr {
		info.Type = info.Type.Elem()
	}

	if info.ID == 0 {
		panic(fmt.Sprintf("message ID invalid:%v", info.Type.Name()))
	}

	if _, ok := messageByID[info.ID]; ok {
		panic(fmt.Sprintf("message has register id:%v", info.Type.Name()))
	} else {
		messageByID[info.ID] = info
	}

	if _, ok := messageByType[info.Type]; ok {
		panic(fmt.Sprintf("message has register type:%v", info.Type.Name()))
	} else {
		messageByType[info.Type] = info
	}

	if _, ok := messageByName[info.Type.Name()]; ok {
		panic(fmt.Sprintf("message has register name:%v", info.Type.Name()))
	} else {
		messageByName[info.Type.Name()] = info
	}

	//log.Printf("message register [id|type|name][%v%v|%v]\n", info.ID,info.Type,info.Type.Name())
}

func MessageInfoByID(id int) *MessageInfo {
	if data, ok := messageByID[id]; ok {
		return data
	}
	return nil
}

func MessageInfoByMsg(msg interface{}) *MessageInfo {
	msgType := reflect.TypeOf(msg)
	if msgType.Kind() == reflect.Ptr {
		if info, ok := messageByType[msgType.Elem()]; ok {
			return info
		}
		return nil
	} else {
		if info, ok := messageByType[msgType]; ok {
			return info
		}
		return nil
	}
}

func MessageInfoByName(name string) *MessageInfo {
	if info, ok := messageByName[name]; ok {
		return info
	}
	return nil
}

func MessageToString(msg interface{}) string {
	if msg == nil {
		return ""
	}
	if str, ok := msg.(interface{ String() string }); ok {
		return str.String()
	}
	return ""
}
