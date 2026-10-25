package xgin

import (
	"strconv"
	"strings"
)

// U64 是以十进制字符串序列化的 uint64。
//
// JS 的 Number 无法无损承载超过 2^53-1 的 int64 ( 前端实测 ID 精度损坏 )，
// 因此 HTTP 响应中的全部 int64 ID 均以字符串序列化——与 WS 链路的
// protojson ( int64 -> string ) 行为一致。
//
// 反序列化同时接受字符串与数字字面量，兼容仍在传数字的存量客户端。
type U64 uint64

// String 返回十进制字符串表示。
func (u U64) String() string {
	return strconv.FormatUint(uint64(u), 10)
}

// MarshalJSON 序列化为带引号的十进制字符串。
func (u U64) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, len(u.String())+2)
	buf = append(buf, '"')
	buf = strconv.AppendUint(buf, uint64(u), 10)
	buf = append(buf, '"')
	return buf, nil
}

// UnmarshalJSON 接受十进制字符串或数字字面量 ( null / 空串 视为 0 )。
func (u *U64) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	s = strings.Trim(s, `"`)

	if s == "" || s == "null" {
		*u = 0
		return nil
	}

	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	*u = U64(val)
	return nil
}
