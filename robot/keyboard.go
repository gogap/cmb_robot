package robot

import (
	"github.com/AllenDang/w32"
	"time"
)

func TapKey(keys ...uint16) {
	var onInput []w32.INPUT

	for i := 0; i < len(keys); i++ {
		input := w32.INPUT{
			Type: w32.INPUT_KEYBOARD,
			Ki: w32.KEYBDINPUT{
				WVk: keys[i],
			},
		}
		onInput = append(onInput, input)
	}

	w32.SendInput(onInput)

	time.Sleep(time.Millisecond * 100)

	var offInput []w32.INPUT

	for i := len(keys) - 1; i >= 0; i-- {
		input := w32.INPUT{
			Type: w32.INPUT_KEYBOARD,
			Ki: w32.KEYBDINPUT{
				WVk:     keys[i],
				DwFlags: w32.KEYEVENTF_KEYUP,
			},
		}
		offInput = append(offInput, input)
	}

	w32.SendInput(offInput)
}
