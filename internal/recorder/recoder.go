package recorder

import (
	"bytes"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

type Recorder struct {
	buf bytes.Buffer
	cmd *exec.Cmd
}

func NewRecorder() *Recorder {
	return &Recorder{}
}

func (r *Recorder) Start() {
	r.buf.Reset()
	r.cmd = exec.Command("sox", "-d", "-t", "wav", "-")
	r.cmd.Stdout = &r.buf

	err := r.cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
}

// Stop recording
func (r *Recorder) Stop() {
	err := r.cmd.Process.Signal(os.Interrupt)
	if err != nil {
		log.Fatal(err)
	}

	// Wait for the recording process to finish
	err = r.cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}

	log.Debugf("Recording stopped. recorded %d bytes", r.buf.Len())
}

func (r *Recorder) Buffer() *bytes.Buffer {
	return &r.buf
}
