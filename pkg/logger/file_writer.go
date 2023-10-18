package logger

import (
	"io"
	"sync"
)

type fileWriter struct {
	w  io.Writer
	mu *sync.Mutex
}

func InitPluginWriter(agentId string) io.Writer {
	sharedPluginWriter.mu.Lock()
	defer sharedPluginWriter.mu.Unlock()
	if sharedPluginWriter.w != nil {
		return sharedPluginWriter.w
	}

	f := WriteOnlyFile(agentId)
	writer := f.(io.Writer)
	sharedPluginWriter.w = writer
	return writer
}

func (fw fileWriter) Write(b []byte) (int, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if fw.w == nil {
		return 0, nil
	}

	n, err := fw.w.Write(b)
	if err != nil {
		return n, err
	}

	return n, nil
}
