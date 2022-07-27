package main

import "time"

type SourceType string

const (
	HOSTS SourceType = "hosts"
	REG   SourceType = "regex"
	WILD  SourceType = "wild"
	VOID  SourceType = "void"
	DIR   SourceType = "direct"
	LIST  SourceType = "list"
)

type LocationType string

const (
	LOC LocationType = "local"
	REM LocationType = "remote"
)

type Location struct {
	Path string
	Type LocationType
}

type Source struct {
	Location Location
	Type     SourceType
	Sync     *time.Duration
	Category string
	Tags     []string
}

type Sources []Source

func (s Sources) Records() []*Record {
	//records := make([]*Record)

	//for _, src := range s {
	//	switch src.Type {
	//	case HOSTS:
	//		return src.Hosts()
	//	case REG:
	//		return src.Reg()
	//	case WILD:
	//		return src.Wild()
	//	case VOID:
	//		return src.Void()
	//	case DIR:
	//		return src.Dir()
	//	case LIST:
	//		return src.List()
	//	}
	//}

	return nil
}
