package rpc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"rocommon"
	"rocommon/util"
)

const (
	lenMaxLen  = 2 //包体大小2个字节 uint16
	msgIdLen   = 2 //包ID大小2个字节  uint16
	msgSeqlen  = 4 //发送序列号2个字节大小，用来断线重连
	msgFlaglen = 2 //暂定标记，加解密 1表示RSA，2表示AES

	SC_HAND_SHAKE_NTFMsgId = 1006
	SC_HAND_SHAKE_ACKMsgId = 0
	CS_HAND_SHAKE_REQMsgId = 0
	SC_PING_ACKMsgId       = 1001
)

//var SC_HAND_SHAKE_NTFMsgId = MessageInfoByName("SCHandShakeNtf").ID

///////////////////////
func ReadMessage(reader io.Reader, maxMsgLen int, aesKey *[]byte) (msg interface{}, msgSeqId uint32, err error) {
	var msgId, flagId uint16 = 0, 0
	var msgData []byte
	msgId, msgSeqId, flagId, msgData, err = RecvPackageData(reader, maxMsgLen)
	if err != nil {
		return nil, 0, err
	}

	switch flagId {
	case 1:
		if int(msgId) == SC_HAND_SHAKE_NTFMsgId { //SC_HAND_SHAKE_NTF
			msgData, err = RSADecrypt(msgData, PrivateClientKey)
			if err != nil {
				return nil, 0, err
			}
		} else if int(msgId) == CS_HAND_SHAKE_REQMsgId { //CS_HAND_SHAKE_REQ
			msgData, err = RSADecrypt(msgData, PrivateServerKey)
			if err != nil {
				return nil, 0, err
			}
		} else if int(msgId) == SC_HAND_SHAKE_ACKMsgId { //SC_HAND_SHAKE_ACK
			msgData, err = RSADecrypt(msgData, PrivateClientKey)
			if err != nil {
				return nil, 0, err
			}
		} else {
			msgData, err = RSADecrypt(msgData, PrivateKey)
			if err != nil {
				return nil, 0, err
			}
		}
	case 2:
		msgData, err = AESCtrDecrypt(msgData, *aesKey, *aesKey...)
		//msgData, err = AESCtrDecrypt(msgData, *aesKey)
		if err != nil {
			return nil, 0, err
		}
	}

	//服务器内部不做加密处理
	msg, _, err = DecodeMessage(int(msgId), msgData)
	if err != nil {
		//log.Println("[DecodeMessage] err:", err)
		return nil, 0, errors.New(fmt.Sprintf("msg decodeMessage failed:%v %v", msgId, err))
	}

	/*

		bufMsgLen := make([]byte, lenMaxLen)
		_, err = io.ReadFull(reader, bufMsgLen)
		if err != nil {
			//log.Println("[ReadMessage] read message err:", err)
			return
		}

		if len(bufMsgLen) < lenMaxLen {
			err = errors.New("message too short")
			return
		}

		msgLen := binary.BigEndian.Uint16(bufMsgLen)
		if(msgLen > 0 && msgLen > uint16(maxMsgLen)) || msgLen < lenMaxLen{
			err = errors.New(fmt.Sprintf("message too big33:%v %v\n",msgLen, maxMsgLen))
			return
		}

		msgData := make([]byte, msgLen - lenMaxLen)
		if _, err = io.ReadFull(reader, msgData); err != nil {
			//log.Println("[ReadMessage] read message err:", err)
			return
		}
		if len(msgData) < msgIdLen{
			return nil, 0, errors.New("message id too short")
		}

		msgId := binary.BigEndian.Uint16(msgData)
		msgData = msgData[msgIdLen:]
		msgSeqId = binary.BigEndian.Uint32(msgData) //序列号
		//log.Println("readSeqId:", msgSeqId)
		body := msgData[msgSeqlen:]
		msg, _, err = DecodeMessage(int(msgId), body)
		if err != nil {
			//log.Println("[DecodeMessage] err:", err)
			return nil, 0, errors.New(fmt.Sprintf("msg decodeMessage failed:%v %v",msgId, err))
		}
	*/
	return
}

//消息反序列化
func DecodeMessage(id int, data []byte) (interface{}, *rocommon.MessageInfo, error) {
	msgInfo := rocommon.MessageInfoByID(id)
	if msgInfo == nil {
		return nil, nil, errors.New("msg not register")
	}

	msg := reflect.New(msgInfo.Type).Interface()
	//解码操作这边直接用protobuf即可
	err := msgInfo.Codec.Unmarshal(data, msg)
	if err != nil {
		return nil, msgInfo, err
	}
	return msg, msgInfo, nil
}

