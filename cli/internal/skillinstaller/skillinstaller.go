/*
 * Copyright 2026 Clidey, Inc.
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

// Package skillinstaller installs the bundled WhoDB assistant skills and agents.
package skillinstaller

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	whodbplugin "github.com/clidey/whodb/cli/external-plugin/whodb"
)

// Item describes one bundled skill or agent.
type Item struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// InstallOptions configures a skill installation.
type InstallOptions struct {
	Name          string
	Target        string
	TargetDir     string
	AgentsDir     string
	IncludeAgents bool
	Force         bool
}

// InstallResult describes files written by an install operation.
type InstallResult struct {
	Skills []InstalledFile `json:"skills"`
	Agents []InstalledFile `json:"agents,omitempty"`
}

// InstalledFile records one installed asset.
type InstalledFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// List returns all bundled skills and agents.
func List() ([]Item, error) {
	items := make([]Item, 0)

	skills, err := skillNames()
	if err != nil {
		return nil, err
	}
	for _, name := range skills {
		description, _ := readSkillDescription(name)
		items = append(items, Item{Name: name, Type: "skill", Description: description})
	}

	agents, err := agentNames()
	if err != nil {
		return nil, err
	}
	for _, name := range agents {
		items = append(items, Item{Name: name, Type: "agent"})
	}

	slices.SortFunc(items, func(a, b Item) int {
		if a.Type != b.Type {
			return strings.Compare(a.Type, b.Type)
		}
		return strings.Compare(a.Name, b.Name)
	})
	return items, nil
}

// Install copies bundled skills and optional agents to the selected target.
func Install(opts InstallOptions) (InstallResult, error) {
	targetDir, agentsDir, err := resolveTargetDirs(opts)
	if err != nil {
		return InstallResult{}, err
	}
	if opts.IncludeAgents && strings.TrimSpace(agentsDir) == "" {
		return InstallResult{}, fmt.Errorf("--include-agents requires --target claude-code or --agents-dir")
	}

	names, err := selectedSkillNames(opts.Name)
	if err != nil {
		return InstallResult{}, err
	}

	result := InstallResult{Skills: make([]InstalledFile, 0, len(names))}
	for _, name := range names {
		path, err := installSkill(name, targetDir, opts.Force)
		if err != nil {
			return result, err
		}
		result.Skills = append(result.Skills, InstalledFile{Name: name, Path: path})
	}

	if opts.IncludeAgents {
		agents, err := agentNames()
		if err != nil {
			return result, err
		}
		for _, name := range agents {
			path, err := installAgent(name, agentsDir, opts.Force)
			if err != nil {
				return result, err
			}
			result.Agents = append(result.Agents, InstalledFile{Name: name, Path: path})
		}
	}

	return result, nil
}

func resolveTargetDirs(opts InstallOptions) (string, string, error) {
	if strings.TrimSpace(opts.TargetDir) != "" {
		agentsDir := opts.AgentsDir
		if opts.IncludeAgents && strings.TrimSpace(agentsDir) == "" {
			return "", "", fmt.Errorf("--agents-dir is required when --include-agents is used with --target-dir")
		}
		return opts.TargetDir, agentsDir, nil
	}

	target := strings.TrimSpace(opts.Target)
	if target == "" {
		return "", "", fmt.Errorf("provide --target or --target-dir")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}

	switch target {
	case "codex":
		return filepath.Join(home, ".agents", "skills"), "", nil
	case "claude-code":
		return filepath.Join(home, ".claude", "skills"), filepath.Join(home, ".claude", "agents"), nil
	default:
		return "", "", fmt.Errorf("unsupported target %q", target)
	}
}

func selectedSkillNames(name string) ([]string, error) {
	all, err := skillNames()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" || name == "all" {
		return all, nil
	}
	for _, candidate := range all {
		if candidate == name {
			return []string{name}, nil
		}
	}
	return nil, fmt.Errorf("skill %q not found", name)
}

func skillNames() ([]string, error) {
	entries, err := fs.ReadDir(whodbplugin.FS, "skills")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	return names, nil
}

func agentNames() ([]string, error) {
	entries, err := fs.ReadDir(whodbplugin.FS, "agents")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			names = append(names, strings.TrimSuffix(entry.Name(), ".md"))
		}
	}
	slices.Sort(names)
	return names, nil
}

func installSkill(name, targetDir string, force bool) (string, error) {
	sourcePath := filepath.ToSlash(filepath.Join("skills", name, "SKILL.md"))
	data, err := whodbplugin.FS.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}

	destPath := filepath.Join(targetDir, name, "SKILL.md")
	if err := writeFile(destPath, data, force); err != nil {
		return "", err
	}
	return destPath, nil
}

func installAgent(name, targetDir string, force bool) (string, error) {
	sourcePath := filepath.ToSlash(filepath.Join("agents", name+".md"))
	data, err := whodbplugin.FS.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}

	destPath := filepath.Join(targetDir, name+".md")
	if err := writeFile(destPath, data, force); err != nil {
		return "", err
	}
	return destPath, nil
}

func writeFile(path string, data []byte, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists; use --force to overwrite", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readSkillDescription(name string) (string, error) {
	path := filepath.ToSlash(filepath.Join("skills", name, "SKILL.md"))
	data, err := whodbplugin.FS.ReadFile(path)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "description:") {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "description:")), `"`), nil
		}
	}
	return "", nil
}
