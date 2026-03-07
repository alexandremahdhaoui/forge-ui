// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !js || !wasm

package controller

import (
	"os"
	"path/filepath"

	"github.com/alexandremahdhaoui/forge-ui/internal/adapter"
	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

// CUService provides CU composition visualization.
type CUService interface {
	GetCompoState(wsPath string) (types.CompoState, error)
}

// Compile-time check.
var _ CUService = (*cuService)(nil)

type cuService struct {
	cuAdapter adapter.CUAdapter
}

// NewCUService creates a CUService.
func NewCUService(cuAdapter adapter.CUAdapter) CUService {
	return &cuService{cuAdapter: cuAdapter}
}

func (s *cuService) GetCompoState(wsPath string) (types.CompoState, error) {
	cuRepoPath := filepath.Join(wsPath, ".cu-repo")
	if _, err := os.Stat(cuRepoPath); os.IsNotExist(err) {
		return types.CompoState{}, nil
	}

	return s.cuAdapter.LoadCompo(cuRepoPath)
}
