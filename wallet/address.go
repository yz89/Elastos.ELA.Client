package wallet

import (
	"github.com/elastos/Elastos.ELA/common"
)

const (
	TypeMaster = 0
	TypeStand  = 1 << 1
	TypeMulti  = 1 << 2
)

type Address struct {
	Address      string
	ProgramHash  *common.Uint168
	RedeemScript []byte
	Type         int
}

func (addr *Address) TypeName() string {
	switch addr.Type {
	case TypeMaster:
		return "MASTER"
	case TypeStand:
		return "STAND"
	case TypeMulti:
		return "MULTI"
	default:
		return ""
	}
}
