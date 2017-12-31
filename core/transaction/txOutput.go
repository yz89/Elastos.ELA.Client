package transaction

import (
	"io"

	. "ELAClient/common"
	"ELAClient/common/serialization"
)

type TxOutput struct {
	AssetID     Uint256
	Value       Fixed64
	OutputLock  uint32
	ProgramHash Uint160
}

func (o *TxOutput) Serialize(w io.Writer) {
	o.AssetID.Serialize(w)
	o.Value.Serialize(w)
	serialization.WriteUint32(w, o.OutputLock)
	o.ProgramHash.Serialize(w)
}

func (o *TxOutput) Deserialize(r io.Reader) {
	o.AssetID.Deserialize(r)
	o.Value.Deserialize(r)
	temp, _ := serialization.ReadUint32(r)
	o.OutputLock = uint32(temp)
	o.ProgramHash.Deserialize(r)
}
