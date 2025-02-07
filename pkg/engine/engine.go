/*
 * Copyright (c) 2024, NVIDIA CORPORATION.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/NVIDIA/knavigator/pkg/config"
)

type Engine interface {
	RunTask(context.Context, *config.Task) error
	Reset(context.Context) error
}

type Eng struct {
	log           logr.Logger
	mutex         sync.Mutex
	k8sClient     *kubernetes.Clientset
	dynamicClient *dynamic.DynamicClient
	objMap        map[string]*ObjInfo
}

func New(log logr.Logger, config *rest.Config, sim ...bool) (*Eng, error) {
	eng := &Eng{
		log:    log,
		objMap: make(map[string]*ObjInfo),
	}

	if len(sim) == 0 { // len(sim) != 0 in unit tests
		var err error
		if eng.dynamicClient, err = dynamic.NewForConfig(config); err != nil {
			return nil, err
		}
		if eng.k8sClient, err = kubernetes.NewForConfig(config); err != nil {
			return nil, err
		}
	} else if sim[0] {
		eng.dynamicClient = &dynamic.DynamicClient{}
		eng.k8sClient = &kubernetes.Clientset{}
	}

	return eng, nil
}

func Run(ctx context.Context, eng Engine, testconfig *config.TaskConfig) error {
	var errExec error
	for _, cfg := range testconfig.Tasks {
		if errExec = eng.RunTask(ctx, cfg); errExec != nil {
			break
		}
	}

	errReset := eng.Reset(ctx)

	if errExec != nil {
		return errExec
	}

	return errReset
}

func (eng *Eng) RunTask(ctx context.Context, cfg *config.Task) error {
	runnable, err := eng.GetTask(cfg)
	if err != nil {
		return err
	}

	return execRunnable(ctx, eng.log, runnable)
}

// GetTask initializes and validates task
func (eng *Eng) GetTask(cfg *config.Task) (Runnable, error) {
	eng.mutex.Lock()
	defer eng.mutex.Unlock()

	eng.log.Info("Creating task", "name", cfg.Type, "id", cfg.ID)
	switch cfg.Type {
	case TaskSubmitObj:
		task, err := newSubmitObjTask(eng.log, eng.dynamicClient, eng, cfg)
		if err != nil {
			return nil, err
		}
		return task, nil
	case TaskUpdateObj:
		task, err := newUpdateObjTask(eng.log, eng.dynamicClient, eng, cfg)
		if err != nil {
			return nil, err
		}
		if _, ok := eng.objMap[task.RefTaskID]; !ok {
			return nil, fmt.Errorf("%s: unreferenced task ID %s", task.ID(), task.RefTaskID)
		}
		return task, nil
	case TaskCheckObj:
		task, err := newCheckObjTask(eng.log, eng.dynamicClient, eng, cfg)
		if err != nil {
			return nil, err
		}
		if _, ok := eng.objMap[task.RefTaskID]; !ok {
			return nil, fmt.Errorf("%s: unreferenced task ID %s", task.ID(), task.RefTaskID)
		}
		return task, nil
	case TaskDeleteObj:
		task, err := newDeleteObjTask(eng.log, eng.dynamicClient, eng, cfg)
		if err != nil {
			return nil, err
		}
		if _, ok := eng.objMap[task.RefTaskID]; !ok {
			return nil, fmt.Errorf("%s: unreferenced task ID %s", task.ID(), task.RefTaskID)
		}
		return task, nil
	case TaskUpdateNodes:
		return newUpdateNodesTask(eng.log, eng.k8sClient, cfg)
	case TaskCheckPod:
		task, err := newCheckPodTask(eng.log, eng.k8sClient, eng, cfg)
		if err != nil {
			return nil, err
		}
		if _, ok := eng.objMap[task.RefTaskID]; !ok {
			return nil, fmt.Errorf("%s: unreferenced task ID %s", task.ID(), task.RefTaskID)
		}
		return task, nil
	case TaskSleep:
		task, err := newSleepTask(eng.log, cfg)
		if err != nil {
			return nil, err
		}
		return task, nil

	case TaskPause:
		return newPauseTask(eng.log, cfg), nil

	default:
		return nil, fmt.Errorf("unsupported task type %q", cfg.Type)
	}
}

// SetObjInfo implements ObjSetter interface and maps task ID to the corresponding ObjInfo
func (eng *Eng) SetObjInfo(taskID string, info *ObjInfo) error {
	eng.mutex.Lock()
	defer eng.mutex.Unlock()

	if _, ok := eng.objMap[taskID]; ok {
		return fmt.Errorf("SetObjInfo: duplicate task ID %s", taskID)
	}

	eng.objMap[taskID] = info

	eng.log.V(4).Info("Setting task info", "taskID", taskID)

	return nil
}

// GetObjInfo implements ObjGetter interface returns ObjInfo for given task ID
func (eng *Eng) GetObjInfo(taskID string) (*ObjInfo, error) {
	eng.mutex.Lock()
	defer eng.mutex.Unlock()

	info, ok := eng.objMap[taskID]
	if !ok {
		return nil, fmt.Errorf("GetObjInfo: missing task ID %s", taskID)
	}

	eng.log.V(4).Info("Getting task info", "taskID", taskID)

	return info, nil
}

func execRunnable(ctx context.Context, log logr.Logger, r Runnable) error {
	id := r.ID()
	log.Info("Starting task", "id", id)
	start := time.Now()
	if err := r.Exec(ctx); err != nil {
		log.Error(err, "Task failed", "id", id)
		return err
	}
	log.Info("Task completed", "id", id, "duration", time.Since(start).String())
	return nil
}

func (eng *Eng) Reset(ctx context.Context) error {
	return nil
}
