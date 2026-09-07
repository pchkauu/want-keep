package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"

	application "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
)

type Process struct{ executable string }
type processInput struct {
	Media string
	Data  []byte
}

func NewProcess() (*Process, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, domain.ErrUnavailable
	}
	return &Process{executable: path}, nil
}
func (p *Process) Inspect(ctx context.Context, media string, data []byte) (application.Inspection, error) {
	var result application.Inspection
	input, err := json.Marshal(processInput{media, data})
	if err != nil {
		return result, domain.ErrUnavailable
	}
	defer clear(input)
	cmd := exec.CommandContext(ctx, p.executable, "inspect")
	cmd.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "LANG=C", "LC_ALL=C", "HOME=/nonexistent"}
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stderr = io.Discard
	out := &boundedOutput{maximum: MaxResponseBytes}
	cmd.Stdout = out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// The parser and its PDF children share a process group. A deadline must stop all of them.
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	if err = cmd.Run(); err != nil {
		return result, domain.ErrUnavailable
	}
	if json.Unmarshal(out.Bytes(), &result) != nil {
		return application.Inspection{}, domain.ErrUnavailable
	}
	return result, nil
}
func (e *Engine) InspectStream(ctx context.Context, input io.Reader, output io.Writer) error {
	data, err := io.ReadAll(io.LimitReader(input, 14*1024*1024+1))
	if err != nil || len(data) > 14*1024*1024 {
		return domain.ErrInvalid
	}
	defer clear(data)
	var request processInput
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil || decoder.Decode(new(any)) != io.EOF {
		return domain.ErrInvalid
	}
	defer clear(request.Data)
	result, err := e.Inspect(ctx, request.Media, request.Data)
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(result)
}
