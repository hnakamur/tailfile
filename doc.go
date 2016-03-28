// tailfile package provides a way to reading a log file line by line like the BSD tail program.
// The target file may not exist at the first.  However the directory for the target file must exist all the time.
// This package detects the target file is created, renamed or deleted.
//
// tailfile package assumes the following log file lifecycle.
//
//   1. The target log file is created.
//   2. The log lines are written to the log file.
//   3. The log file are renamed for the log rotation.
//   4. More logs may be written to the renamed file for a while.
//   5. The new log file is created with the original filename.
//   6. The logs are written to the newly created log file. Once this happens logs are never written to the renamed log file.
//
// The other scenarios are not supported. For example, the log file are renamed and then renamed back to the original filename.
//
// Also when you kill the process running this package while reading the renamed log file, it cannot continue reading the
// rest logs in the renamed log file.
//
// This limitation comes from the package http://gopkg.in/fsnotify.v1 which is used to the detect filesystem change.
// It can detect a file is renamed but it does not tell you the new filename.
//
// In the same reason, reading the renamed log file uses the polling as opposed to reading the original log file uses the
// file write event.
//
// See cmd/exampletail/main.go for an example. Run this example alongside cmd/examplewriter/main.go.
package tailfile
