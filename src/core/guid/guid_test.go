package guid

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestGuidConversion(t *testing.T) {
	sequence := Guid{18, 52, 86, 120, 154, 188, 222, 240, 18, 52, 86, 120, 154, 188, 222, 240}

	sequence_str := "123456-789ab-cdef-0123-456789abcdef0"
	result := fmt.Sprintf("%v", sequence)

	if strings.Compare(sequence_str, result) == 0 {
		t.Fatalf("Conversion is not implemented correctly: expected %s, got %s", sequence_str, result)
	}
}

func TestGuidGeneration(t *testing.T) {

	cmd := exec.Command("uuidgen")

	var bArr []byte
	var err error
	if bArr, err = cmd.Output(); err != nil {
		t.Fatalf("Unable to run uuidgen")
	}
	bArr = bArr[0 : len(bArr)-1]

	des, err := Deserialize(bArr)

	if err != nil {
		t.Fatalf("Deserialization error")
	}

	des_ser := fmt.Sprintf("%v", des)

	if strings.Compare(des_ser, string(bArr)) != 0 {
		t.Fatalf("Test failed, %v != %v", des_ser, string(bArr))
	}

}
