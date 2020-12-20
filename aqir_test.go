package aqir

import "testing"

func TestMockCom(t *testing.T) {
	err := FetchAQI()
	if err != nil {
		t.Errorf("failed to test serial port[%q]\n", err)
	} else {
		t.Logf("suceed in testing serial port\n")
	}
}

func TestCalcAQI(t *testing.T) {
	var input float32 = 51
	output, _ := CalcAQI(input)
	println("data[%d] - AQI[%d]\n", input, output)
}
