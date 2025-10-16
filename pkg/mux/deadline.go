package mux

import "time"

// ControlOpDeadline bounds single control-channel operations (Encode/Decode).
// Default: 10s. Tests may shorten this for speed.
var ControlOpDeadline = 10 * time.Second
