package vm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/params"
)

func validateControlFlow(code []byte, section int, metadata []*functionMetadata, jt *JumpTable) (int, error) {
	var (
		maxStackHeight = int(metadata[section].inputs)
		debugging      = !true
		visitCount     = 0
		next           = make([]int, 0, 1)
	)
	var (
		stackBoundsMax = make([]uint16, len(code))
		stackBoundsMin = make([]uint16, len(code))
	)
	setBounds := func(pos, min, maxi int) {
		// The stackboundMax slice is a bit peculiar. We use `0` to denote
		// not set. Therefore, we use `1` to represent the value `0`, and so on.
		// So if the caller wants to store `1` as max bound, we internally store it as
		// `2`.
		if stackBoundsMax[pos] == 0 { // Not yet set
			visitCount++
		}
		if maxi < 65535 {
			stackBoundsMax[pos] = uint16(maxi + 1)
		}
		stackBoundsMin[pos] = uint16(min)
		maxStackHeight = max(maxStackHeight, maxi)
	}
	getStackMaxMin := func(pos int) (ok bool, min, max int) {
		maxi := stackBoundsMax[pos]
		if maxi == 0 { // Not yet set
			return false, 0, 0
		}
		return true, int(stackBoundsMin[pos]), int(maxi - 1)
	}
	// set the initial stack bounds
	setBounds(0, int(metadata[section].inputs), int(metadata[section].inputs))

	qualifiedExit := false
	for pos := 0; pos < len(code); pos++ {
		op := OpCode(code[pos])
		ok, currentStackMin, currentStackMax := getStackMaxMin(pos)
		if !ok {
			if debugging {
				fmt.Printf("Stack bounds not set: %v at %v \n", op, pos)
			}
			return 0, ErrUnreachableCode
		}
		if debugging {
			fmt.Println(pos, op, maxStackHeight, currentStackMin, currentStackMax)
		}

		switch op {
		case CALLF:
			arg, _ := parseUint16(code[pos+1:])
			newSection := metadata[arg]
			if want, have := int(newSection.inputs), currentStackMin; want > have {
				return 0, fmt.Errorf("%w: at pos %d", ErrStackUnderflow{stackLen: have, required: want}, pos)
			}
			if have, limit := currentStackMax+int(newSection.maxStackHeight)-int(newSection.inputs), int(params.StackLimit); have > limit {
				return 0, fmt.Errorf("%w: at pos %d", ErrStackOverflow{stackLen: have, limit: limit}, pos)
			}
			change := int(newSection.outputs) - int(newSection.inputs)
			currentStackMax += change
			currentStackMin += change
		case RETF:
			if currentStackMax != currentStackMin {
				return 0, fmt.Errorf("%w: max %d, min %d, at pos %d", ErrInvalidOutputs, currentStackMax, currentStackMin, pos)
			}
			have := int(metadata[section].outputs)
			if have >= maxOutputItems {
				return 0, fmt.Errorf("%w: at pos %d", ErrInvalidNonReturningFlag, pos)
			}
			if want := currentStackMin; have != want {
				return 0, fmt.Errorf("%w: have %d, want %d, at pos %d", ErrInvalidOutputs, have, want, pos)
			}
			qualifiedExit = true
		case JUMPF:
			arg, _ := parseUint16(code[pos+1:])
			newSection := metadata[arg]
			if have, limit := currentStackMax+int(newSection.maxStackHeight)-int(newSection.inputs), int(params.StackLimit); have > limit {
				return 0, fmt.Errorf("%w: at pos %d", ErrStackOverflow{stackLen: have, limit: limit}, pos)
			}
			if newSection.outputs == 0x80 {
				if want, have := int(newSection.inputs), currentStackMin; want > have {
					return 0, fmt.Errorf("%w: at pos %d", ErrStackUnderflow{stackLen: have, required: want}, pos)
				}
			} else {
				if currentStackMax != currentStackMin {
					return 0, fmt.Errorf("%w: max %d, min %d, at pos %d", ErrInvalidOutputs, currentStackMax, currentStackMin, pos)
				}
				if have, want := currentStackMax, int(metadata[section].outputs)+int(newSection.inputs)-int(newSection.outputs); have != want {
					return 0, fmt.Errorf("%w: at pos %d", ErrInvalidOutputs, pos)
				}
			}
			qualifiedExit = qualifiedExit || newSection.outputs < maxOutputItems
		case DUPN:
			arg := int(code[pos+1]) + 1
			if want, have := arg, currentStackMin; want > have {
				return 0, fmt.Errorf("%w: at pos %d", ErrStackUnderflow{stackLen: have, required: want}, pos)
			}
		case SWAPN:
			arg := int(code[pos+1]) + 1
			if want, have := arg+1, currentStackMin; want > have {
				return 0, fmt.Errorf("%w: at pos %d", ErrStackUnderflow{stackLen: have, required: want}, pos)
			}
		case EXCHANGE:
			arg := int(code[pos+1])
			n := arg>>4 + 1
			m := arg&0x0f + 1
			if want, have := n+m+1, currentStackMin; want > have {
				return 0, fmt.Errorf("%w: at pos %d", ErrStackUnderflow{stackLen: have, required: want}, pos)
			}
		default:
			if want, have := jt[op].minStack, currentStackMin; want > have {
				return 0, fmt.Errorf("%w: at pos %d", ErrStackUnderflow{stackLen: have, required: want}, pos)
			}
		}
		if !terminals[op] && op != CALLF {
			change := int(params.StackLimit) - jt[op].maxStack
			currentStackMax += change
			currentStackMin += change
		}
		next = next[:0]
		switch op {
		case RJUMP:
			nextPos := pos + 2 + parseInt16(code[pos+1:])
			next = append(next, nextPos)
			// We set the stack bounds of the destination
			// and skip the argument, only for RJUMP, all other opcodes are handled later
			if nextPos+1 < pos {
				ok, nextMin, nextMax := getStackMaxMin(nextPos + 1)
				if !ok {
					return 0, ErrInvalidBackwardJump
				}
				if nextMax != currentStackMax || nextMin != currentStackMin {
					return 0, ErrInvalidMaxStackHeight
				}
			} else {
				ok, nextMin, nextMax := getStackMaxMin(nextPos + 1)
				if !ok {
					setBounds(nextPos+1, currentStackMin, currentStackMax)
				} else {
					setBounds(nextPos+1, min(nextMin, currentStackMin), max(nextMax, currentStackMax))
				}
			}
		case RJUMPI:
			arg := parseInt16(code[pos+1:])
			next = append(next, pos+2)
			next = append(next, pos+2+arg)
		case RJUMPV:
			count := int(code[pos+1]) + 1
			next = append(next, pos+1+2*count)
			for i := 0; i < count; i++ {
				arg := parseInt16(code[pos+2+2*i:])
				next = append(next, pos+1+2*count+arg)
			}
		default:
			if imm := int(immediates[op]); imm != 0 {
				next = append(next, pos+imm)
			} else {
				// Simple op, no operand.
				next = append(next, pos)
			}
		}
		if debugging {
			fmt.Println(next)
		}

		if op != RJUMP && !terminals[op] {
			for _, instr := range next {
				nextPC := instr + 1
				if nextPC >= len(code) {
					return 0, fmt.Errorf("%w: end with %s, pos %d", ErrInvalidCodeTermination, op, pos)
				}
				if nextPC > pos {
					// target reached via forward jump or seq flow
					ok, nextMin, nextMax := getStackMaxMin(nextPC)
					if !ok {
						setBounds(nextPC, currentStackMin, currentStackMax)
					} else {
						setBounds(nextPC, min(nextMin, currentStackMin), max(nextMax, currentStackMax))
					}
				} else {
					// target reached via backwards jump
					ok, nextMin, nextMax := getStackMaxMin(nextPC)
					if !ok {
						return 0, ErrInvalidBackwardJump
					}
					if currentStackMax != nextMax {
						return 0, fmt.Errorf("%w want %d as current max got %d at pos %d,", ErrInvalidBackwardJump, currentStackMax, nextMax, pos)
					}
					if currentStackMin != nextMin {
						return 0, fmt.Errorf("%w want %d as current min got %d at pos %d,", ErrInvalidBackwardJump, currentStackMin, nextMin, pos)
					}
				}
			}
		}

		if op == RJUMP {
			pos += 2 // skip the immediate
		} else {
			pos = next[0]
		}
	}
	if qualifiedExit != (metadata[section].outputs < maxOutputItems) {
		return 0, fmt.Errorf("%w no RETF or qualified JUMPF", ErrInvalidNonReturningFlag)
	}
	if maxStackHeight >= int(params.StackLimit) {
		return 0, ErrStackOverflow{maxStackHeight, int(params.StackLimit)}
	}
	if maxStackHeight != int(metadata[section].maxStackHeight) {
		if debugging {
			fmt.Print(maxStackHeight, metadata[section].maxStackHeight)
		}
		return 0, fmt.Errorf("%w in code section %d: have %d, want %d", ErrInvalidMaxStackHeight, section, maxStackHeight, metadata[section].maxStackHeight)
	}
	return visitCount, nil
}
