package main

import (
	"encoding/hex"
	"testing"
	"unicode/utf16"
)

func decodeUCS2(inputStr string) (string, error) {
	bytes, err := hex.DecodeString(inputStr)
	if err != nil {
		return inputStr, nil
	}
	if len(bytes)%2 != 0 {
		return inputStr, nil
	}

	runes := make([]uint16, len(bytes)/2)
	for i := 0; i < len(runes); i++ {
		runes[i] = uint16(bytes[2*i])<<8 | uint16(bytes[2*i+1])
	}

	decoded := string(utf16.Decode(runes))
	return decoded, nil
}

func TestDecodeUCS2(t *testing.T) {
	input := "004D00E30020005400E000690020006B0068006F1EA3006E0020004100700070006C0065003A0020003300310033003500370032002E002001101EEB006E006700200063006800690061002000731EBB0020006D00E3002E"
	actual, _ := decodeUCS2(input)
	t.Log(actual)
}
