package ui

import (
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
    "golang.design/x/clipboard"
)

/* ===============================
   TEXT FIELD
   =============================== */

type TextField struct {
	Value     string
	Numeric   bool
	ReadOnly  bool

	CursorPos int
	CaretTicks int
    
    BackspaceTicks int
	DeleteTicks    int
    
    LeftTicks  int
	RightTicks int
}

/* ===============================
   UPDATE
   =============================== */

// Update processes keyboard input for the text field.
// Call this ONLY when the field is focused.
func (tf *TextField) Update() {
	tf.CaretTicks++

	// Clipboard

    ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) ||
		ebiten.IsKeyPressed(ebiten.KeyControlLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyControlRight)

	// Ctrl+C copy
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyC) {
		if tf.Value != "" {
			clipboard.Write(clipboard.FmtText, []byte(tf.Value))
		}
		return
	}

    // Read-only guard
	if tf.ReadOnly {
		return
	}

    // Clipboard

	// Ctrl+V paste
	if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyV) {
		data := clipboard.Read(clipboard.FmtText)
		if len(data) == 0 {
			return
		}

		text := string(data)

		// Enforce numeric-only fields
		if tf.Numeric {
			filtered := ""
			for i, r := range text {
				if unicode.IsDigit(r) || (r == '-' && i == 0) {
					filtered += string(r)
				}
			}
			text = filtered
		}

		tf.Value = text
		tf.CursorPos = len(tf.Value)
		tf.CaretTicks = 0
		return
	}

	// Text input

	for _, r := range ebiten.AppendInputChars(nil) {
		if !unicode.IsPrint(r) {
			continue
		}
		if tf.Numeric && !unicode.IsDigit(r) && r != '-' {
			continue
		}
		tf.InsertRune(r)
	}

	// Deletion keys (backspace + delete)
	const (
        initialDelay = 15 // frames before repeat starts
        repeatRate   = 3  // frames between repeats
    )

    // Backspace repeat

    if ebiten.IsKeyPressed(ebiten.KeyBackspace) {
        tf.BackspaceTicks++

        if tf.BackspaceTicks == 1 ||
            (tf.BackspaceTicks > initialDelay &&
                (tf.BackspaceTicks-initialDelay)%repeatRate == 0) {
            tf.Backspace()
        }
    } else {
        tf.BackspaceTicks = 0
    }

    // Delete repeat

    if ebiten.IsKeyPressed(ebiten.KeyDelete) {
        tf.DeleteTicks++

        if tf.DeleteTicks == 1 ||
            (tf.DeleteTicks > initialDelay &&
                (tf.DeleteTicks-initialDelay)%repeatRate == 0) {
            tf.Delete()
        }
    } else {
        tf.DeleteTicks = 0
    }

	// Cursor movement

    if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
        tf.LeftTicks++

        if tf.LeftTicks == 1 ||
            (tf.LeftTicks > initialDelay &&
                (tf.LeftTicks-initialDelay)%repeatRate == 0) {
            tf.MoveLeft()
        }
    } else {
        tf.LeftTicks = 0
    }

    if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
        tf.RightTicks++

        if tf.RightTicks == 1 ||
            (tf.RightTicks > initialDelay &&
                (tf.RightTicks-initialDelay)%repeatRate == 0) {
            tf.MoveRight()
        }
    } else {
        tf.RightTicks = 0
    }

	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		tf.MoveHome()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		tf.MoveEnd()
	}
}


/* ===============================
   EDIT OPERATIONS
   =============================== */

func (tf *TextField) InsertRune(r rune) {
	before := tf.Value[:tf.CursorPos]
	after := tf.Value[tf.CursorPos:]
	tf.Value = before + string(r) + after
	tf.CursorPos++
	tf.CaretTicks = 0
}

func (tf *TextField) Backspace() {
	if tf.CursorPos > 0 {
		tf.Value = tf.Value[:tf.CursorPos-1] + tf.Value[tf.CursorPos:]
		tf.CursorPos--
		tf.CaretTicks = 0
	}
}

func (tf *TextField) Delete() {
	if tf.CursorPos < len(tf.Value) {
		tf.Value = tf.Value[:tf.CursorPos] + tf.Value[tf.CursorPos+1:]
		tf.CaretTicks = 0
	}
}

/* ===============================
   CURSOR MOVEMENT
   =============================== */

func (tf *TextField) MoveLeft() {
	if tf.CursorPos > 0 {
		tf.CursorPos--
		tf.CaretTicks = 0
	}
}

func (tf *TextField) MoveRight() {
	if tf.CursorPos < len(tf.Value) {
		tf.CursorPos++
		tf.CaretTicks = 0
	}
}

func (tf *TextField) MoveHome() {
	tf.CursorPos = 0
	tf.CaretTicks = 0
}

func (tf *TextField) MoveEnd() {
	tf.CursorPos = len(tf.Value)
	tf.CaretTicks = 0
}

/* ===============================
   MOUSE SUPPORT
   =============================== */

// SetCursorFromMouse sets the caret based on mouse X position.
// charWidth is typically 7 for basicfont.Face7x13.
func (tf *TextField) SetCursorFromMouse(mx, valueX, charWidth int) {
	if mx <= valueX {
		tf.CursorPos = 0
	} else {
		tf.CursorPos = (mx - valueX) / charWidth
	}

	if tf.CursorPos < 0 {
		tf.CursorPos = 0
	}
	if tf.CursorPos > len(tf.Value) {
		tf.CursorPos = len(tf.Value)
	}

	tf.CaretTicks = 0
}

/* ===============================
   CARET VISIBILITY
   =============================== */

func (tf *TextField) CaretVisible() bool {
	if tf.ReadOnly {
		return false
	}
	return (tf.CaretTicks/30)%2 == 0
}

/* ===============================
   UTIL
   =============================== */

func (tf *TextField) SetValue(v string) {
	tf.Value = v
	tf.CursorPos = len(v)
	tf.CaretTicks = 0
}

func (tf *TextField) Clear() {
	tf.Value = ""
	tf.CursorPos = 0
	tf.CaretTicks = 0
}