func SendMessage(writer io.Writer, msg interface{}, aesKey *[]byte, maxMsgLen int, nodeName string) (err error) {
	var (
		msgData []byte
		msgId   uint16
		seqId   uint32
		msgInfo *rocommon.MessageInfo
	)

	switch m := msg.(type) {
	case *rocommon.TransmitPacket:
		msgData = m.MsgData
		msgId = uint16(m.MsgId)
		seqId = m.SeqId
	default:
		msgData, msgInfo, err = EncodeMessage(msg)
		if err != nil {
			return err
		}
		msgId = uint16(msgInfo.ID)
	}

	//todo
	// 注意上层发包不要超过最大值
	msgLen := len(msgData)
	var cryptType uint16 = 0

	//握手阶段
	if msgId == uint16(SC_HAND_SHAKE_NTFMsgId) {
		cryptType = 1
		msgData, err = RSAEncrypt(msgData, PublicClientKey)
		if err != nil {
			return err
		}
		msgLen = len(msgData)
	} else {
		if len(*aesKey) > 0 && msgId != SC_PING_ACKMsgId {
			cryptType = 2
			msgData, err = AESCtrEncrypt(msgData, *aesKey, *aesKey...)
			//msgData, err = AESCtrEncrypt(msgData, *aesKey)
			if err != nil {
				return err
			}
			msgLen = len(msgData)
		}
	}

	if msgLen > maxMsgLen {
		err = errors.New(fmt.Sprintf("message too big msgId=%v msglen=%v maxlen=%v", msgId, msgLen, maxMsgLen))
		util.FatalF("SendMessage err=%v", err)
		err = nil
		return
	}

	//data := make([]byte, lenMaxLen + msgIdLen + msgLen)
	data := make([]byte, lenMaxLen+msgIdLen+msgSeqlen+msgFlaglen+msgLen) //head + body
	//lenMaxLen
	binary.BigEndian.PutUint16(data, uint16(msgLen))
	//msgIdLen
	binary.BigEndian.PutUint16(data[lenMaxLen:], msgId)
	//seq 返回客户端发送的序列号
	binary.BigEndian.PutUint32(data[lenMaxLen+msgIdLen:], seqId)
	//log.Println("sendSeqId:", seqId)
	//使用的加密方式AES
	binary.BigEndian.PutUint16(data[lenMaxLen+msgIdLen+msgSeqlen:], cryptType)

	//body
	if msgLen > 0 {
		copy(data[lenMaxLen+msgIdLen+msgSeqlen+msgFlaglen:], msgData)
	}

	//ioutil.go
	err = util.WriteFull(writer, data)

	//todo...使用内存池是否data数据

	return err
}

//消息序列化
func EncodeMessage(msg interface{}) (data []byte, info *rocommon.MessageInfo, err error) {
	info = rocommon.MessageInfoByMsg(msg)
	if info == nil {
		return nil, nil, errors.New("msg not register")
	}

	//log.Println("EncodeMessage:", msg)
	tempData, e := info.Codec.Marshal(msg)
	data = tempData.([]byte)
	err = e
	return
}

//获取原始包数据(二进制)，不做解析处理
func RecvPackageData(reader io.Reader, maxMsgLen int) (msgId uint16, msgSeqId uint32, msgFlagId uint16, msgData []byte, err error) {
	bufMsgLen := make([]byte, lenMaxLen)
	_, err = io.ReadFull(reader, bufMsgLen)
	if err != nil {
		//log.Println("[ReadMessage] read message err:", err)
		return
	}

	if len(bufMsgLen) < lenMaxLen {
		//err = errors.New("message too short")
		return
	}

	//msgId
	bufIdLen := make([]byte, msgIdLen)
	_, err = io.ReadFull(reader, bufIdLen)
	if err != nil {
		//log.Println("[ReadMessage] read message err:", err)
		return
	}
	if len(bufIdLen) < msgIdLen {
		//err = errors.New("message too short")
		return
	}
	msgId = binary.BigEndian.Uint16(bufIdLen)
	//msgseqid
	bufSeqIdLen := make([]byte, msgSeqlen)
	_, err = io.ReadFull(reader, bufSeqIdLen)
	if err != nil {
		//log.Println("[ReadMessage] read message err:", err)
		return
	}
	if len(bufSeqIdLen) < msgSeqlen {
		//err = errors.New("message too short")
		return
	}
	msgSeqId = binary.BigEndian.Uint32(bufSeqIdLen)

	//msgFlaglen 1表示RSA，2表示AES
	bufFlagLen := make([]byte, msgFlaglen)
	_, err = io.ReadFull(reader, bufFlagLen)
	if err != nil {
		return
	}
	if len(bufFlagLen) < msgFlaglen {
		return
	}
	msgFlagId = binary.BigEndian.Uint16(bufFlagLen)

	//BigEndian
	msgLen := binary.BigEndian.Uint16(bufMsgLen)
	if msgLen > 0 && msgLen > uint16(maxMsgLen) {
		//err = errors.New("message too big")
		err = errors.New(fmt.Sprintf("message too big msgid=%v mslen=%v maxlen=%v bufMsgLen=%v msgFlagId=%v\n",
			msgId, msgLen, maxMsgLen, len(bufMsgLen), msgFlagId))
		util.FatalF("RecvPackageData err=%v", err)
		err = nil
		return
	}

	//todo 可以使用内存池
	if msgLen > 0 {
		//body := make([]byte, msgLen)
		//if _, err = io.ReadFull(reader, body); err != nil {
		//	//log.Println("[ReadMessage] read message err:", err)
		//	return
		//}
		//if len(body) < int(msgLen) {
		//	err = errors.New(fmt.Sprintf("message id too short msgid=%v", msgId))
		//	return
		//}
		//
		////msgId = binary.BigEndian.Uint16(body)
		////body = body[msgIdLen:]
		////msgSeqId = binary.BigEndian.Uint32(body) //序列号
		////log.Println("readSeqId:", msgSeqId)
		////msgData = body[msgSeqlen:]
		//msgData = body

		msgData = make([]byte, msgLen)
		if _, err = io.ReadFull(reader, msgData); err != nil {
			//log.Println("[ReadMessage] read message err:", err)
			return
		}
		if len(msgData) < int(msgLen) {
			err = errors.New(fmt.Sprintf("message id too short msgid=%v", msgId))
			return
		}
	}

	return
}
