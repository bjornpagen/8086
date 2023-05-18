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

// rewrite using an array
var reg16 = []string{
	"ax",
	"cx",
	"dx",
	"bx",
	"sp",
	"bp",
	"si",
	"di",
}

var reg8 = []string{
	"al",
	"cl",
	"dl",
	"bl",
	"ah",
	"ch",
	"dh",
	"bh",
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
				dFlag := b & 0b00000010 >> 1
				wFlag := b & 0b00000001
				op, err := d.decodeMOV(dFlag, wFlag)
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

func (d *Disassembler) decodeMOV(dFlag, wFlag byte) (s string, err error) {
	b, err := d.Read()
	if err != nil {
		return "", fmt.Errorf("read byte: %v", err)
	}

	mod := b & 0b11000000 >> 6
	reg := b & 0b00111000 >> 3
	rm := b & 0b00000111
	switch mod {
	case 0b11: // reg, reg
		{
			return d.decodeMOVRegReg(dFlag, wFlag, reg, rm)
		}
	case 0b10: // mem, reg: displacement of 16 bits
		{
			return d.decodeMOVMemReg(dFlag, wFlag, reg, rm)
		}
	}
	return "", fmt.Errorf("invalid mov")
}

func (d *Disassembler) decodeMOVRegReg(dFlag, wFlag, reg, rm byte) (s string, err error) {
	reg1 := reg
	reg2 := rm

	var reg1str, reg2str string

	if wFlag == 1 {
		reg1str = reg16[reg1]
		reg2str = reg16[reg2]
	} else {
		reg1str = reg8[reg1]
		reg2str = reg8[reg2]
	}

	if dFlag == 1 {
		return fmt.Sprintf("mov %s, %s", reg1str, reg2str), nil
	} else {
		return fmt.Sprintf("mov %s, %s", reg2str, reg1str), nil
	}
}

func (d *Disassembler) decodeMOVMemReg(dFlag, wFlag, reg, rm byte) (s string, err error) {
	return "", nil
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
