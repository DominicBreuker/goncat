package mux

import "time"

// ControlOpDeadline is the deadline used for single control-channel operations
// (gob Encode/Decode). Default is 10s in normal runs, but tests can shorten
// this by setting the variable to a smaller duration.
var ControlOpDeadline = 10 * time.Second
