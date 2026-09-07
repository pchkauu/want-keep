package processor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	application "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
)

type Process struct {
	executable string
	workspaces *workspaces
	mutex      sync.Mutex
	blocked    bool
}
type processInput struct {
	Media string
	Data  []byte
}

func NewProcess() (*Process, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, domain.ErrUnavailable
	}
	workspaces, err := openWorkspaces(filepath.Join(os.TempDir(), "want-keep-documents"))
	if err != nil {
		return nil, err
	}
	return &Process{executable: path, workspaces: workspaces}, nil
}
func (p *Process) Inspect(ctx context.Context, media string, data []byte) (result application.Inspection, resultErr error) {
	if !p.mutex.TryLock() {
		return application.Inspection{}, domain.ErrUnavailable
	}
	defer p.mutex.Unlock()
	if p.workspaces == nil || p.blocked {
		return application.Inspection{}, domain.ErrUnavailable
	}
	if err := p.workspaces.recover(); err != nil {
		return application.Inspection{}, err
	}
	name, err := p.workspaces.create()
	if err != nil {
		return application.Inspection{}, err
	}
	defer func() {
		if !p.blocked {
			if err := p.workspaces.remove(name); err != nil {
				result = application.Inspection{}
				resultErr = err
			}
		}
	}()
	input, err := json.Marshal(processInput{media, data})
	if err != nil {
		return result, domain.ErrUnavailable
	}
	defer clear(input)
	cmd := exec.CommandContext(ctx, p.executable, "inspect")
	cmd.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "LANG=C", "LC_ALL=C", "HOME=/nonexistent", "TMPDIR=" + p.workspaces.path(name)}
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stderr = io.Discard
	cmd.WaitDelay = 2 * time.Second
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
	runErr := cmd.Run()
	// A parser crash can leave descendants alive. Stop its remaining group before cleanup.
	if cmd.Process != nil && p.stopGroup(cmd.Process.Pid) != nil {
		p.blocked = true
		return result, domain.ErrUnavailable
	}
	if runErr != nil {
		return result, domain.ErrUnavailable
	}
	if json.Unmarshal(out.Bytes(), &result) != nil {
		return application.Inspection{}, domain.ErrUnavailable
	}
	return result, nil
}

// The deployment's init process reaps orphaned PDF tools. Wait for the group
// to disappear before reclaiming paths which those tools could still write.
func (p *Process) stopGroup(pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGKILL); errors.Is(err, syscall.ESRCH) {
		return nil
	} else if err != nil {
		return domain.ErrUnavailable
	}
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := syscall.Kill(-pid, 0); errors.Is(err, syscall.ESRCH) {
			return nil
		} else if err != nil {
			return domain.ErrUnavailable
		}
		select {
		case <-deadline.C:
			return domain.ErrUnavailable
		case <-tick.C:
		}
	}
}
func (p *Process) Close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.workspaces != nil {
		p.workspaces.close()
		p.workspaces = nil
	}
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

func (p *Process) Ready(ctx context.Context) bool {
	if ctx.Err() != nil {
		return false
	}
	if !p.mutex.TryLock() {
		return true
	}
	defer p.mutex.Unlock()
	return p.workspaces != nil && !p.blocked
}
