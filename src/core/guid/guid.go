package guid

import (
	"errors"
	"fmt"
	"os/exec"
)

type Guid [16]byte

const (
	maskLo byte = 0xf
)

func (g Guid) String() string {
	str := ""

	for i, v := range g {
		piece_one := v >> 4
		piece_two := v & maskLo

		str = fmt.Sprintf("%s%x%x", str, piece_one, piece_two)

		if i == 3 || i == 5 || i == 7 || i == 9 {
			str = fmt.Sprintf("%s-", str)
		}
	}

	return str
}

func convert_byte(b byte) (byte, error) {

	if b >= 48 && b <= 57 {
		return b - 48, nil
	} else if b >= 97 && b <= 102 {
		return b - 87, nil
	} else if b >= 65 && b <= 70 {
		return b - 55, nil
	} else {
		return 0, errors.New("byte given is not an hex")
	}
}

func Deserialize(str []byte) (Guid, error) {

	retVal := Guid{}
	pos := 0
	for i := 0; i < len(str); i += 2 {
		if str[i] == byte('-') {
			i += 1
		}

		var conv1, conv2 byte
		var err error

		conv1, err = convert_byte(str[i])
		if err != nil {
			return Guid{}, fmt.Errorf("byte %d was not an hex", i)
		}

		conv2, err = convert_byte(str[i+1])
		if err != nil {
			return Guid{}, fmt.Errorf("byte %d was not an hex", i+1)
		}

		retVal[pos] = conv1<<4 | conv2
		pos++
	}

	return retVal, nil
}

func GenerateGUID() (Guid, error) {
	cmd := exec.Command("uuidgen")

	var bArr []byte
	var err error
	if bArr, err = cmd.Output(); err != nil {
		return Guid{}, errors.New("uuidgen run failed")
	}
	bArr = bArr[0 : len(bArr)-1]

	return Deserialize(bArr)

}
