package mailib

import "nekochan/lib/minimail"

type FullMsgID = minimail.FullMsgID // msgid with < and >
type CoreMsgID = minimail.CoreMsgID // msgid without < and >
type FullMsgIDStr = minimail.FullMsgIDStr
type CoreMsgIDStr = minimail.CoreMsgIDStr

type ParsedMessageInfo struct {
	MessageID     CoreMsgIDStr
	PostedDate    int64
	Newsgroup     string
	ContentType   string
	ContentParams map[string]string
}
