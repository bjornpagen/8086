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

var addr = []string{
	"bx + si",
	"bx + di",
	"bp + si",
	"bp + di",
	"si",
	"di",
	"bp",
	"bx",
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
		case b&0b11110000 == 0b10110000: // MOV immediate
			{
				wFlag := b & 0b00001000 >> 3
				reg := b & 0b00000111
				op, err := d.decodeMOVImm(wFlag, reg)
				if err != nil {
					return s, fmt.Errorf("decode MOV immediate: %v", err)
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
	case 0b01: // mem, reg, 8-bit displacement
		{
			return d.decodeMOVMemReg8(dFlag, wFlag, reg, rm)
		}
	case 0b10: // mem, reg, 16-bit displacement
		{
			return d.decodeMOVMemReg16(dFlag, wFlag, reg, rm)
		}
	case 0b00: // mem, reg, no displacement
		{
			return d.decodeMOVMemReg0(dFlag, wFlag, reg, rm)
		}
	}
	return "", fmt.Errorf("invalid mov: 0b%b", b)
}

func (d *Disassembler) decodeMOVImm(wFlag, reg byte) (s string, err error) {
	b, err := d.Read()
	if err != nil {
		return "", fmt.Errorf("read byte: %v", err)
	}

	var regstr string
	var imm int16
	if wFlag == 1 {
		regstr = reg16[reg]
		imm = int16(b)
		b, err = d.Read()
		if err != nil {
			return "", fmt.Errorf("read byte: %v", err)
		}
		imm |= int16(b) << 8
	} else {
		regstr = reg8[reg]
		imm = int16(b)
	}
	immstr := fmt.Sprintf("%d", imm)

	return fmt.Sprintf("mov %s, %s", regstr, immstr), nil
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
	}
	return fmt.Sprintf("mov %s, %s", reg2str, reg1str), nil
}

func (d *Disassembler) decodeMOVMemReg8(dFlag, wFlag, reg, rm byte) (s string, err error) {
	b, err := d.Read()
	if err != nil {
		return "", fmt.Errorf("read byte: %v", err)
	}

	displacement := int8(b)
	var addrexp string
	if displacement != 0 {
		addrexp = fmt.Sprintf("[%s + %d]", addr[rm], displacement)
	} else {
		addrexp = fmt.Sprintf("[%s]", addr[rm])
	}

	var regstr string
	if wFlag == 1 {
		regstr = reg16[reg]
	} else {
		regstr = reg8[reg]
	}

	if dFlag == 1 {
		return fmt.Sprintf("mov %s, %s", regstr, addrexp), nil
	}
	return fmt.Sprintf("mov %s, %s", addrexp, regstr), nil
}

func (d *Disassembler) decodeMOVMemReg16(dFlag, wFlag, reg, rm byte) (s string, err error) {
	b, err := d.Read()
	if err != nil {
		return "", fmt.Errorf("read byte: %v", err)
	}

	displacement := int16(b)
	b, err = d.Read()
	if err != nil {
		return "", fmt.Errorf("read byte: %v", err)
	}
	displacement |= int16(b) << 8

	var addrexp string
	if displacement != 0 {
		addrexp = fmt.Sprintf("[%s + %d]", addr[rm], displacement)
	} else {
		addrexp = fmt.Sprintf("[%s]", addr[rm])
	}

	var regstr string
	if wFlag == 1 {
		regstr = reg16[reg]
	} else {
		regstr = reg8[reg]
	}

	if dFlag == 1 {
		return fmt.Sprintf("mov %s, %s", regstr, addrexp), nil
	}
	return fmt.Sprintf("mov %s, %s", addrexp, regstr), nil
}

func (d *Disassembler) decodeMOVMemReg0(dFlag, wFlag, reg, rm byte) (s string, err error) {
	var addrexp string
	if rm == 0b110 {
		b, err := d.Read()
		if err != nil {
			return "", fmt.Errorf("read byte: %v", err)
		}

		displacement := int16(b)
		b, err = d.Read()
		if err != nil {
			return "", fmt.Errorf("read byte: %v", err)
		}

		displacement |= int16(b) << 8

		addrexp = fmt.Sprintf("[%d]", displacement)
	} else {
		addrexp = fmt.Sprintf("[%s]", addr[rm])
	}

	var regstr string
	if wFlag == 1 {
		regstr = reg16[reg]
	} else {
		regstr = reg8[reg]
	}

	if dFlag == 1 {
		return fmt.Sprintf("mov %s, %s", regstr, addrexp), nil
	}
	return fmt.Sprintf("mov %s, %s", addrexp, regstr), nil
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

	fmt.Printf("; %s disassembly:\n", os.Args[1])
	for _, line := range s {
		fmt.Println(line)
	}
}
