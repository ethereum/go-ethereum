// +build gofuzz

package cid

func Fuzz(data []byte) int {
	cid, err := Cast(data)

	if err != nil {
		return 0
	}

	_ = cid.Bytes()
	_ = cid.String()
	p := cid.Prefix()
	_ = p.Bytes()

	if !cid.Equals(cid) {
		panic("inequality")
	}

	// json loop
	json, err := cid.MarshalJSON()
	if err != nil {
		panic(err.Error())
	}
	cid2 := Cid{}
	err = cid2.UnmarshalJSON(json)
	if err != nil {
		panic(err.Error())
	}

	if !cid.Equals(cid2) {
		panic("json loop not equal")
	}

	return 1
}
