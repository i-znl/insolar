// Code generated by "stringer -type=ActivationStatus"; DO NOT EDIT.

package conveyor

import "strconv"

const _ActivationStatus_name = "EmptyElementActiveElementNotActiveElement"

var _ActivationStatus_index = [...]uint8{0, 12, 25, 41}

func (i ActivationStatus) String() string {
	if i < 0 || i >= ActivationStatus(len(_ActivationStatus_index)-1) {
		return "ActivationStatus(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ActivationStatus_name[_ActivationStatus_index[i]:_ActivationStatus_index[i+1]]
}
