package downloader

import "context"

type Request struct {
	URL, Directory, FileName, Referer, UserAgent, Proxy string
	Connections                                         int
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
