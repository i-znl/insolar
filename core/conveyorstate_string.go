// Code generated by "stringer -type=ConveyorState"; DO NOT EDIT.

package core

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ConveyorActive-0]
	_ = x[ConveyorPreparingPulse-1]
	_ = x[ConveyorShuttingDown-2]
	_ = x[ConveyorInactive-3]
}

const _ConveyorState_name = "ConveyorActiveConveyorPreparingPulseConveyorShuttingDownConveyorInactive"

var _ConveyorState_index = [...]uint8{0, 14, 36, 56, 72}

func (i ConveyorState) String() string {
	if i < 0 || i >= ConveyorState(len(_ConveyorState_index)-1) {
		return "ConveyorState(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ConveyorState_name[_ConveyorState_index[i]:_ConveyorState_index[i+1]]
}