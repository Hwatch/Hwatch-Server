package downloader

import (
	"context"
	"errors"
)

var ErrCapacityReached = errors.New("downloader capacity reached")

type Request struct {
	URL, Directory, FileName, Referer, UserAgent, Proxy, ExternalID string
	Connections                                                     int
}

type Status struct {
	ExternalID, State, ErrorCode, ErrorMessage, Directory string
	Completed, Total, DownloadSpeed                       int64
	Connections                                           int
	Files                                                 []string
}

type Health struct {
	ID, Name, Version, Error string
	Online, SupportsMulti    bool
	Active, Connections      int
	DownloadSpeed            int64
	UptimeSeconds            int64
}

type Engine interface {
	ID() string
	Add(context.Context, Request) (string, error)
	Tell(context.Context, string) (Status, error)
	Pause(context.Context, string) error
	Resume(context.Context, string) error
	Remove(context.Context, string) error
	PauseAll(context.Context) error
	ResumeAll(context.Context) error
	SetGlobal(context.Context, int, int64) error
	Health(context.Context) Health
}

type CapacityWaiter interface {
	WaitForCapacity(context.Context) error
}

// ResultCleaner removes downloader-side history after HWatch has durably
// recorded a terminal state. Active transfer data is never removed.
type ResultCleaner interface {
	CleanResult(context.Context, string) error
}
