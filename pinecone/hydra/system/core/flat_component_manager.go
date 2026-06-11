package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/system/arch"
)

type FlatComponentManager struct {
	components []arch.Component
	index      map[string]arch.Component
}

var _ arch.ComponentManager = (*FlatComponentManager)(nil)

func NewFlatComponentManager() *FlatComponentManager {
	return &FlatComponentManager{
		index: map[string]arch.Component{},
	}
}

func (m *FlatComponentManager) Register(component arch.Component) error {
	if component == nil {
		return fmt.Errorf("component is nil")
	}
	name := strings.TrimSpace(component.Name())
	if name == "" {
		return fmt.Errorf("component name is empty")
	}
	if m.index == nil {
		m.index = map[string]arch.Component{}
	}
	if _, exists := m.index[name]; exists {
		return fmt.Errorf("component already registered: %s", name)
	}
	m.components = append(m.components, component)
	m.index[name] = component
	return nil
}

func (m *FlatComponentManager) Get(name string) (arch.Component, bool) {
	if m == nil || m.index == nil {
		return nil, false
	}
	component, ok := m.index[strings.TrimSpace(name)]
	return component, ok
}

func (m *FlatComponentManager) List() []arch.Component {
	if m == nil || len(m.components) == 0 {
		return nil
	}
	components := make([]arch.Component, len(m.components))
	copy(components, m.components)
	return components
}

func (m *FlatComponentManager) VitalizeAll(ctx context.Context) error {
	if m == nil {
		return nil
	}
	for _, component := range m.components {
		if err := component.Vitalize(ctx); err != nil {
			return fmt.Errorf("vitalize component %q: %w", component.Name(), err)
		}
	}
	return nil
}

func (m *FlatComponentManager) ShutdownAll(ctx context.Context) error {
	if m == nil {
		return nil
	}
	var firstErr error
	for index := len(m.components) - 1; index >= 0; index-- {
		component := m.components[index]
		if err := component.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("shutdown component %q: %w", component.Name(), err)
		}
	}
	return firstErr
}
