package server_register

import "time"

type NodeStatus int32

const (
	NodeOnline  NodeStatus = iota
	NodeOffline NodeStatus = 1
	NodeRemoved NodeStatus = 2
)

type Node struct {
	Hostname string     `json:"hostname"`
	Status   NodeStatus `json:"status"`
	Time     time.Time  `json:"time"`
}

type Task struct {
	Id      int32             `json:"id"`
	Command []string          `json:"command"`
	Result  map[string]string `json:"result"`
}

type Return struct {
	TaskId int32  `json:"task_id"`
	Output string `json:"output"`
}
