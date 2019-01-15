package srndtrip

import (
	"encoding/hex"
	"strings"
)

// this package implements unicode tripcode originally used in srnd (and srnd2)

func MakeSRNdTrip(pubkey string, length int) string {
	var b strings.Builder

	data, err := hex.DecodeString(pubkey)
	if err != nil {
		panic(err)
	}

	if length <= 0 {
		length = len(data)
	}

	appendch := func(ch byte) {
		chnum := 9600 + rune(ch)
		b.WriteRune(chnum)
	}
	i := 0
	for ; i < length/2; i++ {
		appendch(data[i])
	}
	for ; i < length; i++ {
		appendch(data[len(data)-length+i])
	}

	return b.String()
}
