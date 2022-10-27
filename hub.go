package server_register

import (
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	ErrHostnameEmpty   = "error: node's hostname is empty"
	ErrTaskIdNoExisted = "error: task_id was not existed"
)

type Hub struct {
	nodeMu sync.RWMutex
	nodes  map[string]*Node

	taskMu       sync.RWMutex
	taskId       int32
	maxTaskCount int
	tasks        []*Task
	ret          map[string]*Return
}

func NewNode(hostname string) *Node {
	return &Node{
		Hostname: hostname,
		Status:   NodeOnline,
	}
}

func NewHub(max int) *Hub {
	return &Hub{
		nodes:        make(map[string]*Node, 0),
		taskId:       0,
		maxTaskCount: max,
		tasks:        make([]*Task, 0),
		ret:          make(map[string]*Return, 0),
	}
}

func (h *Hub) Register(hostname string) error {
	if hostname == "" {
		return fmt.Errorf(ErrHostnameEmpty)
	}
	h.nodeMu.Lock()
	defer h.nodeMu.Unlock()
	if _, ok := h.nodes[hostname]; !ok {
		h.nodes[hostname] = NewNode(hostname)
	}
	return nil
}

func (h *Hub) PullTask(hostname string) *Task {
	h.taskMu.RLock()
	defer h.taskMu.RUnlock()
	taskNum := len(h.tasks)
	if taskNum == 0 {
		return &Task{Id: 0, Command: make([]string, 0)}
	}
	if h.tasks[taskNum-1].Result[hostname] == "" {
		h.tasks[taskNum-1].Result[hostname] = "wait output"
	}
	return h.tasks[taskNum-1]
}

func (h *Hub) CompleteTask(hostname string, taskId int32, output string) error {
	fmt.Println("CompleteTask hostname:%s taskId:%d output:(%s)", hostname, taskId, output)
	h.taskMu.Lock()
	defer h.taskMu.Unlock()
	for i, v := range h.tasks {
		if v.Id == taskId {
			h.tasks[i].Result[hostname] = output
		}

		if taskId >= v.Id {
			h.ret[hostname] = &Return{
				TaskId: taskId,
				Output: output,
			}
			return nil
		}
	}
	return fmt.Errorf(ErrTaskIdNoExisted)
}

func (h *Hub) NewTask(commands []string) {
	h.taskMu.Lock()
	defer h.taskMu.Unlock()
	task := &Task{
		Id:      atomic.AddInt32(&h.taskId, 1),
		Command: commands,
		Result:  make(map[string]string, 0),
	}
	h.tasks = append(h.tasks, task)
}
