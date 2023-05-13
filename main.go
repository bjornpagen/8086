package main

import (
	"fmt"
	"os"
)

type Disassembler struct {
	b []byte
	i int
}

func New(b []byte) Disassembler {
	return Disassembler{
		b: b,
		i: 0,
	}
}

func (d *Disassembler) Read() (b byte, err error) {
	if d.i >= len(d.b) {
		return b, fmt.Errorf("end of buffer")
	}

	b = d.b[d.i]
	d.i++

	return b, nil
}

func (d *Disassembler) Peek() (b byte, err error) {
	if d.i >= len(d.b) {
		return 0, fmt.Errorf("end of buffer")
	}

	b = d.b[d.i]

	return b, nil
}

func (d *Disassembler) Unread(n int) (err error) {
	if d.i-n < 0 {
		return fmt.Errorf("cannot unread %d bytes", n)
	}

	d.i -= n

	return nil
}

var reg16 = map[byte]string{
	0b000: "ax",
	0b001: "cx",
	0b010: "dx",
	0b011: "bx",
	0b100: "sp",
	0b101: "bp",
	0b110: "si",
	0b111: "di",
}

var reg8 = map[byte]string{
	0b000: "al",
	0b001: "cl",
	0b010: "dl",
	0b011: "bl",
	0b100: "ah",
	0b101: "ch",
	0b110: "dh",
	0b111: "bh",
}

func (d *Disassembler) Disassemble() (s []string, err error) {
	s = append(s, "bits 16")

	for {
		b, err := d.Read()
		if err != nil {
			break
		}

		switch {
		case b&0b11111100 == 0b10001000: // MOV
			{
				d.Unread(1)
				op, err := d.decodeMOV()
				if err != nil {
					return s, fmt.Errorf("decode MOV: %v", err)
				}

				s = append(s, op)
			}
		default:
			return s, fmt.Errorf("invalid instruction: 0b%b", b)
		}
	}

	return s, nil
}

func (d *Disassembler) decodeMOV() (s string, err error) {
	b, err := d.Read()
	if err != nil {
		return "", fmt.Errorf("read byte: %v", err)
	}

	dFlag := b & 0b00000010 >> 1
	wFlag := b & 0b00000001

	b, err = d.Read()
	if err != nil {
		return "", fmt.Errorf("read byte: %v", err)
	}

	switch b & 0b11000000 {
	case 0b11000000: // MOV reg, reg
		{
			d.Unread(1)
			return d.decodeMOVRegReg(dFlag == 1, wFlag == 1)
		}
	}
	return "", fmt.Errorf("invalid mov")
}

func (d *Disassembler) decodeMOVRegReg(dFlag, wFlag bool) (s string, err error) {
	b, err := d.Read()
	if err != nil {
		return "", fmt.Errorf("read byte: %v", err)
	}

	reg1 := b & 0b00111000 >> 3
	reg2 := b & 0b00000111

	var reg1str, reg2str string

	if wFlag {
		reg1str = reg16[reg1]
		reg2str = reg16[reg2]
	} else {
		reg1str = reg8[reg1]
		reg2str = reg8[reg2]
	}

	if dFlag {
		return fmt.Sprintf("mov %s, %s", reg1str, reg2str), nil
	} else {
		return fmt.Sprintf("mov %s, %s", reg2str, reg1str), nil
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: disassembler <file>")
		os.Exit(1)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}

	stat, err := file.Stat()
	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
		os.Exit(1)
	}

	b := make([]byte, stat.Size())
	_, err = file.Read(b)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	d := New(b)

	s, err := d.Disassemble()
	if err != nil {
		fmt.Printf("Error disassembling file: %v\n", err)
		os.Exit(1)
	}

	for _, line := range s {
		fmt.Println(line)
	}
}
